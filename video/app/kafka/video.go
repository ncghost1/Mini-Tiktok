package main

import (
	"Mini-Tiktok/video/app/kafka/internal/config"
	"Mini-Tiktok/video/app/kafka/internal/logic"
	"Mini-Tiktok/video/app/kafka/internal/svc"
	"context"
	"flag"
	"fmt"
	"log"
)

var configFile = flag.String("f", "etc/video.yaml", "the config file")

func main() {
	var c config.Config
	config.MustLoad(*configFile, &c)
	svcctx := svc.NewServiceContext(c)
	l := logic.NewWriteDbLogic(context.Background(), svcctx)
	fmt.Println("Database Writer Service Start...")
	fmt.Println("start consuming ...")
	logic.InitModels()
	for {
		m, err := svcctx.KafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalln(err)
		}

		// 消息格式：value: logic.MsgInfo , json 结构
		fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
		err = l.WriteDb(m.Value)
		if err != nil {
			log.Println(err)
		}
		fmt.Println("Write to database succeeded...")
	}
}
