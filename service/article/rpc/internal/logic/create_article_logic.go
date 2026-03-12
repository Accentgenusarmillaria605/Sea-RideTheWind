package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zeromicro/go-zero/core/logx"
	"sea-try-go/service/article/common/errmsg"
	"sea-try-go/service/article/rpc/internal/model"
	"sea-try-go/service/article/rpc/internal/svc"
	"sea-try-go/service/article/rpc/metrics"
	"sea-try-go/service/article/rpc/pb"
	"sea-try-go/service/common/logger"
	"sea-try-go/service/common/snowflake"
	"strings"

	"github.com/minio/minio-go/v7"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreateArticleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateArticleLogic {
	return &CreateArticleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateArticleLogic) CreateArticle(in *__.CreateArticleRequest) (*__.CreateArticleResponse, error) {
	tracer := otel.Tracer("article-rpc")
	ctx, span := tracer.Start(l.ctx, "CreateArticle", trace.WithAttributes(
		attribute.String("author_id", in.AuthorId),
		attribute.String("title", in.Title),
	))
	defer span.End()

	idInt, err := snowflake.GetID()
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	articleId := fmt.Sprintf("%d", idInt)
	span.SetAttributes(attribute.String("article_id", articleId))

	objectName := fmt.Sprintf("%s%s.md", l.svcCtx.Config.MinIO.ArticlePath, articleId)
	contentType := "text/markdown"
	reader := strings.NewReader(in.MarkdownContent)

	//统计 MinIO put 操作耗时
	timer := prometheus.NewTimer(metrics.MinioRequestDuration.WithLabelValues("put"))
	span.AddEvent("start upload to minio")
	_, err = l.svcCtx.MinioClient.PutObject(ctx, l.svcCtx.Config.MinIO.BucketName, objectName,
		reader, int64(len(in.MarkdownContent)), minio.PutObjectOptions{ContentType: contentType})
	timer.ObserveDuration()

	if err != nil {
		span.RecordError(err)
		//统计 MinIO put 操作失败数
		metrics.MinioRequestErrors.WithLabelValues("put").Inc()
		logger.LogBusinessErr(ctx, errmsg.ErrorMinioUpload, fmt.Errorf("upload to minio failed: %w", err), logger.WithArticleID(articleId), logger.WithUserID(in.AuthorId))
		return nil, err
	}
	span.AddEvent("upload to minio success")
	//统计 markdown 文件上传总数
	metrics.FileUploadTotal.WithLabelValues("markdown").Inc()

	newArticle := &model.Article{
		ID:            articleId,
		Title:         in.Title,
		Brief:         *in.Brief,
		Content:       objectName, // 这里存的是 MinIO 的路径，而不是原文
		CoverImageURL: *in.CoverImageUrl,
		ManualTypeTag: in.ManualTypeTag,
		SecondaryTags: model.StringArray(in.SecondaryTags),
		AuthorID:      in.AuthorId,
		Status:        int32(__.ArticleStatus_REVIEWING),
	}

	span.AddEvent("start db insert")
	if err := l.svcCtx.ArticleRepo.Insert(ctx, newArticle); err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorDbUpdate, err, logger.WithArticleID(articleId), logger.WithUserID(in.AuthorId))
		return nil, err
	}
	span.AddEvent("db insert success")
	//统计文章创建总数
	metrics.ArticleTotal.WithLabelValues("create").Inc()
	//统计待审核状态的文章总数
	metrics.ArticleStatusTotal.WithLabelValues("reviewing").Inc()

	msg := struct {
		ArticleId   string `json:"article_id"`
		AuthorId    string `json:"author_id"`
		ContentPath string `json:"content_path"`
	}{
		ArticleId:   articleId,
		AuthorId:    in.AuthorId,
		ContentPath: objectName,
	}

	msgBytes, _ := json.Marshal(msg)
	span.AddEvent("start kafka push")
	if err := l.svcCtx.KqPusher.Push(ctx, string(msgBytes)); err != nil {
		span.RecordError(err)
		//统计 Kafka 消息推送失败数
		metrics.KafkaPushErrors.WithLabelValues().Inc()
		err = fmt.Errorf("kafka push failed, payload: %s, error: %w", string(msgBytes), err)
		logger.LogBusinessErr(ctx, errmsg.Error, err, logger.WithArticleID(articleId), logger.WithUserID(in.AuthorId))
	} else {
		span.AddEvent("kafka push success")
	}

	return &__.CreateArticleResponse{
		ArticleId: articleId,
	}, nil
}
