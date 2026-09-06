.PHONY: dev stop logs build up down test check tidy lint lint-fix install-air

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