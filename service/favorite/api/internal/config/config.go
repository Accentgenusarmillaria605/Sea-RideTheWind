package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	UserAuth struct {
		AccessSecret string
		AccessExpire int64
	}
	FavoriteRpc zrpc.RpcClientConf
	BizRedis    redis.RedisConf
}
