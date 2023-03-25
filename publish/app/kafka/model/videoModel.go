package model

import "strconv"

// Video 表结构
type Video struct {
	Id         uint64 `gorm:"column:id"`
	UserId     string `gorm:"column:user_id"`
	Title      string `gorm:"column:title"`
	PlayUrl    string `gorm:"column:play_url"`
	CoverUrl   string `gorm:"column:cover_url"`
	CreateTime int64  `gorm:"column:create_time"`
}

func (Video) TableName() string {
	return "video"
}

// CacheKey 返回 Video 对应的缓存 key 名称，
// Video 缓存类型为 string 类型，key: VideoId:VideoInfo:{视频id} value: 视频信息 json
func (Video) CacheKey(videoId uint64) string {
	return "VideoId:VideoInfo:" + strconv.FormatUint(videoId, 10)
}

// FeedCacheKey 返回 Feed 流 对应的缓存 key 名称，
// Feed 缓存类型为 zset 类型，key: Feed, member: 视频信息 json, score: 视频时间戳
// Video 缓存存储了视频信息，而 Feed 缓存也选择冗余存储了视频信息，
// 是因为 Feed 缓存默认配置的数值不大（3000条最新视频），所以可以放心进行冗余存储
func (Video) FeedCacheKey() string {
	return "Feed"
}
