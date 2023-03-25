package model

import "strconv"

// User 表结构
type User struct {
	Id       uint64 `gorm:"column:id"`
	Username string `gorm:"column:username"`
	Password string `gorm:"column:password"`
}

const (
	UserCacheKeyPrefix = "User:Userid:UserInfo:Hash"
	UsernameField      = "username"
	FollowCountField   = "followCount"
	FollowerCountField = "followerCount"
)

func (User) TableName() string {
	return "user"
}

// CacheKey 返回 user 对应的缓存 key 名称，
// user 缓存使用的是 Redis Hash 结构，field-value 存储 username,follow count,follower count
// 默认过期时间： 12h
// 超过 10w 粉丝或关注数的用户将不设置过期时间
func (User) CacheKey(userid uint64) string {
	return UserCacheKeyPrefix + strconv.FormatUint(userid, 10)
}
