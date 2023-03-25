package redisCache

import (
	"Mini-Tiktok/video/app/rpc/internal/config"
	"Mini-Tiktok/video/app/rpc/model"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type RedisPool struct {
	pool *redis.Pool
}

const (
	CACHE_KEY_NOT_EXISTS_MSG = "cache key not exists or has wrong type"
	COUNT_NOT_FOUND          = int64(-1)
	FAVORITE_EXISTS          = "favorite already exists"
)

var cacheConfig *config.CacheConfig

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

func (p *RedisPool) IncrExStringVal(conn redis.Conn, key string, ttl int) error {
	_, err := conn.Do("EVAL", "redis.call('EXPIRE', KEYS[1],ARGV[1]); "+
		"redis.call('INCR', KEYS[1]); "+
		"return nil", 1, key, ttl)
	if err != nil {
		return err
	}
	return nil
}

func (p *RedisPool) DecrExStringVal(conn redis.Conn, key string, ttl int) error {
	_, err := conn.Do("EVAL", "redis.call('EXPIRE', KEYS[1],ARGV[1]); "+
		"redis.call('DECR', KEYS[1]); "+
		"return nil", 1, key, ttl)
	if err != nil {
		return err
	}
	return nil
}

func (p *RedisPool) GetInt64Val(conn redis.Conn, key string) (int64, error) {
	raw, err := conn.Do("GET", key)
	if err != nil {
		return -1, err
	}
	count, ok := raw.(int64)
	if !ok {
		return -1, err
	}
	return count, nil
}

func (p *RedisPool) GetExInt64Val(conn redis.Conn, key string, ttl int) (int64, error) {
	raw, err := conn.Do("GETEX", key, "EX", ttl)
	if err != nil {
		return -1, err
	}
	count, ok := raw.(int64)
	if !ok {
		return -1, err
	}
	return count, nil
}

func (p *RedisPool) GetExStringVal(conn redis.Conn, key string, ttl int) ([]byte, error) {
	raw, err := conn.Do("GETEX", key, "EX", ttl)
	if err != nil {
		return nil, err
	}
	val, ok := raw.([]byte)
	if !ok {
		return nil, err
	}
	return val, nil
}

func (p *RedisPool) SetExInt64(conn redis.Conn, key string, val int64, ttl int) error {
	_, err := conn.Do("SETEX", key, ttl, val)
	if err != nil {
		return err
	}
	return nil
}

// IsFavorite 从为用户存储的最新 30 个点赞视频缓存中查找 videoId 是否存在
// 不存在不代表用户未点赞过该视频，需要到数据库中查找
func (p *RedisPool) IsFavorite(conn redis.Conn, key string, videoId uint64) (bool, error) {
	raw, err := conn.Do("ZRANK", key, videoId)
	if err != nil {
		return false, err
	}
	_, ok := raw.(int64)
	if !ok {
		if r, ok := raw.(string); ok {
			if r == "nil" {
				return false, nil
			} else {
				return false, errors.New(CACHE_KEY_NOT_EXISTS_MSG)
			}
		}
		return false, errors.New(CACHE_KEY_NOT_EXISTS_MSG)
	}
	return true, nil
}

// AddFavorite 完成视频点赞数 + 1 和将视频 id 写入用户最新点赞视频有序集合的操作
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) AddFavorite(conn redis.Conn, videoId, userId uint64, createTime int64) error {
	CacheKey := model.Favorite{}.CacheKey(userId)
	CountCacheKey := model.Favorite{}.CountCacheKey(videoId)
	Exat := createTime + int64(cacheConfig.FAVORITE_CACHE_TTL)
	raw, err := conn.Do("EVAL", "if (tonumber(redis.call('ZRANK', KEYS[1], ARGV[1])) ~= nil) then "+
		"return -1; end; "+
		"if (redis.call('ZCARD',KEYS[1]) >= tonumber(ARGV[3])) then "+
		"redis.call('ZPOPMIN', KEYS[1]); end;"+
		"redis.call('ZADD', KEYS[1], ARGV[2],ARGV[1]); "+
		"redis.call('INCR', KEYS[2]); "+
		"redis.call('EXPIREAT', KEYS[1], ARGV[4]); "+
		"redis.call('EXPIREAT', KEYS[2], ARGV[4]); "+
		"return nil; ", 2, CacheKey, CountCacheKey, videoId, createTime, cacheConfig.VIDEO_FAVORITE_MAX_CACHE_SIZE, Exat)
	if err != nil {
		return err
	}
	if raw != nil {
		return errors.New(FAVORITE_EXISTS)
	}
	return nil
}

// SendSetExFavorCount 设置缓存视频点赞数并设置超时时间
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendSetExFavorCount(conn redis.Conn, videoId uint64, count int64, ttl int) error {
	CountCacheKey := model.Favorite{}.CountCacheKey(videoId)
	err := conn.Send("SETEX", CountCacheKey, ttl, count)
	if err != nil {
		return err
	}
	return nil
}

// SendAddFavorList 添加用户最新点赞视频缓存，并设置超时时间
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendAddFavorList(conn redis.Conn, userid, videoid uint64, create_time int64, ttl int) error {
	CacheKey := model.Favorite{}.CacheKey(userid)
	err := conn.Send("EVAL", "redis.call('ZADD', KEYS[1], ARGV[3],ARGV[1]); "+
		"redis.call('EXPIRE', KEYS[1], ARGV[2]); "+
		"if (redis.call('ZCARD',KEYS[1]) > tonumber(ARGV[4])) then "+
		"redis.call('ZPOPMIN', KEYS[1]); end; "+
		"return nil; ", 1, CacheKey, videoid, ttl, create_time, cacheConfig.VIDEO_FAVORITE_MAX_CACHE_SIZE)
	if err != nil {
		return err
	}
	return nil
}

// DelFavorite 完成视频点赞数 - 1 和将视频 id 从用户最新点赞视频有序集合中删除的操作
// 同时要将该视频 id 加入到用户最近取消点赞的视频集合，因为异步写库带来的延迟可能会导致刷新后仍然在已点赞状态
// 所以我们用最近取消点赞的视频集合缓存来辅助判断是否点赞过该视频，当写库完成时会将该视频 id 从集合中删除
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) DelFavorite(conn redis.Conn, videoId, userId uint64, ttl, delttl int) error {
	CacheKey := model.Favorite{}.CacheKey(userId)
	CountCacheKey := model.Favorite{}.CountCacheKey(videoId)
	DelCacheKey := model.Favorite{}.DelCacheKey(userId)
	_, err := conn.Do("EVAL", "redis.call('ZREM', KEYS[1],ARGV[1]);"+
		"redis.call('DECR', KEYS[2]); "+
		"redis.call('SADD', KEYS[3], ARGV[1]); "+
		"redis.call('EXPIRE', KEYS[1], ARGV[2]); "+
		"redis.call('EXPIRE', KEYS[2], ARGV[2]); "+
		"redis.call('EXPIRE', KEYS[3], ARGV[3]); "+
		"return nil; ", 3, CacheKey, CountCacheKey, DelCacheKey, videoId, ttl, delttl)
	if err != nil {
		return err
	}
	return nil
}

// PopFavoriteDelCache 将视频 id 从用户最近取消点赞的视频集合中删除
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) PopFavoriteDelCache(conn redis.Conn, videoId, userId uint64) error {
	DelCacheKey := model.Favorite{}.DelCacheKey(userId)
	_, err := conn.Do("EVAL", "redis.call('SREM', KEYS[1],ARGV[1]);"+
		"if (redis.call('SCARD', KEYS[1]) == 0) then "+
		"redis.call('DEL', KEYS[1]); end;"+
		"return nil; ", 1, DelCacheKey, videoId)
	if err != nil {
		return err
	}
	return nil
}

// AddComment 完成视频评论数 + 1 ，将评论 id 写入视频最新评论有序集合，以及将评论信息写入评论缓存的操作
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) AddComment(conn redis.Conn, videoId, commentId uint64, createTime int64, commentJson []byte) error {
	CacheKey := model.Comment{}.CacheKey(videoId)
	IdCacheKey := model.Comment{}.IdCacheKey(videoId)
	CountCacheKey := model.Comment{}.CountCacheKey(videoId)
	Exat := createTime + int64(cacheConfig.COMMENT_CACHE_TTL)
	_, err := conn.Do("EVAL", "if (redis.call('ZCARD',KEYS[2]) >= tonumber(ARGV[5])) then "+
		"redis.call('ZPOPMIN', KEYS[2]); end;"+
		"redis.call('ZADD', KEYS[2], ARGV[2], ARGV[3]); "+
		"redis.call('INCR', KEYS[3]); "+
		"redis.call('SET', KEYS[1], ARGV[4], 'EXAT',ARGV[6]); "+
		"redis.call('EXPIREAT', KEYS[2], ARGV[6]); "+
		"redis.call('EXPIREAT', KEYS[3], ARGV[6]); "+
		"return nil; ", 3, CacheKey, IdCacheKey, CountCacheKey, videoId, createTime, commentId, commentJson, cacheConfig.VIDEO_COMMENT_MAX_CACHE_SIZE, Exat)
	if err != nil {
		return err
	}
	return nil
}

// SendSetExCommentCount 设置缓存视频点赞数并设置超时时间
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendSetExCommentCount(conn redis.Conn, videoId uint64, count int64, ttl int) error {
	CountCacheKey := model.Comment{}.CountCacheKey(videoId)
	err := conn.Send("SETEX", CountCacheKey, ttl, count)
	if err != nil {
		return err
	}
	return nil
}

// SendSetExCommentJson 设置缓存视频 Json 信息并设置超时时间
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendSetExCommentJson(conn redis.Conn, commentId uint64, Info []byte, ttl int) error {
	CacheKey := model.Comment{}.CacheKey(commentId)
	err := conn.Send("SETEX", CacheKey, ttl, Info)
	if err != nil {
		return err
	}
	return nil
}

// SendAddCommentList 添加视频最新评论缓存，并设置超时时间
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendAddCommentList(conn redis.Conn, videoid, commentId uint64, create_time int64, ttl int) error {
	IdCacheKey := model.Comment{}.IdCacheKey(videoid)
	err := conn.Send("EVAL", "redis.call('ZADD', KEYS[2], ARGV[3]);"+
		"redis.call('EXPIRE', KEYS[2], ARGV[2]); "+
		"if (redis.call('ZCARD', KEYS[1]) > tonumber(ARGV[4])) then "+
		"redis.call('ZPOPMIN', KEYS[1]); end;"+
		"return nil; ", 1, IdCacheKey, commentId, ttl, create_time, cacheConfig.VIDEO_COMMENT_MAX_CACHE_SIZE)
	if err != nil {
		return err
	}
	return nil
}

// DelComment 完成视频评论数 - 1 ，将评论 id 从视频最新评论有序集合中删除，以及将评论信息从评论缓存中删除的操作
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) DelComment(conn redis.Conn, videoId, commentId uint64, ttl int) error {
	CacheKey := model.Comment{}.CacheKey(commentId)
	IdCacheKey := model.Comment{}.IdCacheKey(videoId)
	CountCacheKey := model.Comment{}.CountCacheKey(videoId)
	_, err := conn.Do("EVAL", "redis.call('ZREM', KEYS[2],ARGV[1]);"+
		"redis.call('DECR', KEYS[3]); "+
		"redis.call('DEL', KEYS[1]);"+
		"redis.call('EXPIREAT', KEYS[1], ARGV[2]);"+
		"redis.call('EXPIREAT', KEYS[2], ARGV[2]);"+
		"redis.call('EXPIREAT', KEYS[3], ARGV[2]);"+
		"return nil; ", 3, CacheKey, IdCacheKey, CountCacheKey, commentId, time.Now().Unix()+int64(ttl))
	if err != nil {
		return err
	}
	return nil
}

// AddVideoInfoAndFeed 将视频信息加入 Redis 中视频信息缓存， Feed 流缓存以及用户最近发布视频列表缓存
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) AddVideoInfoAndFeed(conn redis.Conn, userid, videoId uint64, videoJson []byte, createTime int64, ttl int) error {
	FeedCacheKey := model.Video{}.FeedCacheKey()
	VideoCacheKey := model.Video{}.CacheKey(videoId)
	PubListCacheKey := model.Video{}.PublishListCacheKey(userid)
	Exat := time.Now().Unix() + int64(ttl)
	// 如果 Feed 缓存已满，则将数量为总容量 1/10 的最早的一批视频从缓存删除掉
	_, err := conn.Do("EVAL",
		"redis.call('SET', KEYS[2], ARGV[2],'EXAT', ARGV[5]); "+
			"redis.call('ZADD', KEYS[1], ARGV[3], ARGV[2]); "+
			"redis.call('EXPIREAT', KEYS[1], ARGV[5]); "+
			"redis.call('ZADD', KEYS[3], ARGV[3], ARGV[1]); "+
			"redis.call('EXPIREAT', KEYS[3], ARGV[5]); "+
			"if (redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[4])) then "+
			"redis.call('ZPOPMIN', KEYS[1], ARGV[4]/10); end; "+
			"return nil; ", 3, FeedCacheKey, VideoCacheKey, PubListCacheKey, videoId, videoJson, createTime, cacheConfig.FEED_MAX_CACHE_SIZE, Exat)
	if err != nil {
		return err
	}
	return nil
}

// SendAddVideoInfo 将视频信息加入 Redis 中视频信息缓存与用户最近发布视频列表缓存
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendAddVideoInfo(conn redis.Conn, userid, videoId uint64, videoJson []byte, createTime int64, ttl int) error {
	VideoCacheKey := model.Video{}.CacheKey(videoId)
	PubListCacheKey := model.Video{}.PublishListCacheKey(userid)
	Exat := time.Now().Unix() + int64(ttl)

	// 如果 Feed 缓存已满，则将数量为总容量 1/10 的最早的一批视频从缓存删除掉
	err := conn.Send("EVAL", "redis.call('SET', KEYS[1], ARGV[2],'EXAT', ARGV[4]); "+
		"redis.call('ZADD', KEYS[2], ARGV[3], ARGV[1]); "+
		"redis.call('EXPIREAT', KEYS[2], ARGV[4]); "+
		"return nil; ", 2, VideoCacheKey, PubListCacheKey, videoId, videoJson, createTime, Exat)
	if err != nil {
		return err
	}
	return nil
}

// SendSetExVideoInfo 将视频信息加入 Redis 中视频信息缓存并设置过期时间
// 注意该函数只是将命令写到缓冲区上，并未发送，需要调用 Redis 连接使用 Flush() 发送
func (p *RedisPool) SendSetExVideoInfo(conn redis.Conn, videoId uint64, videoJson []byte, ttl int) error {
	VideoCacheKey := model.Video{}.CacheKey(videoId)
	// 如果 Feed 缓存已满，则将数量为总容量 1/10 的最早的一批视频从缓存删除掉
	err := conn.Send("SETEX", VideoCacheKey, ttl, videoJson)
	if err != nil {
		return err
	}
	return nil
}

// GetExVideoInfo 获取缓存中的视频信息Json
func (p *RedisPool) GetExVideoInfo(conn redis.Conn, videoId uint64, ttl int) ([]byte, bool, error) {
	val, err := p.GetExStringVal(conn, model.Video{}.CacheKey(videoId), ttl)
	if err != nil {
		return nil, false, err
	}
	if val == nil {
		return nil, false, nil
	}
	return val, true, nil
}

// GetExPublishList 获取缓存中的视频信息Json
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) GetExPublishList(conn redis.Conn, userid uint64, ttl int) ([][]byte, bool, error) {
	var publishList [][]byte
	pubLishCacheKey := model.Video{}.PublishListCacheKey(userid)
	raw, err := conn.Do("EVAL", "if (redis.call('EXISTS',KEYS[1]) ~= 1) then "+
		"return nil; "+
		"else "+
		"local zlist = redis.call('ZRANGE', KEYS[1], 0, -1, 'REV'); "+
		"redis.call('EXPIRE', KEYS[1], ARGV[1]);"+
		"local len = #zlist; "+
		"local res_array = {}; "+
		"for i, v in pairs(zlist) do "+
		"local t = {ARGV[2], v}; "+
		"local key = table.concat(t) "+
		"local info = redis.call('GETEX', key, 'EX', ARGV[1]); "+
		"if(info == false) then "+
		"return nil; else "+
		"table.insert(res_array,info); end; end; "+
		"return res_array; end; ", 1, pubLishCacheKey, ttl, model.VideoCacheKeyPrefix)
	if err != nil {
		return nil, false, err
	}

	if r, ok := raw.([]interface{}); ok {
		for _, v := range r {
			if v == nil {
				return nil, false, nil
			}
			publishList = append(publishList, v.([]byte))
		}
		return publishList, true, nil
	}
	return nil, false, nil

}

// GetExFavComCountIsFavor 获取缓存视频点赞数，评论数，用户是否点赞过该视频
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) GetExFavComCountIsFavor(conn redis.Conn, videoId, userid uint64, favttl, comttl int) (favCount int64, comCount int64, isFavor bool, err error) {
	FavCountKey := model.Favorite{}.CountCacheKey(videoId)
	ComCountKey := model.Comment{}.CountCacheKey(videoId)
	FavListKey := model.Favorite{}.CacheKey(userid)
	now := time.Now().Unix()
	favExat := now + int64(favttl)
	comExat := now + int64(comttl)
	raw, err := conn.Do("EVAL", "local res_array = {}; "+
		"local val = redis.call('GETEX',KEYS[1],'EXAT',ARGV[2]);"+
		"table.insert(res_array, val); "+
		"val = redis.call('GETEX', KEYS[2], 'EXAT', ARGV[3]);"+
		"table.insert(res_array, val); "+
		"val = redis.call('ZRANK', KEYS[3], ARGV[1]);"+
		"table.insert(res_array, val); "+
		"redis.call('EXPIREAT', KEYS[3], ARGV[2]); "+
		"return res_array; ", 3, FavCountKey, ComCountKey, FavListKey, userid, favExat, comExat)
	if err != nil {
		return -1, -1, false, err
	}

	if list, ok := raw.([]interface{}); ok {
		b, ok := list[0].([]byte)
		if !ok {
			favCount = COUNT_NOT_FOUND
		} else {
			str := string(b)
			favCount, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				favCount = COUNT_NOT_FOUND
			}
		}

		b, ok = list[1].([]byte)
		if !ok {
			comCount = COUNT_NOT_FOUND
		} else {
			str := string(b)
			comCount, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				comCount = COUNT_NOT_FOUND
			}
		}

		isFavor = list[2] != nil
	}
	return favCount, comCount, isFavor, nil
}

// GetExCommentCount 获取缓存中的视频评论数，同时更新过期时间
func (p *RedisPool) GetExCommentCount(conn redis.Conn, videoId uint64, ttl int) (int64, bool, error) {
	key := model.Comment{}.CountCacheKey(videoId)
	exists, err := p.ExistKey(conn, key)
	if err != nil {
		return -1, false, err
	}
	if !exists {
		return -1, false, nil
	}

	val, err := p.GetExInt64Val(conn, key, ttl)
	if err != nil {
		return val, false, err
	}
	return val, true, nil
}

// GetexCommentCount 获取缓存中的视频评论数，同时更新过期时间
func (p *RedisPool) GetExFavoriteCount(conn redis.Conn, videoId uint64, ttl int) (int64, bool, error) {
	key := model.Favorite{}.CountCacheKey(videoId)
	exists, err := p.ExistKey(conn, key)
	if err != nil {
		return -1, false, err
	}
	if !exists {
		return -1, false, nil
	}

	val, err := p.GetExInt64Val(conn, key, ttl)
	if err != nil {
		return val, false, err
	}
	return val, true, nil
}

// ExistKey 查询 Key 是否存在
func (p *RedisPool) ExistKey(conn redis.Conn, key string) (bool, error) {
	raw, err := conn.Do("EXISTS", key)
	if err != nil {
		return false, err
	}
	if flag, ok := raw.(int64); ok {
		if flag == 1 {
			return true, nil
		}
		return false, nil
	}

	// 永远不会到达的 return
	return false, err
}

// GetExFavoriteList 获取用户最近点赞过的视频列表并设置过期时间
// 注意返回的数组，二维末尾为该视频列表中最早视频的时间戳
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) GetExFavoriteList(conn redis.Conn, userid uint64, ttl int) ([][]byte, bool, error) {
	var FavoriteList [][]byte
	CacheKey := model.Favorite{}.CacheKey(userid)
	raw, err := conn.Do("EVAL", "if (redis.call('EXISTS',KEYS[1]) ~= 1) then "+
		"return nil; "+
		"else "+
		"local zlist = redis.call('ZRANGE', KEYS[1], 0, -1, 'REV'); "+
		"local len = #zlist; "+
		"redis.call('EXPIRE', KEYS[1], ARGV[1]); "+
		"local res_array = {}; "+
		"for i, v in pairs(zlist) do "+
		"local t = {ARGV[2], v}; "+
		"local key = table.concat(t) "+
		"local info = redis.call('GETEX', key, 'EX', ARGV[1]); "+
		"if(info == false) then "+
		"return nil; else "+
		"table.insert(res_array, info); "+
		"if (i == len) then table.insert(res_array, redis.call('ZSCORE', KEYS[1], v)); end; end; end; "+
		"return res_array; end; ", 1, CacheKey, ttl, model.VideoCacheKeyPrefix)
	if err != nil {
		return nil, false, err
	}

	if r, ok := raw.([]interface{}); ok {
		for _, v := range r {
			if v == nil {
				return nil, false, nil
			}
			FavoriteList = append(FavoriteList, v.([]byte))
		}
		return FavoriteList, true, nil
	}
	return nil, false, nil
}

// GetExCommentList 获取视频最新评论信息（默认30条）并设置过期时间
// 使用 lua 脚本将多次操作整合为一次 RTT
func (p *RedisPool) GetExCommentList(conn redis.Conn, videoId uint64, ttl int) ([][]byte, bool, error) {
	var commentList [][]byte
	IdCacheKey := model.Comment{}.IdCacheKey(videoId)
	raw, err := conn.Do("EVAL", "if (redis.call('EXISTS',KEYS[1]) ~= 1) then "+
		"return nil; "+
		"else "+
		"local zlist = redis.call('ZRANGE', KEYS[1], 0, -1, 'REV'); "+
		"local len = #zlist; "+
		"redis.call('EXPIRE', KEYS[1], ARGV[1]);"+
		"local res_array = {}; "+
		"for i, v in pairs(zlist) do "+
		"local t = {ARGV[2], v}; "+
		"local key = table.concat(t) "+
		"local info = redis.call('GETEX', key, 'EX', ARGV[1]); "+
		"if(info == nil) then "+
		"return nil; else "+
		"table.insert(res_array,info); end; end; "+
		"return res_array; end; ", 1, IdCacheKey, ttl, model.ComCacheKeyPrefix)
	if err != nil {
		return nil, false, err
	}
	if r, ok := raw.([]interface{}); ok {
		for _, v := range r {
			if v == nil {
				return nil, false, nil
			}
			commentList = append(commentList, v.([]byte))
		}
		return commentList, true, nil
	}
	return nil, false, nil
}

// GetFeed 获取 latestTime 前发布的 count 个视频
func (p *RedisPool) GetFeed(conn redis.Conn, latestTime, count int64) ([][]byte, error) {
	var FeedJsonList [][]byte
	raw, err := conn.Do("ZRANGE", model.Video{}.FeedCacheKey(), latestTime, 0, "BYSCORE", "REV", "LIMIT", 0, count)
	if err != nil {
		return nil, err
	}
	for _, v := range raw.([]interface{}) {
		FeedJsonList = append(FeedJsonList, v.([]byte))
	}
	return FeedJsonList, nil
}

// InitRedis 通过从 DB 获取数据初始化 redis（缓存预热）
func (p *RedisPool) InitRedis(conf *config.CacheConfig, db *gorm.DB) error {
	// 从 DB 中获取数据
	cacheConfig = conf
	var VideoList []*model.Video
	var favoriteCount int64
	var commentCount int64
	err := db.Find(&VideoList).Order("create_time DESC").Limit(cacheConfig.VIDEO_MAX_CACHE_SIZE).Error
	if err != nil {
		return err
	}
	conn := p.NewRedisConn()
	defer conn.Close()
	// 初始化最新的 VIDEO_MAX_CACHE_SIZE（默认值：3000） 条视频信息缓存，评论数，点赞数
	for _, v := range VideoList {
		js, err := json.Marshal(v)
		if err != nil {
			return err
		}
		err = p.AddVideoInfoAndFeed(conn, v.UserId, v.Id, js, v.CreateTime, cacheConfig.VIDEO_CACHE_TTL)
		if err != nil {
			return err
		}

		err = db.Model(&model.Favorite{}).Where(&model.Favorite{VideoId: v.Id}).Count(&favoriteCount).Error
		if err != nil {
			return err
		}
		err = p.SetExInt64(conn, model.Favorite{}.CountCacheKey(v.Id), favoriteCount, cacheConfig.FAVORITE_CACHE_TTL)
		if err != nil {
			return err
		}

		err = db.Model(&model.Comment{}).Where(&model.Comment{VideoId: v.Id}).Count(&commentCount).Error
		if err != nil {
			return err
		}
		err = p.SetExInt64(conn, model.Comment{}.CountCacheKey(v.Id), commentCount, cacheConfig.COMMENT_CACHE_TTL)
		if err != nil {
			return err
		}
	}

	return nil
}
