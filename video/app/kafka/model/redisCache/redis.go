package redisCache

import (
	"Mini-Tiktok/video/app/kafka/internal/config"
	"Mini-Tiktok/video/app/kafka/model"
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

// RemDelCacheMember 将视频 id 从用户最近取消点赞视频集合中删除的操作
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) RemDelCacheMember(conn redis.Conn, videoId, userId uint64) error {
	DelCacheKey := model.Favorite{}.DelCacheKey(userId)
	_, err := conn.Do("SREM", DelCacheKey, videoId)
	if err != nil {
		return err
	}
	return nil
}

// DelFavorite 完成视频点赞数 - 1 和将视频 id 从用户最新点赞视频有序集合中删除的操作
// 用于在点赞消息消费失败时将缓存的点赞数据删除，相当于给缓存回滚，保证数据一致。
// （取消点赞失败不需要回滚，因为获取点赞消息若缓存找不到会查数据库，以数据库的数据为准）
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) DelFavorite(conn redis.Conn, videoId, userId uint64) error {
	CacheKey := model.Favorite{}.CacheKey(userId)
	CountCacheKey := model.Favorite{}.CountCacheKey(videoId)
	_, err := conn.Do("EVAL", "redis.call('ZREM', KEYS[1],ARGV[1]);"+
		"redis.call('DECR', KEYS[2]); "+
		"return nil; ", 2, CacheKey, CountCacheKey, videoId)
	if err != nil {
		return err
	}
	return nil
}
