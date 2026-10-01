FROM apache/kafka:4.3.1

USER root

COPY server.properties /mnt/shared/config/server.properties
COPY create-topics.sh /usr/local/bin/kafka-create-topics

RUN chmod 0755 /usr/local/bin/kafka-create-topics

USER appuser

ENV KAFKA_LOG4J_ROOT_LOGLEVEL=WARN

HEALTHCHECK --interval=10s --timeout=10s --start-period=45s --retries=10 \
    CMD /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1 || exit 1
