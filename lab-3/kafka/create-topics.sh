#!/usr/bin/env bash
set -euo pipefail

bootstrap="${KAFKA_BOOTSTRAP_SERVERS}"
partitions="${TOPIC_PARTITIONS}"
topics="${KAFKA_TOPICS}"

for topic in $topics; do
  echo "===> Creating topic $topic (partitions=$partitions)"
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server "$bootstrap" \
    --create --if-not-exists --topic "$topic" \
    --partitions "$partitions" \
    --replication-factor 1
done
