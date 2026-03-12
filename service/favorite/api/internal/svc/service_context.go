package svc

import (
	"sea-try-go/service/favorite/api/internal/config"
	"sea-try-go/service/favorite/api/internal/middleware"
	"sea-try-go/service/favorite/rpc/favoriteservice"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                   config.Config
	FavoriteRpc              favoriteservice.FavoriteService
	CheckBlacklistMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisDB := redis.MustNewRedis(c.BizRedis)

	return &ServiceContext{
		Config:                   c,
		FavoriteRpc:              favoriteservice.NewFavoriteService(zrpc.MustNewClient(c.FavoriteRpc)),
		CheckBlacklistMiddleware: middleware.NewCheckBlacklistMiddleware(redisDB).Handle,
	}
}
