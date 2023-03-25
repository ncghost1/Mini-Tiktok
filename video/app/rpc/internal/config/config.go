package config

import (
	"github.com/zeromicro/go-zero/zrpc"
	"strconv"
)

type Config struct {
	zrpc.RpcServerConf
	UserRpc     zrpc.RpcClientConf
	KafkaConfig struct {
		Host         string
		Topic        string
		BatchTimeout int
		BatchSize    int
		BatchBytes   int64
	}
	DbConfig    DbConfig
	RedisConfig struct {
		Host        string
		Port        int
		Username    string
		Password    string
		Auth        bool
		MaxIdle     int
		Active      int
		IdleTimeout int
	}
	CacheConfig CacheConfig
	WorkerId    uint32
}

type DbConfig struct {
	Path         string `json:"path" yaml:"path"`                     // 服务器地址
	Port         int    `json:"port" yaml:"port"`                     //:端口
	Config       string `json:"config" yaml:"config"`                 // 高级配置
	Dbname       string `json:"db-name" yaml:"db-name"`               // 数据库名
	Username     string `json:"username" yaml:"username"`             // 数据库用户名
	Password     string `json:"password" yaml:"password"`             // 数据库密码
	MaxIdleConns int    `json:"max-idle-conns" yaml:"max-idle-conns"` // 空闲中的最大连接数
	MaxOpenConns int    `json:"max-open-conns" yaml:"max-open-conns"` // 打开到数据库的最大连接数
}

type Mysql struct {
	DbConfig
}

// Dsn 获取 Database Source Name
func (m *Mysql) Dsn() string {
	return m.Username + ":" + m.Password + "@tcp(" + m.Path + ":" + strconv.FormatInt(int64(m.Port), 10) + ")/" + m.Dbname + "?" + m.Config
}

type CacheConfig struct {
	VIDEO_CACHE_TTL               int
	FAVORITE_CACHE_TTL            int
	COMMENT_CACHE_TTL             int
	FAVORITE_DEL_CACHE_TTL        int
	FEED_MAX_CACHE_SIZE           int
	VIDEO_MAX_CACHE_SIZE          int
	VIDEO_FAVORITE_MAX_CACHE_SIZE int
	VIDEO_COMMENT_MAX_CACHE_SIZE  int
}
