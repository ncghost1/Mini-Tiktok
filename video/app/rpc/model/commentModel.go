package model

import "strconv"

// Comment 表结构
type Comment struct {
	Id         uint64 `json:"id" gorm:"column:id"`
	UserId     uint64 `json:"user_id" gorm:"column:user_id"`
	VideoId    uint64 `json:"video_id" gorm:"column:video_id"`
	Content    string `json:"content" gorm:"column:content"`
	CreateTime int64  `json:"createTime" gorm:"column:create_time"`
}

const (
	ComCacheKeyPrefix      = "Com:CommentId:CommentJson:"
	ComIdCacheKeyPrefix    = "Com:VideoId:CommentId:ZSET:"
	ComCountCacheKeyPrefix = "Com:VideoId:CommentCount:"
)

func (Comment) TableName() string {
	return "comment"
}

// CacheKey 返回 Comment（视频评论）对应的缓存 key 名称，
// CacheKey 缓存类型为 string 类型，key: CommentId:Comment:{评论id}, value: 评论信息json
// 为视频维护一个最新的 30 条评论信息 json 缓存，
// 我们为每个缓存设置默认 12 小时的过期时间，用于淘汰冷门视频的评论缓存
func (Comment) CacheKey(commentId uint64) string {
	return ComCacheKeyPrefix + strconv.FormatUint(commentId, 10)
}

// IdCacheKey 返回 Comment（视频评论id）对应的缓存 key 名称，
// Comment Id 缓存类型为 zset 类型，key: VideoId:CommentId:ZSET:{视频id},members:{评论id} score: 创建时间
// 为视频维护一个最新的 30 条评论 id 的有序集合，
// 我们为每个缓存设置默认 12 小时的过期时间，用于淘汰冷门视频的评论缓存
func (Comment) IdCacheKey(videoId uint64) string {
	return ComIdCacheKeyPrefix + strconv.FormatUint(videoId, 10)
}

// CountCacheKey 返回 Comment Count （视频评论数）对应的缓存 key 名称，
// Comment Count 缓存类型为 string 类型，key: VideoId:CommentCount:{视频id} value: 评论数
// 我们为每个缓存设置默认 12 小时的过期时间，用于淘汰冷门视频的评论缓存
func (Comment) CountCacheKey(videoId uint64) string {
	return ComCountCacheKeyPrefix + strconv.FormatUint(videoId, 10)
}
