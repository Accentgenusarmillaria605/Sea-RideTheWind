package logic

import (
	"context"
	"fmt"
	"sea-try-go/service/common/logger"
	"sea-try-go/service/common/snowflake"
	"sea-try-go/service/user/admin/rpc/internal/model"
	"sea-try-go/service/user/admin/rpc/internal/svc"
	"sea-try-go/service/user/admin/rpc/pb"
	"sea-try-go/service/user/common/cryptx"
	"sea-try-go/service/user/common/errmsg"

	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAdminLogic {
	return &CreateAdminLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateAdminLogic) CreateAdmin(in *pb.CreateAdminReq) (*pb.CreateAdminResp, error) {
	tracer := otel.Tracer("admin-rpc")
	ctx, span := tracer.Start(l.ctx, "Action-Admin-CreateAdmin")
	defer span.End()
	span.SetAttributes(
		attribute.String("audit.admin_username", in.Username),
	)
	_, err := l.svcCtx.AdminModel.FindOneAdminByUsername(ctx, in.Username)
	if err == nil {
		logger.LogBusinessErr(ctx, errmsg.ErrorUserExist, fmt.Errorf("username has existed"))
		return nil, status.Error(codes.AlreadyExists, "用户名已存在")
	}
	if err != model.ErrorNotFound {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorDbSelect, err)
		return nil, status.Error(codes.Internal, "DB查询错误")
	}

	password, err := cryptx.PasswordEncrypt(in.Password)
	if err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, err)
		return nil, status.Error(codes.Internal, "密码加密失败")

	}
	uid, err := snowflake.GetID()
	if err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorServerCommon, err)
		return nil, status.Error(codes.Internal, "ID生成失败")
	}
	admin := &model.Admin{
		Uid:       uid,
		Username:  in.Username,
		Password:  password,
		Email:     in.Email,
		ExtraInfo: in.ExtraInfo,
	}
	err = l.svcCtx.AdminModel.InsertOneAdmin(ctx, admin)
	if err != nil {
		span.RecordError(err)
		logger.LogBusinessErr(ctx, errmsg.ErrorDbUpdate, err)
		return nil, status.Error(codes.Internal, "DB添加失败")
	}
	logger.LogInfo(ctx, fmt.Sprintf("add admin success,uid: %d", uid))
	return &pb.CreateAdminResp{
		Uid: admin.Uid,
	}, nil
}
