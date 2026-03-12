package logic

import (
	"context"
	"sea-try-go/service/comment/rpc/internal/metrics"
	"sea-try-go/service/comment/rpc/internal/model"
	"sea-try-go/service/comment/rpc/internal/svc"
	"sea-try-go/service/comment/rpc/pb"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type GetCommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCommentLogic {
	return &GetCommentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCommentLogic) GetComment(in *pb.GetCommentReq) (resp *pb.GetCommentResp, err error) {
	start := time.Now()

	defer func() {
		result := "ok"
		if err != nil {
			result = "sys_fail"
		}

		metrics.CommentRequestCounterMetric.
			WithLabelValues("comment_rpc", "GetComment", result).
			Inc()

		metrics.CommentRequestSecondsCounterMetric.
			WithLabelValues("comment_rpc", "GetComment").
			Add(time.Since(start).Seconds())
	}()

	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx, cancel := context.WithTimeout(l.ctx, time.Second)
	defer cancel()

	tracer := otel.Tracer("comment.rpc")
	ctx, span := tracer.Start(ctx, "GetComment")
	defer span.End()

	span.SetAttributes(
		attribute.String("target_id", in.TargetId),
		attribute.Int64("root_id", in.RootId),
		attribute.Int64("page", in.Page),
		attribute.String("target_type", in.TargetType),
		attribute.Int64("sort_type", in.SortType),
	)

	conn := l.svcCtx.CommentModel

	//子span
	var subject model.Subject
	{
		cctx, cspan := tracer.Start(ctx, "cache_aside.GetSubjectWithCache")
		s, e := l.svcCtx.CommentCache.GetSubjectWithCache(cctx, in.TargetType, in.TargetId, conn)
		if e != nil {
			cspan.RecordError(e)
			cspan.SetStatus(codes.Error, e.Error())
			cspan.End()
			span.RecordError(e)
			span.SetStatus(codes.Error, e.Error())
			err = e
			return nil, err
		}
		subject = s
		cspan.End()
	}
	/*subject, err := l.svcCtx.CommentCache.GetSubjectWithCache(ctx, in.TargetType, in.TargetId, conn)
	if err != nil {
		return nil, err
	}*/
	var sortType model.ReplySort
	if in.SortType == 1 {
		sortType = model.ReplySortTime
	} else {
		sortType = model.ReplySortTime
	}

	var ids []int64
	{
		cctx, cspan := tracer.Start(ctx, "cache_aside.GetReplyIDsPageCache")
		got, e := l.svcCtx.CommentCache.GetReplyIDsPageCache(cctx, model.GetReplyIDsPageReq{
			TargetType: in.TargetType,
			TargetId:   in.TargetId,
			RootId:     in.RootId,
			Offset:     0,
			Limit:      int(in.Page),
			Sort:       sortType,
			OnlyNormal: false,
		}, conn)
		if e != nil {
			cspan.RecordError(e)
			cspan.SetStatus(codes.Error, e.Error())
			cspan.End()
			span.RecordError(e)
			span.SetStatus(codes.Error, e.Error())
			err = e
			return nil, err
		}
		ids = got
		cspan.SetAttributes(attribute.Int("reply_ids.count", len(ids)))
		cspan.End()
	}
	/*ids, err := l.svcCtx.CommentCache.GetReplyIDsPageCache(ctx, model.GetReplyIDsPageReq{
		TargetType: in.TargetType,
		TargetId:   in.TargetId,
		RootId:     in.RootId,
		Offset:     0,
		Limit:      int(in.Page),
		Sort:       sortType,
		OnlyNormal: false,
	}, conn)
	if err != nil {
		return nil, err
	}*/
	var index []model.CommentIndex
	{
		cctx, cspan := tracer.Start(ctx, "cache_aside.GetCommentIndexCache")
		got, e := l.svcCtx.CommentCache.GetCommentIndexCache(cctx, ids, conn)
		if e != nil {
			cspan.RecordError(e)
			cspan.SetStatus(codes.Error, e.Error())
			cspan.End()
			span.RecordError(e)
			span.SetStatus(codes.Error, e.Error())
			err = e
			return nil, err
		}
		index = got
		cspan.End()
	}
	/*index, err := l.svcCtx.CommentCache.GetCommentIndexCache(ctx, ids, conn)
	if err != nil {
		return nil, err
	}*/
	var content []model.CommentContent
	{
		cctx, cspan := tracer.Start(ctx, "cache_aside.BatchGetContentCache")
		got, e := l.svcCtx.CommentCache.BatchGetContentCache(cctx, ids, conn)
		if e != nil {
			cspan.RecordError(e)
			cspan.SetStatus(codes.Error, e.Error())
			cspan.End()
			span.RecordError(e)
			span.SetStatus(codes.Error, e.Error())
			err = e
			return nil, err
		}
		content = got
		cspan.End()
	}
	/*content, err := l.svcCtx.CommentCache.BatchGetContentCache(ctx, ids, conn)
	if err != nil {
		return nil, err
	}*/
	comment := make([]*pb.CommentItem, 0, len(content))
	for i, _ := range index {
		u := index[i]
		v := content[i]
		comment = append(comment, &pb.CommentItem{
			Id:           u.Id,
			UserId:       u.UserId,
			Content:      v.Content,
			RootId:       u.RootId,
			ParentId:     u.ParentId,
			LikeCount:    u.LikeCount,
			DislikeCount: u.DislikeCount,
			ReplyCount:   u.ReplyCount,
			Attribute:    u.Attribute,
			State:        pb.CommentState(u.State),
			CreatedAt:    u.CreatedAt.Format("2006-01-02 15:04:05"),
			Meta:         v.Meta,
			Children:     nil, //日后再说
		})
	}

	//logger.LogInfo(l.ctx, "get comment success")

	metrics.CommentListSizeGaugeMetric.
		WithLabelValues("comment_list", "GetComment").
		Set(float64(len(comment)))
	
	resp = &pb.GetCommentResp{
		Comment: comment,
		Subject: &pb.SubjectInfo{
			TargetType: subject.TargetType,
			TargetId:   subject.TargetId,
			TotalCount: subject.TotalCount,
			RootCount:  subject.RootCount,
			State:      pb.SubjectState(subject.State),
			Attribute:  subject.Attribute,
		},
	}
	return resp, nil
}
