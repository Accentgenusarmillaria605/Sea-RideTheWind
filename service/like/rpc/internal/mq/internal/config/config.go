package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

type Config struct {
	service.ServiceConf
	HotEventPusherConf struct {
		Brokers []string
		Topic   string
	}
	Kafka kq.KqConf
	DB    struct {
		DataSource string
	}
}
