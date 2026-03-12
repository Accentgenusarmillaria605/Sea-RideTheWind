package svc

import (
	"sea-try-go/service/user/admin/rpc/internal/config"
	"sea-try-go/service/user/admin/rpc/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config     config.Config
	AdminModel *model.AdminModel
	BizRedis   *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {

	db := model.InitDB(c.DataSource)
	return &ServiceContext{
		Config:     c,
		AdminModel: model.NewAdminModel(db),
		BizRedis:   redis.MustNewRedis(c.BizRedis),
	}
}
