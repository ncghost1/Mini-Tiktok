package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
)

type Config struct {
	DbConfig    DbConfig        `yaml:"DbConfig"`
	KafkaConfig KafkaConfig     `yaml:"KafkaConfig"`
	RedisConfig RedisConfig     `yaml:"RedisConfig"`
	AliyunOss   AliyunOssConfig `yaml:"AliyunOss"`
	WorkerId    uint32          `yaml:"WorkerId"`
	CacheConfig struct {
		FEED_MAX_CACHE_SIZE int
		VIDEO_CACHE_TTL     int
	}
}

type KafkaConfig struct {
	Host     string `yaml:"Host"`
	Topic    string `yaml:"Topic"`
	GroupId  string `yaml:"GroupId"`
	MinBytes int    `yaml:"MinBytes"`
	MaxBytes int    `yaml:"MaxBytes"`
}

type RedisConfig struct {
	Host        string `yaml:"Host"`
	Port        int    `yaml:"Port"`
	Username    string `yaml:"Username"`
	Password    string `yaml:"Password"`
	Auth        bool   `yaml:"Auth"`
	MaxIdle     int    `yaml:"MaxIdle"`
	Active      int    `yaml:"Active"`
	IdleTimeout int    `yaml:"IdleTimeout"`
}

type AliyunOssConfig struct {
	Endpoint        string `yaml:"Endpoint"`
	AccessKeyId     string `yaml:"AccessKeyId"`
	AccessKeySecret string `yaml:"AccessKeySecret"`
	Bucket          string `yaml:"Bucket"`
	VideoPath       string `yaml:"VideoPath"`
	CoverPath       string `yaml:"CoverPath"`
	UrlPrefix       string `yaml:"UrlPrefix"`
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

func MustLoad(configPath string, c *Config) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(content, &c)
	if err != nil {
		panic(err)
	}
}
