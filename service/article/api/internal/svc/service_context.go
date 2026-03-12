// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/zeromicro/go-zero/zrpc"
	"sea-try-go/service/article/api/internal/config"
	"sea-try-go/service/article/rpc/articleservice"
	"sea-try-go/service/security/rpc/client/imagesecurityservice"
)

type ServiceContext struct {
	Config     config.Config
	ArticleRpc articleservice.ArticleService
	SecurityRpc imagesecurityservice.ImageSecurityService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		ArticleRpc: articleservice.NewArticleService(zrpc.MustNewClient(c.ArticleRpcConf)),
		SecurityRpc: imagesecurityservice.NewImageSecurityService(zrpc.MustNewClient(c.SecurityRpcConf)),
	}
}
