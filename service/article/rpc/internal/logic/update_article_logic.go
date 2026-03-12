package logic

import (
	"context"
	"strings"

	"fmt"

	"sea-try-go/service/article/common/errmsg"
	"sea-try-go/service/article/rpc/internal/svc"
	"sea-try-go/service/article/rpc/metrics"
	"sea-try-go/service/article/rpc/pb"
	"sea-try-go/service/common/logger"

	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UpdateArticleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateArticleLogic {
	return &UpdateArticleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateArticleLogic) UpdateArticle(in *__.UpdateArticleRequest) (*__.UpdateArticleResponse, error) {
	tracer := otel.Tracer("article-rpc")
	ctx, span := tracer.Start(l.ctx, "UpdateArticle", trace.WithAttributes(
		attribute.String("article_id", in.ArticleId),
	))
	defer span.End()

	span.AddEvent("start find article")
	article, err := l.svcCtx.ArticleRepo.FindOne(ctx, in.ArticleId)
	if err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorDbSelect, err, logger.WithArticleID(in.ArticleId))
		return nil, err
	}
	if article == nil {
		err = fmt.Errorf("article not found")
		span.RecordError(err)
		return nil, err
	}
	span.AddEvent("find article success")

	if in.Title != nil {
		article.Title = *in.Title
	}
	if in.Brief != nil {
		article.Brief = *in.Brief
	}
	if in.MarkdownContent != nil {
		objectName := article.Content
		if objectName == "" {
			objectName = fmt.Sprintf("%s%s.md", l.svcCtx.Config.MinIO.ArticlePath, article.ID)
			article.Content = objectName
		}

		contentType := "text/markdown"
		reader := strings.NewReader(*in.MarkdownContent)

		// 统计 MinIO put 操作（更新文章内容）耗时
		timer := prometheus.NewTimer(metrics.MinioRequestDuration.WithLabelValues("put"))
		span.AddEvent("start update minio content")
		_, err = l.svcCtx.MinioClient.PutObject(ctx, l.svcCtx.Config.MinIO.BucketName, objectName,
			reader, int64(len(*in.MarkdownContent)), minio.PutObjectOptions{ContentType: contentType})
		timer.ObserveDuration()

		if err != nil {
			span.RecordError(err)
			//统计 MinIO put 操作（更新文章内容）失败数
			metrics.MinioRequestErrors.WithLabelValues("put").Inc()
			logger.LogBusinessErr(ctx, errmsg.ErrorMinioUpload, fmt.Errorf("update minio content failed: %w", err), logger.WithArticleID(in.ArticleId))
			return nil, err
		}
		span.AddEvent("update minio content success")
		//统计 markdown 文件（更新）上传总数
		metrics.FileUploadTotal.WithLabelValues("markdown").Inc()
	}
	if in.CoverImageUrl != nil {
		article.CoverImageURL = *in.CoverImageUrl
	}
	if in.ManualTypeTag != nil {
		article.ManualTypeTag = *in.ManualTypeTag
	}
	if len(in.SecondaryTags) > 0 {
		article.SecondaryTags = in.SecondaryTags
	}
	if in.Status != nil && *in.Status != __.ArticleStatus_ARTICLE_STATUS_UNSPECIFIED {
		article.Status = int32(in.Status.Number())
	}

	span.AddEvent("start db update")
	if err := l.svcCtx.ArticleRepo.Update(ctx, article); err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorDbUpdate, err, logger.WithArticleID(in.ArticleId))
		return nil, err
	}
	span.AddEvent("db update success")
	//统计文章更新总数
	metrics.ArticleTotal.WithLabelValues("update").Inc()

	return &__.UpdateArticleResponse{
		Success: true,
	}, nil
}
