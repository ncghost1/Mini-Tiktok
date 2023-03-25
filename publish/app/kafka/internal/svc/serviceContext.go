package svc

import (
	"Mini-Tiktok/publish/app/kafka/internal/config"
	"Mini-Tiktok/publish/app/kafka/model"
	"Mini-Tiktok/publish/app/kafka/model/redisCache"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"gorm.io/gorm"
	"log"
)

type ServiceContext struct {
	Config config.Config
	Oss    *oss.Client
	Redis  *redisCache.RedisPool
	Db     *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	endpoint := c.AliyunOss.Endpoint
	accessKeyId := c.AliyunOss.AccessKeyId
	accessKeySecret := c.AliyunOss.AccessKeySecret
	ossCli, err := oss.New(endpoint, accessKeyId, accessKeySecret)

	db, err := model.InitGorm(c.DbConfig)
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	pool := redisCache.NewRedisPool(c)
	conn := pool.NewRedisConn()
	_, err = conn.Do("PING") // 测试连接
	defer conn.Close()
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return &ServiceContext{
		Config: c,
		Oss:    ossCli,
		Db:     db,
		Redis:  pool,
	}
}
