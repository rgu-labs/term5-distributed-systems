package main

import (
	"context"
	"log"
	"time"

	"github.com/rgu-labs/term5-distributed-systems/lab-3/internal/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	client, err := kgo.NewClient(kgo.SeedBrokers(kafka.Seeds...))
	if err != nil {
		panic(err)
	}

	defer client.Close()

	ctx := context.Background()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		record := &kgo.Record{Topic: "Ping", Value: []byte("Ping!")}
		client.Produce(ctx, record, func(r *kgo.Record, err error) {
			if err != nil {
				log.Printf("record had a produce error: %v\n", err)
			}
			log.Printf("Ping sent to topic %s, partition %d, offset %d\n", r.Topic, r.Partition, r.Offset)
		})
	}
}
