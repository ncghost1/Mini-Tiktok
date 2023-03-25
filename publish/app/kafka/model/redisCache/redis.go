package redisCache

import (
	"Mini-Tiktok/publish/app/kafka/internal/config"
	"Mini-Tiktok/publish/app/kafka/model"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"time"
)

type RedisPool struct {
	pool *redis.Pool
}

// NewRedisPool 新建一个 redis 连接池
func NewRedisPool(config config.Config) *RedisPool {
	return &RedisPool{&redis.Pool{
		MaxIdle:     config.RedisConfig.MaxIdle, //最大空闲连接数
		MaxActive:   config.RedisConfig.Active,  //最大连接数
		IdleTimeout: time.Duration(config.RedisConfig.IdleTimeout) * time.Second,
		Wait:        true, //超过连接数后是否等待
		Dial: func() (redis.Conn, error) {
			redisUri := fmt.Sprintf("%s:%d", config.RedisConfig.Host, config.RedisConfig.Port)
			if config.RedisConfig.Auth {
				redisConn, err := redis.Dial("tcp", redisUri,
					redis.DialUsername(config.RedisConfig.Username),
					redis.DialPassword(config.RedisConfig.Password))
				if err != nil {
					return nil, err
				}
				return redisConn, nil
			} else {
				redisConn, err := redis.Dial("tcp", redisUri)
				if err != nil {
					return nil, err
				}
				return redisConn, nil
			}
		},
	}}
}

// NewRedisConn 从连接池中获取一个连接
func (p *RedisPool) NewRedisConn() redis.Conn {
	return p.pool.Get()
}

// AddVideoInfoAndFeed 将视频信息加入 Redis 中视频信息缓存与 Feed 流缓存
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) AddVideoInfoAndFeed(conn redis.Conn, videoId uint64, videoJson []byte, createTime int64, ttl, size int) error {
	FeedCacheKey := model.Video{}.FeedCacheKey()
	VideoCacheKey := model.Video{}.CacheKey(videoId)
	Exat := createTime + int64(ttl)

	// 如果 Feed 缓存已满，则将数量为总容量 1/10 的最早的一批视频从缓存删除掉
	_, err := conn.Do("EVAL", "if (redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[3])) then "+
		"redis.call('ZPOPMIN', KEYS[1], ARGV[3]/10); end;"+
		"redis.call('SET', KEYS[2], ARGV[1],'EXAT', ARGV[4]); "+
		"redis.call('ZADD', KEYS[1], ARGV[2],ARGV[1]); "+
		"redis.call('EXPIREAT', KEYS[1], ARGV[4]); "+
		"return nil; ", 2, FeedCacheKey, VideoCacheKey, videoJson, createTime, size, Exat)
	if err != nil {
		return err
	}
	return nil
}
