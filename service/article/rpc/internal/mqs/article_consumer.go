package mqs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-queue/kq"
	"io"
	"net/url"
	"regexp"
	"sea-try-go/service/article/common/errmsg"
	"sea-try-go/service/article/rpc/internal/svc"
	pb "sea-try-go/service/article/rpc/pb"
	"sea-try-go/service/common/logger"
	imagesecurity "sea-try-go/service/security/rpc/client/imagesecurityservice"
	security "sea-try-go/service/security/rpc/pb/sea-try-go/service/security/rpc/pb"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/zeromicro/go-zero/core/logx"
)

type ArticleConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type ArticleHotEvent struct {
	ArticleID  string `json:"article_id"`
	ArticleTag string `json:"article_tag"`
	Content    string `json:"content"`
	CoverUrl   string `json:"cover_url"`
}

func NewArticleConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ArticleConsumer {
	return &ArticleConsumer{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ArticleConsumer) Consume(ctx context.Context, key, val string) error {
	logger.LogInfo(ctx, fmt.Sprintf("DataClean Service Consuming: %s", val))

	var msg struct {
		ArticleId   string `json:"article_id"`
		AuthorId    string `json:"author_id"`
		ContentPath string `json:"content_path"`
	}

	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, fmt.Errorf("unmarshal error: %w", err))
		return nil
	}

	object, err := l.svcCtx.MinioClient.GetObject(ctx, l.svcCtx.Config.MinIO.BucketName, msg.ContentPath, minio.GetObjectOptions{})
	if err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorMinioDownload, fmt.Errorf("failed to get content from minio: %w", err), logger.WithArticleID(msg.ArticleId))
		return err
	}
	defer object.Close()

	contentBytes, err := io.ReadAll(object)
	if err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorMinioDownload, fmt.Errorf("failed to read minio content: %w", err), logger.WithArticleID(msg.ArticleId))
		return err
	}
	articleContent := string(contentBytes)

	article, err := l.svcCtx.ArticleRepo.FindOne(ctx, msg.ArticleId)
	if err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorDbSelect, fmt.Errorf("failed to find article %s: %w", msg.ArticleId, err), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
		return err
	}

	if article.Status != int32(pb.ArticleStatus_REVIEWING) {
		logger.LogInfo(ctx, fmt.Sprintf("Article %s status is %d, skipping duplicate processing.", msg.ArticleId, article.Status))
		return nil
	}

	if l.svcCtx.SecurityRpc == nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, fmt.Errorf("ContentSecurity client is not initialized"), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
		return fmt.Errorf("ContentSecurity client not initialized")
	}

	result, err := l.svcCtx.SecurityRpc.SanitizeContent(ctx, &security.SanitizeContentRequest{
		Text: articleContent,
		Options: &security.SanitizeOptions{
			EnableAdDetection:             true,
			EnableHtmlSanitization:        true,
			EnableUnicodeNormalization:    true,
			EnableWhitespaceNormalization: true,
		},
	})
	if err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, fmt.Errorf("ContentSecurity RPC error: %w", err), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
		return err
	}

	if !result.Success {
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, fmt.Errorf("ContentSecurity service failed: %s", result.ErrorMessage), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
		return fmt.Errorf("ContentSecurity service error: %s", result.ErrorMessage)
	}

	if result.IsAd {
		logger.LogInfo(ctx, fmt.Sprintf("Article %s RISK DETECTED (Text Ad)! Confidence: %f", msg.ArticleId, result.AdConfidence), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
		article.Status = int32(pb.ArticleStatus_REJECTED)
		if err := l.svcCtx.ArticleRepo.Update(ctx, article); err != nil {
			logger.LogBusinessErr(ctx, errmsg.ErrorDbUpdate, fmt.Errorf("failed to update article status to Rejected: %w", err), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
			return err
		}
		return nil
	}

	imageUrls := l.extractImageUrls(articleContent)
	if article.CoverImageURL != "" {
		imageUrls = append(imageUrls, article.CoverImageURL)
	}

	for _, imgUrl := range imageUrls {
		isAd, confidence, err := l.auditImage(ctx, imgUrl)
		if err != nil {
			logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, fmt.Errorf("audit image %s failed: %w", imgUrl, err), logger.WithArticleID(msg.ArticleId))
			continue
		}

		if isAd {
			logger.LogInfo(ctx, fmt.Sprintf("Article %s RISK DETECTED (Image Ad: %s)! Confidence: %f", msg.ArticleId, imgUrl, confidence), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
			article.Status = int32(pb.ArticleStatus_REJECTED)
			if err := l.svcCtx.ArticleRepo.Update(ctx, article); err != nil {
				logger.LogBusinessErr(ctx, errmsg.ErrorDbUpdate, fmt.Errorf("failed to update article status to Rejected: %w", err), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
				return err
			}
			return nil
		}
	}

	logger.LogInfo(ctx, fmt.Sprintf("Article %s passed safety check.", msg.ArticleId), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))

	article.Status = int32(pb.ArticleStatus_PUBLISHED)
	if err := l.svcCtx.ArticleRepo.Update(ctx, article); err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorDbUpdate, fmt.Errorf("failed to update article status to Published: %w", err), logger.WithArticleID(msg.ArticleId), logger.WithUserID(msg.AuthorId))
		return err
	}

	if err := PushHotEvent(ctx, l.svcCtx.HotEventPusher, article.ID, article.ManualTypeTag, articleContent, article.CoverImageURL); err != nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, fmt.Errorf("failed to push hot event for article %s: %w", article.ID, err), logger.WithArticleID(article.ID))
	}

	return nil
}

func (l *ArticleConsumer) extractImageUrls(content string) []string {
	// 匹配 Markdown 格式: ![alt](url)
	re := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	matches := re.FindAllStringSubmatch(content, -1)
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			urls = append(urls, match[1])
		}
	}
	return urls
}

func (l *ArticleConsumer) auditImage(ctx context.Context, imgUrl string) (bool, float32, error) {
	u, err := url.Parse(imgUrl)
	if err != nil {
		return false, 0, err
	}
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return false, 0, fmt.Errorf("invalid image path: %s", path)
	}
	objectName := parts[1]

	object, err := l.svcCtx.MinioClient.GetObject(ctx, l.svcCtx.Config.MinIO.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return false, 0, err
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return false, 0, err
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	mimeType := "image/jpeg"
	lower := strings.ToLower(objectName)
	if strings.HasSuffix(lower, ".png") {
		mimeType = "image/png"
	} else if strings.HasSuffix(lower, ".webp") {
		mimeType = "image/webp"
	}
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	resp, err := l.svcCtx.ImageSecurityRpc.DetectImageAd(ctx, &imagesecurity.DetectImageAdRequest{
		ImageBase64: dataURI,
		Options: &imagesecurity.DetectOptions{
			ConfidenceThreshold:  0.7,
			EnableTextExtraction: true,
		},
	})
	if err != nil {
		return false, 0, err
	}

	return resp.IsAd, resp.AdConfidence, nil
}

func PushHotEvent(ctx context.Context, pusher *kq.Pusher, articleID string, articleTag string, content string, coverUrl string) error {
	payload, err := json.Marshal(ArticleHotEvent{
		ArticleID:  articleID,
		ArticleTag: articleTag,
		Content:    content,
		CoverUrl:   coverUrl,
	})
	if err != nil {
		return err
	}
	return pusher.Push(ctx, string(payload))
}
