package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"sea-try-go/service/common/logger"
	"sea-try-go/service/like/rpc/internal/model"
	"sea-try-go/service/like/rpc/internal/mq/internal/config"
	"sea-try-go/service/like/rpc/internal/mq/internal/mqs"
	"sea-try-go/service/like/rpc/internal/mq/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/mq.yaml", "the config file")

type OutboxRelayService struct {
	ctx    context.Context
	cancel context.CancelFunc
	svcCtx *svc.ServiceContext
	sender *mqs.LikeOutboxSender
}

func (s *OutboxRelayService) Start() {
	fmt.Println("Starting Outbox Relay...")
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-s.ctx.Done():
			fmt.Println("Stopping Outbox Relay...")
		case <-ticker.C:
			_ = s.sender.SendPending(s.ctx, 100)
		}
	}
}

func (s *OutboxRelayService) Stop() {
	s.cancel()
}

func main() {
	flag.Parse()

	var c config.Config

	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	logger.Init(c.Name)

	model.InitDB(c.DB.DataSource)

	backgroundCtx := context.Background()

	serviceGroup := service.NewServiceGroup()

	defer serviceGroup.Stop()

	consumer := kq.MustNewQueue(c.Kafka, mqs.NewLikeUpdateService(backgroundCtx, ctx))

	serviceGroup.Add(consumer)

	relayCtx, cancel := context.WithCancel(backgroundCtx)
	relayService := &OutboxRelayService{
		ctx:    relayCtx,
		cancel: cancel,
		svcCtx: ctx,
		sender: mqs.NewLikeOutboxSender(ctx),
	}
	serviceGroup.Add(relayService)

	fmt.Printf("Starting mq consumer [%s]...\n", c.Name)

	serviceGroup.Start()
}
