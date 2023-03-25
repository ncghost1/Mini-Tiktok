package model

import "strconv"

// Video 表结构
type Video struct {
	Id         uint64 `json:"id" gorm:"column:id"`
	UserId     uint64 `json:"user_id" gorm:"column:user_id"`
	Title      string `json:"title" gorm:"column:title"`
	PlayUrl    string `json:"play_url" gorm:"column:play_url"`
	CoverUrl   string `json:"cover_url" gorm:"column:cover_url"`
	CreateTime int64  `json:"create_time" gorm:"column:create_time"`
}

const (
	VideoCacheKeyPrefix       = "Vid:VideoId:VideoInfo:"
	PublishListCacheKeyPrefix = "Vid:UserId:VideoId:ZSET:"
	FeedCacheKey              = "Feed"
)

func (Video) TableName() string {
	return "video"
}

// CacheKey 返回 Video 对应的缓存 key 名称，
// Video 缓存类型为 string 类型，key: VideoId:VideoInfo:{视频id} value: 视频信息 json
// 默认过期时间：12小时
func (Video) CacheKey(videoId uint64) string {
	return VideoCacheKeyPrefix + strconv.FormatUint(videoId, 10)
}

// PublishListCacheKey 返回用户最新发布视频列表对应的缓存 key 名称，
// PublishList Cache 缓存类型为 ZSET 类型，key: UserId:VideoId:ZSET:{用户id}, member: {视频id}, score: 视频时间戳
// 默认过期时间：12小时
func (Video) PublishListCacheKey(userId uint64) string {
	return PublishListCacheKeyPrefix + strconv.FormatUint(userId, 10)
}

// FeedCacheKey 返回 Feed 流 对应的缓存 key 名称，
// Feed 缓存类型为 zset 类型，key: Feed, member: 视频信息 json, score: 视频时间戳
// Video 缓存存储了视频信息，而 Feed 缓存也选择冗余存储了视频信息，
// 是因为 Feed 缓存默认配置的数值不大（3000条最新视频），所以可以放心进行冗余存储
func (Video) FeedCacheKey() string {
	return FeedCacheKey
}
