package main

import (
	"Mini-Tiktok/publish/app/kafka/internal/config"
	"Mini-Tiktok/publish/app/kafka/internal/logic"
	"Mini-Tiktok/publish/app/kafka/internal/svc"
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	kafka "github.com/segmentio/kafka-go"
)

var configFile = flag.String("f", "etc/publish.yaml", "the config file")

func getKafkaReader(kafkaURL, topic, groupID string, minBytes, maxBytes int) *kafka.Reader {
	brokers := strings.Split(kafkaURL, ",")
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: minBytes,
		MaxBytes: maxBytes,
	})
}

func main() {
	var c config.Config
	config.MustLoad(*configFile, &c)
	reader := getKafkaReader(c.KafkaConfig.Host, c.KafkaConfig.Topic, c.KafkaConfig.GroupId, c.KafkaConfig.MinBytes, c.KafkaConfig.MaxBytes)
	defer reader.Close()
	svcctx := svc.NewServiceContext(c)
	l := logic.NewTranscodingLogic(context.Background(), svcctx)
	fmt.Println("TransCoding Service Start...")
	fmt.Println("start consuming ...")
	err := reader.SetOffset(kafka.LastOffset)
	if err != nil {
		return
	}

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
		err = l.TransCoding(string(m.Key), m.Value)
		if err != nil {
			log.Println(err)
		}

		fmt.Println("TransCoding completed...")
	}
}
