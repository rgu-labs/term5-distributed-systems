.PHONY: dev stop logs build up down down-v test check tidy lint lint-fix install-air \
	kafka-topics kafka-consume

dev:
	@export $$(grep -v '^#' .env 2>/dev/null | xargs); air

stop:
	docker compose down 2>/dev/null

logs:
	docker compose logs -f

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

down-v:
	docker compose down -v

kafka-topics:
	docker compose exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

kafka-consume:
	docker compose exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
		--bootstrap-server localhost:9092 --topic $(or $(TOPIC),Ping) --from-beginning

test:
	go test -race ./...
	@echo "all tests passed"

check:
	go vet ./...
	go test -race ./...
	golangci-lint run ./...
	@echo "check passed"

tidy:
	go mod tidy

lint:
	golangci-lint run ./...
	@echo "lint passed"

lint-fix:
	golangci-lint run --fix ./...
	@echo "lint passed"

install-air:
	go install github.com/air-verse/air@latest
