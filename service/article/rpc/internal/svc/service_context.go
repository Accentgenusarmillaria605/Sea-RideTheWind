package svc

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"
	"log"
	"sea-try-go/service/article/rpc/internal/config"
	"sea-try-go/service/article/rpc/internal/model"
	"sea-try-go/service/common/snowflake"
	"sea-try-go/service/security/rpc/client/contentsecurityservice"
	"sea-try-go/service/security/rpc/client/imagesecurityservice"
)

type ServiceContext struct {
	Config      config.Config
	ArticleRepo *model.ArticleRepo
	KqPusher    *kq.Pusher

	MinioClient      *minio.Client
	HotEventPusher   *kq.Pusher
	SecurityRpc      contentsecurityservice.ContentSecurityService
	ImageSecurityRpc imagesecurityservice.ImageSecurityService
}

func NewServiceContext(c config.Config, articleRepo *model.ArticleRepo) *ServiceContext {

	minioClient, err := minio.New(c.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.MinIO.AccessKeyID, c.MinIO.SecretAccessKey, ""),
		Secure: c.MinIO.UseSSL,
	})
	if err != nil {
		panic(err)
	}

	err = minioClient.MakeBucket(context.Background(), c.MinIO.BucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), c.MinIO.BucketName)
		if errBucketExists == nil && exists {
		} else {
			log.Println("Error creating bucket:", err)
		}
	} else {
		policy := `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::` + c.MinIO.BucketName + `/*"],"Sid": ""}]}`
		err = minioClient.SetBucketPolicy(context.Background(), c.MinIO.BucketName, policy)
		if err != nil {
			log.Println("Error setting bucket policy:", err)
		}
	}

	snowflake.Init()

	securityClient := zrpc.MustNewClient(c.SecurityConf)
	return &ServiceContext{
		Config:      c,
		ArticleRepo: articleRepo,
		KqPusher:    kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic),
		MinioClient: minioClient,
		HotEventPusher: kq.NewPusher(
			c.HotEventPusherConf.Brokers,
			c.HotEventPusherConf.Topic,
		),
		SecurityRpc:      contentsecurityservice.NewContentSecurityService(securityClient),
		ImageSecurityRpc: imagesecurityservice.NewImageSecurityService(securityClient),
	}
}
