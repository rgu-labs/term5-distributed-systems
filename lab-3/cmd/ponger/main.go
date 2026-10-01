package main

import (
	"context"
	"log"

	"github.com/rgu-labs/term5-distributed-systems/lab-3/internal/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(kafka.Seeds...),
		kgo.ConsumerGroup("Pongers"),
		kgo.ConsumeTopics("Ping"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)

	if err != nil {
		panic(err)
	}

	defer client.Close()

	ctx := context.Background()

	for {
		fetches := client.PollFetches(ctx)

		fetches.EachError(func(topic string, partition int32, err error) {
			log.Printf("fetch error in %s topic, %d partition: %v\n", topic, partition, err)
		})

		fetches.EachRecord(func(record *kgo.Record) {
			log.Printf("got: %s from %s topic, %d partition at %d offset", record.Value, record.Topic, record.Partition, record.Offset)
		})

	}
}
