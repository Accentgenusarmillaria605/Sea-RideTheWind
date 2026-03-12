package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Postgres struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		Mode     string
	}
	BizRedis redis.RedisConf
	UserAuth struct {
		AccessSecret string
		AccessExpire int64
	}
}
