#!/usr/bin/env bash
set -euo pipefail

bootstrap="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
partitions="${TOPIC_PARTITIONS:-1}"
topics="${KAFKA_TOPICS:-Ping Pong}"

for topic in $topics; do
  echo "===> Creating topic $topic (partitions=$partitions)"
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server "$bootstrap" \
    --create --if-not-exists --topic "$topic" \
    --partitions "$partitions" \
    --replication-factor 1
done
