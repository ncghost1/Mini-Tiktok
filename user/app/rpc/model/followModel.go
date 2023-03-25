package model

import "strconv"

type Follow struct {
	Follower   uint64 `gorm:"column:follower_id"`
	Following  uint64 `gorm:"column:following_id"`
	CreateTime int64  `gorm:"column:create_time"`
}

const (
	FollowListCacheKeyPrefix   = "FollowList:Userid:FollowId:"
	FollowerListCacheKeyPrefix = "FollowerList:Userid:FollowId:"
)

func (Follow) TableName() string {
	return "follow"
}

// FollowListCacheKey 返回指定 userid 的最近关注列表缓存 key 名称
// FollowList 缓存使用的是 Redis Zset 结构，member 存储关注用户 id，score 为关注时间戳
// 默认为每个用户维护最多 30 位关注用户列表，过期时间: 1h
func (Follow) FollowListCacheKey(userid uint64) string {
	return FollowListCacheKeyPrefix + strconv.FormatUint(userid, 10)
}

// FollowerListCacheKey 返回指定 userid 的最近粉丝列表缓存 key 名称
// FollowerList 缓存使用的是 Redis Zset 结构，member 存储粉丝用户 id，score 为关注时间戳
// 默认为每个用户维护最多 30 位粉丝用户列表，过期时间: 1h
func (Follow) FollowerListCacheKey(userid uint64) string {
	return FollowerListCacheKeyPrefix + strconv.FormatUint(userid, 10)
}
