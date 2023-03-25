package svc

import (
	"Mini-Tiktok/api/internal/config"
	"Mini-Tiktok/jwt/app/rpc/jwtrpc"
	"Mini-Tiktok/user/app/rpc/userrpc"
	"Mini-Tiktok/video/app/rpc/videorpc"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/zrpc"
	"time"
)

type ServiceContext struct {
	Config      config.Config
	UserRpc     userrpc.UserRpc
	JwtRpc      jwtrpc.JwtRpc
	KafkaWriter *kafka.Writer
	VideoRpc    videorpc.VideoRpc
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:  c,
		UserRpc: userrpc.NewUserRpc(zrpc.MustNewClient(c.UserRpc)),
		JwtRpc:  jwtrpc.NewJwtRpc(zrpc.MustNewClient(c.JwtRpc)),
		KafkaWriter: getKafkaWriter(c.KafkaConfig.Host,
			c.KafkaConfig.Topic,
			c.KafkaConfig.BatchTimeout,
			c.KafkaConfig.BatchSize,
			c.KafkaConfig.BatchBytes,
		),
		VideoRpc: videorpc.NewVideoRpc(zrpc.MustNewClient(c.VideoRpc)),
	}
}

func getKafkaWriter(host, topic string, timeout int, size int, bytes int64) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(host),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: time.Millisecond * time.Duration(timeout),
		BatchSize:    size,
		BatchBytes:   bytes,
	}
}
