// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package comment

import (
	"context"
	"encoding/json"
	"fmt"

	"sea-try-go/service/comment/api/internal/svc"
	"sea-try-go/service/comment/api/internal/types"
	"sea-try-go/service/comment/common/errmsg"
	"sea-try-go/service/comment/rpc/pb"
	"sea-try-go/service/common/logger"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/status"
)

type ReportCommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReportCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReportCommentLogic {
	return &ReportCommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReportCommentLogic) ReportComment(req *types.ReportCommentReq) (resp *types.ReportCommentResp, err error) {
	userId, ok := l.ctx.Value("userId").(json.Number)
	if !ok {
		logger.LogBusinessErr(l.ctx, errmsg.ErrorTokenRuntime, fmt.Errorf("ctx userId is not json.Number"))
		return nil, errmsg.NewErrCode(errmsg.ErrorTokenRuntime)
	}
	uid, err := userId.Int64()

	if err != nil {
		logger.LogBusinessErr(l.ctx, errmsg.ErrorTokenRuntime, fmt.Errorf("parse userId to int64 failed: %v", err))
		return nil, errmsg.NewErrCode(errmsg.ErrorTokenRuntime)
	}

	rpcReq := &pb.ReportCommentReq{
		UserId:     uid,
		CommentId:  req.CommentId,
		TargetType: req.TargetType,
		TargetId:   req.TargetId,
		Reason:     pb.ReportReason(req.Reason),
		Detail:     req.Detail,
	}

	rpcResp, err := l.svcCtx.CommentCli.ReportComment(l.ctx, rpcReq)

	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return nil, errmsg.NewErrCodeMsg(int(st.Code()), st.Message())
		}
		logger.LogBusinessErr(l.ctx, errmsg.ErrorServerCommon, err)
		return nil, errmsg.NewErrCode(errmsg.ErrorServerCommon)
	}

	return &types.ReportCommentResp{
		Success: rpcResp.Success,
	}, nil
}
