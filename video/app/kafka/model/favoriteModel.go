package model

import "strconv"

// Favorite 表结构
type Favorite struct {
	UserId     uint64 `gorm:"column:user_id"`
	VideoId    uint64 `gorm:"column:video_id"`
	CreateTime int64  `gorm:"column:create_time"`
}

func (Favorite) TableName() string {
	return "favorite"
}

const (
	FavorCacheKeyPrefix      = "Fav:UserId:VideoId:ZSET:"
	FavorCountCacheKeyPrefix = "Fav:VideoId:FavoriteCount:"
	FavorDelCacheKeyPrefix   = "Fav:UserId:DelFavoriteVideoId:SET:"
)

// CacheKey 返回 用户最近点赞视频 对应的缓存 key 名称，
// Favorite 缓存类型为 zset 类型，key: UserId:VideoId:ZSET:{用户id}, members:{视频id} score: 点赞时间
// 为用户维护一个最新点赞的 30 个视频 id 的有序集合，假设有 1000 万用户的有序集合存在会占用约 6G 内存空间
// 我们为每个用户的缓存设置默认 12 小时的过期时间，用于淘汰不活跃用户的缓存
func (Favorite) CacheKey(userId uint64) string {
	return FavorCacheKeyPrefix + strconv.FormatUint(userId, 10)
}

// CountCacheKey 返回 视频点赞数 对应的缓存 key 名称，
// Favorite Count 缓存类型为 string 类型，key: VideoId:FavoriteCount:{视频id} value: 点赞数
// 我们为每个用户的缓存设置默认 12 小时的过期时间，用于淘汰不活跃用户的缓存
func (Favorite) CountCacheKey(videoId uint64) string {
	return FavorCountCacheKeyPrefix + strconv.FormatUint(videoId, 10)
}

// DelCacheKey 返回 已取消点赞 对应的缓存 key 名称，
// Favorite Del 缓存类型为 set 类型，key: UserId:DelFavoriteVideoId:SET:{用户id} member: {视频id}
// 为用户维护一个最近取消点赞的视频 id 的集合
// 这是由于异步写库，会导致用户取消点赞之后再次刷新同视频时，因为缓存查不到用户最近对该视频的点赞之后直接去查库，得到仍在点赞状态的脏信息
// 为了防止出现这种状况，我们为每个用户维护一个最近取消点赞视频的集合
// 在取消点赞时将视频 id 加入集合，而写入数据库后会将该 id 从集合中去除
// 缓存设置 5 分钟的过期时间，5 分钟未被访问则销毁，这里的销毁当集合为空时也应执行删除操作
// 如果 5 分钟未访问且尚未因空集而被销毁，大概率是消息消费失败了，可以试着重试消息，但我暂时就不实现了~
func (Favorite) DelCacheKey(userId uint64) string {
	return FavorDelCacheKeyPrefix + strconv.FormatUint(userId, 10)
}
