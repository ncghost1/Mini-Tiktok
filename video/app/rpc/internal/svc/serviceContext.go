package svc

import (
	"Mini-Tiktok/user/app/rpc/userrpc"
	"Mini-Tiktok/video/app/rpc/internal/config"
	"Mini-Tiktok/video/app/rpc/model"
	"Mini-Tiktok/video/app/rpc/model/redisCache"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"
	"log"
	"time"
)

type ServiceContext struct {
	Config      config.Config
	UserRpc     userrpc.UserRpc
	Redis       *redisCache.RedisPool
	Db          *gorm.DB
	KafkaWriter *kafka.Writer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := model.InitGorm(c.DbConfig)
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	pool := redisCache.NewRedisPool(c)
	conn := pool.NewRedisConn()
	_, err = conn.Do("PING")
	defer conn.Close()
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return &ServiceContext{
		Config:  c,
		UserRpc: userrpc.NewUserRpc(zrpc.MustNewClient(c.UserRpc)),
		Redis:   pool,
		Db:      db,
		KafkaWriter: getKafkaWriter(c.KafkaConfig.Host,
			c.KafkaConfig.Topic,
			c.KafkaConfig.BatchTimeout,
			c.KafkaConfig.BatchSize,
			c.KafkaConfig.BatchBytes,
		),
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
