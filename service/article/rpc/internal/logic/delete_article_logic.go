package logic

import (
	"context"
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

type DeleteArticleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteArticleLogic {
	return &DeleteArticleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteArticleLogic) DeleteArticle(in *__.DeleteArticleRequest) (*__.DeleteArticleResponse, error) {
	tracer := otel.Tracer("article-rpc")
	spanCtx, span := tracer.Start(l.ctx, "DeleteArticle", trace.WithAttributes(
		attribute.String("article_id", in.ArticleId),
	))
	defer span.End()

	span.AddEvent("start find article")
	article, err := l.svcCtx.ArticleRepo.FindOne(spanCtx, in.ArticleId)
	if err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(spanCtx, errmsg.ErrorDbSelect, err, logger.WithArticleID(in.ArticleId))
		return nil, err
	}
	if article == nil {
		err = fmt.Errorf("article not found")
		span.RecordError(err)
		return nil, err
	}
	span.AddEvent("find article success")

	if article.Content != "" {
		//统计 MinIO delete 操作耗时
		timer := prometheus.NewTimer(metrics.MinioRequestDuration.WithLabelValues("delete"))
		span.AddEvent("start remove minio object")
		err = l.svcCtx.MinioClient.RemoveObject(spanCtx, l.svcCtx.Config.MinIO.BucketName, article.Content, minio.RemoveObjectOptions{})
		timer.ObserveDuration()
		if err != nil {
			span.RecordError(err)
			//统计 MinIO delete 操作失败数
			metrics.MinioRequestErrors.WithLabelValues("delete").Inc()
			logger.LogBusinessErr(spanCtx, errmsg.ErrorMinioDelete, fmt.Errorf("remove minio object failed: %w", err), logger.WithArticleID(in.ArticleId))
			return nil, err
		}
		span.AddEvent("remove minio object success")
	}

	span.AddEvent("start db delete")
	if err := l.svcCtx.ArticleRepo.Delete(spanCtx, in.ArticleId); err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(spanCtx, errmsg.ErrorDbUpdate, err, logger.WithArticleID(in.ArticleId))
		return nil, err
	}
	span.AddEvent("db delete success")
	//统计文章删除总数
	metrics.ArticleTotal.WithLabelValues("delete").Inc()

	return &__.DeleteArticleResponse{
		Success: true,
	}, nil
}
