package main

import (
	"flag"
	"fmt"

	"Mini-Tiktok/video/app/rpc/internal/config"
	"Mini-Tiktok/video/app/rpc/internal/server"
	"Mini-Tiktok/video/app/rpc/internal/svc"
	"Mini-Tiktok/video/app/rpc/video"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/video.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	err := ctx.Redis.InitRedis(&c.CacheConfig, ctx.Db)
	if err != nil {
		panic(err)
	}
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		video.RegisterVideoRpcServer(grpcServer, server.NewVideoRpcServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
