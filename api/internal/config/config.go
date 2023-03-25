package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	UserRpc   zrpc.RpcClientConf
	JwtRpc    zrpc.RpcClientConf
	VideoRpc  zrpc.RpcClientConf
	AliyunOss struct {
		Endpoint        string
		AccessKeyId     string
		AccessKeySecret string
		VideoBucket     string
		VideoPath       string
	}
	JwtConfig struct {
		AccessExpire int64
	}
	KafkaConfig struct {
		Host         string
		Topic        string
		BatchTimeout int
		BatchSize    int
		BatchBytes   int64
	}
	FeedLimit int64
}
