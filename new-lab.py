#!/usr/bin/env python3
"""Scaffold a new lab in the term5-distributed-systems monorepo.

Creates a lab directory from the hardcoded templates below, registers it in
the root go.work and runs `go mod tidy` to generate go.sum.

Usage:
    python3 new-lab.py lab-2            # HTTP-only lab
    python3 new-lab.py lab-2 --db       # lab with Postgres (migrations, dev compose)
"""

from __future__ import annotations

import argparse
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent
MODULE_ROOT = "github.com/rgu-labs/term5-distributed-systems"

DB_NAME = "app"
GO_VERSION = "go 1.27.0"
PGX_VERSION = "github.com/jackc/pgx/v5 v5.10.0"

NAME_RE = re.compile(r"^[a-z0-9][a-z0-9-]*$")


def die(msg: str) -> None:
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def write(path: pathlib.Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    print(f"  create {path.relative_to(ROOT)}")


def render(template: str, name: str) -> str:
    return (
        template.replace("{MODULE_ROOT}", MODULE_ROOT)
        .replace("{NAME}", name)
        .replace("{GO_VERSION}", GO_VERSION)
        .replace("{PGX_VERSION}", PGX_VERSION)
        .replace("{DB_NAME}", DB_NAME)
    )


# ---------------------------------------------------------------------------
# Templates
# ---------------------------------------------------------------------------

GO_MOD = """module {MODULE_ROOT}/{NAME}

{GO_VERSION}

require {MODULE_ROOT}/lib v0.0.0

replace {MODULE_ROOT}/lib => ../lib
"""

GO_MOD_DB = """module {MODULE_ROOT}/{NAME}

{GO_VERSION}

require (
	{MODULE_ROOT}/lib v0.0.0
	{PGX_VERSION}
)

replace {MODULE_ROOT}/lib => ../lib
"""

MAKEFILE = """.PHONY: dev stop logs build up down test check tidy lint lint-fix install-air

dev:
\t@export $$(grep -v '^#' .env 2>/dev/null | xargs); air

stop:
\tdocker compose down 2>/dev/null

logs:
\tdocker compose logs -f

build:
\tdocker compose build

up:
\tdocker compose up -d

down:
\tdocker compose down

test:
\tgo test -race ./...
\t@echo "all tests passed"

check:
\tgo vet ./...
\tgo test -race ./...
\tgolangci-lint run ./...
\t@echo "check passed"

tidy:
\tgo mod tidy

lint:
\tgolangci-lint run ./...
\t@echo "lint passed"

lint-fix:
\tgolangci-lint run --fix ./...
\t@echo "lint passed"

install-air:
\tgo install github.com/air-verse/air@latest
"""

MAKEFILE_DB = """.PHONY: dev infra stop logs build up down test migrate migrate-create \\
        install-air lint lint-fix db-shell db-reset tidy check

dev: infra
\t@export $$(grep -v '^#' .env 2>/dev/null | xargs); air

infra:
\tdocker compose -f docker-compose.dev.yaml up -d
\t@echo "waiting for db..."
\t@until docker compose -f docker-compose.dev.yaml exec db pg_isready -U app_admin -d {DB_NAME} >/dev/null 2>&1; do sleep 1; done

stop:
\tdocker compose -f docker-compose.dev.yaml down 2>/dev/null

logs:
\tdocker compose -f docker-compose.dev.yaml logs -f

build:
\tdocker compose build

up:
\tdocker compose up -d

down:
\tdocker compose down

db-shell:
\tdocker compose -f docker-compose.dev.yaml exec db psql -U app_admin -d {DB_NAME}

db-reset:
\tdocker compose -f docker-compose.dev.yaml down -v
\tdocker compose -f docker-compose.dev.yaml up -d db
\t@echo "waiting for db..."
\t@until docker compose -f docker-compose.dev.yaml exec db pg_isready -U app_admin -d {DB_NAME} >/dev/null 2>&1; do sleep 1; done
\t$(MAKE) migrate

migrate:
\t@export $$(grep -v '^#' .env 2>/dev/null | xargs); \\
\tif [ -z "$${MIGRATOR_PASSWORD:-}" ]; then echo "error: MIGRATOR_PASSWORD not set in .env"; exit 1; fi; \\
\tdocker compose -f docker-compose.dev.yaml run --rm migrator -path /migrations -database "postgres://migrator:$${MIGRATOR_PASSWORD}@db:5432/{DB_NAME}?sslmode=disable" up

migrate-create:
\t@if [ -z "$(NAME)" ]; then echo "usage: make migrate-create NAME=add_users"; exit 1; fi
\t@sh -c 'LAST=$$(ls migrations/*.up.sql 2>/dev/null | sed "s|.*/||;s|_.*||" | sort -n | tail -1); NEXT=$${LAST:-0}; NEXT=$$((NEXT + 1)); NUM=$$(printf "%06d" $$NEXT); touch "migrations/$${NUM}_$(NAME).up.sql"; touch "migrations/$${NUM}_$(NAME).down.sql"; echo "created migrations/$${NUM}_$(NAME).{up,down}.sql"'

test:
\tgo test -race ./...
\t@echo "all tests passed"

check:
\tgo vet ./...
\tgo test -race ./...
\tgolangci-lint run ./...
\t@echo "check passed"

tidy:
\tgo mod tidy

lint:
\tgolangci-lint run ./...
\t@echo "lint passed"

lint-fix:
\tgolangci-lint run --fix ./...
\t@echo "lint passed"

install-air:
\tgo install github.com/air-verse/air@latest
"""

DOCKERFILE = """FROM golang:1.27-alpine AS builder

WORKDIR /app

# go.mod/go.sum only, so module download is cached before source is copied.
# The lab module is built standalone (its go.mod has a ../lib replace), and
# must not depend on the root go.work which lists every lab.
COPY lib/go.mod lib/go.sum ./lib/
COPY {NAME}/go.mod {NAME}/go.sum ./{NAME}/

RUN cd {NAME} && go mod download

COPY lib/ ./lib/
COPY {NAME}/ ./{NAME}/

RUN cd {NAME} && CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /app/server /app/server

USER app

EXPOSE 8080

CMD ["/app/server"]
"""

DOCKER_COMPOSE = """services:
  app:
    build:
      context: ..
      dockerfile: {NAME}/Dockerfile
    restart: unless-stopped
    environment:
      - ENV=${ENV:-dev}
      - PORT=8080
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8080/healthz || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s
    ports:
      - "127.0.0.1:8080:8080"
"""

DOCKER_COMPOSE_DB = """services:
  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: {DB_NAME}
      POSTGRES_USER: app_admin
      POSTGRES_PASSWORD: ${APP_ADMIN_PASSWORD}
      APP_PASSWORD: ${APP_PASSWORD}
      MIGRATOR_PASSWORD: ${MIGRATOR_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app_admin -d {DB_NAME}"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 10s
    volumes:
      - db_data:/var/lib/postgresql/data
      - ./docker/db/entrypoint-initdb.sh:/docker-entrypoint-initdb.d/entrypoint-initdb.sh
      - ./docker/db/init.sql:/scripts/init.sql:ro
    networks:
      - {NAME}

  migrator:
    image: migrate/migrate:v4.19.1
    volumes:
      - ./migrations:/migrations
    command:
      [
        "-path",
        "/migrations",
        "-database",
        "postgres://migrator:${MIGRATOR_PASSWORD}@db:5432/{DB_NAME}?sslmode=disable",
        "up",
      ]
    depends_on:
      db:
        condition: service_healthy
    networks:
      - {NAME}

  app:
    build:
      context: ..
      dockerfile: {NAME}/Dockerfile
    restart: unless-stopped
    environment:
      - ENV=${ENV:-dev}
      - PORT=8080
      - DATABASE_URL=postgres://app_user:${APP_PASSWORD}@db:5432/{DB_NAME}?sslmode=disable
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8080/readyz || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      migrator:
        condition: service_completed_successfully
    networks:
      - {NAME}

volumes:
  db_data:

networks:
  {NAME}:
"""

DOCKER_COMPOSE_DEV = """services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: {DB_NAME}
      POSTGRES_USER: app_admin
      POSTGRES_PASSWORD: ${APP_ADMIN_PASSWORD}
      APP_PASSWORD: ${APP_PASSWORD}
      MIGRATOR_PASSWORD: ${MIGRATOR_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app_admin -d {DB_NAME}"]
      interval: 5s
      timeout: 5s
      retries: 5
    ports:
      - "${DB_PORT:-5432}:5432"
    volumes:
      - db_data:/var/lib/postgresql/data
      - ./docker/db/entrypoint-initdb.sh:/docker-entrypoint-initdb.d/entrypoint-initdb.sh
      - ./docker/db/init.sql:/scripts/init.sql:ro
    networks:
      - {NAME}

  migrator:
    image: migrate/migrate:v4.19.1
    volumes:
      - ./migrations:/migrations
    command:
      [
        "-path",
        "/migrations",
        "-database",
        "postgres://migrator:${MIGRATOR_PASSWORD}@db:5432/{DB_NAME}?sslmode=disable",
        "up",
      ]
    depends_on:
      db:
        condition: service_healthy
    networks:
      - {NAME}

volumes:
  db_data:

networks:
  {NAME}:
"""

AIR_TOML = """root = "."
tmp_dir = "tmp"
env_files = [".env"]

[build]
  entrypoint = ["./tmp/app"]
  cmd = "go build -o ./tmp/app ./cmd/api"
  delay = 1000
  exclude_dir = ["tmp", "docs", "docker", ".git"]
  include_ext = ["go"]
  send_interrupt = true
  stop_on_error = true

[log]
  time = false

[color]
  main    = "magenta"
  watcher = "cyan"
  build   = "yellow"
  runner  = "green"

[misc]
  clean_on_exit = true
"""

ENV_EXAMPLE = """# Copy to .env and change values before any real deployment.

# App
PORT=8080
ENV=dev
"""

ENV_EXAMPLE_DB = """# Copy to .env and change all secrets before any real deployment.

# Postgres roles (created by docker/db/init.sql on first DB boot)
APP_ADMIN_PASSWORD=change-me-admin
APP_PASSWORD=change-me-app
MIGRATOR_PASSWORD=change-me-migrator

# Postgres host port for local dev only
DB_PORT=5432

# App
# app_user is created by docker/db/init.sql with APP_PASSWORD.
# ${APP_PASSWORD} is expanded by docker compose and by the app itself
# (cmd/api); the resulting password must be URL-safe (no @ : / # ? chars).
DATABASE_URL=postgres://app_user:${APP_PASSWORD}@localhost:5432/{DB_NAME}?sslmode=disable
PORT=8080
ENV=dev
"""

CONFIG_GO = """package main

type Config struct {
	Env  string `default:"dev"`
	Port int    `default:"8080"`
}

func (c *Config) GetEnv() string { return c.Env }
"""

CONFIG_GO_DB = """package main

type Config struct {
	Env         string `default:"dev"`
	Port        int    `default:"8080"`
	DatabaseURL string
}

func (c *Config) GetEnv() string { return c.Env }
"""

MAIN_GO = """package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rgu-labs/term5-distributed-systems/lib/config"
	httpserver "github.com/rgu-labs/term5-distributed-systems/lib/http"
	"github.com/rgu-labs/term5-distributed-systems/lib/log"
	"github.com/rgu-labs/term5-distributed-systems/lib/shutdown"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	var cfg Config
	config.MustLoad(&cfg)

	log.Init(log.Options{
		Level: logLevel(cfg.Env),
		JSON:  cfg.Env == envProd,
		Color: cfg.Env == envDev,
	})

	ctx := context.Background()

	_, srv, _ := httpserver.New(
		httpserver.Config{Addr: fmt.Sprintf(":%d", cfg.Port)},
		httpserver.Default(),
	)

	sm := shutdown.NewManager()
	sm.Register("http", srv.Shutdown)

	go func() {
		log.Info("http server started", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()

	sm.RunWithTimeout(ctx, 15*time.Second)
}

func logLevel(env string) log.Level {
	if env == envDev {
		return log.LevelDebug
	}
	return log.LevelInfo
}
"""

MAIN_GO_DB = """package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/rgu-labs/term5-distributed-systems/lib/config"
	httpserver "github.com/rgu-labs/term5-distributed-systems/lib/http"
	"github.com/rgu-labs/term5-distributed-systems/lib/log"
	"github.com/rgu-labs/term5-distributed-systems/lib/pg"
	"github.com/rgu-labs/term5-distributed-systems/lib/retry"
	"github.com/rgu-labs/term5-distributed-systems/lib/shutdown"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	var cfg Config
	config.MustLoad(&cfg)

	log.Init(log.Options{
		Level: logLevel(cfg.Env),
		JSON:  cfg.Env == envProd,
		Color: cfg.Env == envDev,
	})

	ctx := context.Background()

	pool, err := pg.New(ctx, pg.Config{DSN: expandEnvRefs(cfg.DatabaseURL)})
	if err != nil {
		log.Error("create pg pool", "err", err)
		os.Exit(1)
	}
	if err := retry.Do(ctx, func() error { return pool.Ping(ctx) }, 5, 500*time.Millisecond); err != nil {
		pool.Close()
		log.Error("connect to postgres", "err", err)
		os.Exit(1)
	}

	_, srv, health := httpserver.New(
		httpserver.Config{Addr: fmt.Sprintf(":%d", cfg.Port)},
		httpserver.Default(),
	)
	health.Check("postgres", pg.HealthCheck(pool))

	sm := shutdown.NewManager()
	sm.Register("pg", func(context.Context) error {
		pool.Close()
		return nil
	})
	sm.Register("http", srv.Shutdown)

	go func() {
		log.Info("http server started", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()

	sm.RunWithTimeout(ctx, 15*time.Second)
}

func logLevel(env string) log.Level {
	if env == envDev {
		return log.LevelDebug
	}
	return log.LevelInfo
}

var envRefPattern = regexp.MustCompile(`\\$\\{([A-Za-z_][A-Za-z0-9_]*)\\}`)

func expandEnvRefs(s string) string {
	return envRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1]
		value, ok := os.LookupEnv(name)
		if !ok {
			return match
		}
		return value
	})
}
"""

ENTRYPOINT_INITDB = """#!/bin/sh
set -e

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \\
  -v app_password="$APP_PASSWORD" \\
  -v migrator_password="$MIGRATOR_PASSWORD" \\
  -f /scripts/init.sql
"""

INIT_SQL = """CREATE ROLE app_user WITH LOGIN PASSWORD :'app_password';
CREATE ROLE migrator WITH LOGIN PASSWORD :'migrator_password';

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE CONNECT ON DATABASE {DB_NAME} FROM PUBLIC;

GRANT CONNECT ON DATABASE {DB_NAME} TO app_admin, app_user, migrator;

GRANT USAGE, CREATE ON SCHEMA public TO migrator;
GRANT USAGE ON SCHEMA public TO app_user;

ALTER DEFAULT PRIVILEGES FOR ROLE migrator IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;
ALTER DEFAULT PRIVILEGES FOR ROLE migrator IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO app_user;
"""


# ---------------------------------------------------------------------------
# Scaffolding
# ---------------------------------------------------------------------------

def file_map(name: str, db: bool) -> dict[str, str]:
    files = {
        "go.mod": GO_MOD_DB if db else GO_MOD,
        "Makefile": MAKEFILE_DB if db else MAKEFILE,
        "Dockerfile": DOCKERFILE,
        "docker-compose.yaml": DOCKER_COMPOSE_DB if db else DOCKER_COMPOSE,
        ".air.toml": AIR_TOML,
        ".env.example": ENV_EXAMPLE_DB if db else ENV_EXAMPLE,
        "cmd/api/main.go": MAIN_GO_DB if db else MAIN_GO,
        "cmd/api/config.go": CONFIG_GO_DB if db else CONFIG_GO,
    }
    if db:
        files["docker-compose.dev.yaml"] = DOCKER_COMPOSE_DEV
        files["docker/db/entrypoint-initdb.sh"] = ENTRYPOINT_INITDB
        files["docker/db/init.sql"] = INIT_SQL
        # migrate refuses to run on a directory without any migration files,
        # so scaffold an empty initial pair.
        files["migrations/000001_init.up.sql"] = ""
        files["migrations/000001_init.down.sql"] = ""
    return files


def add_to_go_work(name: str) -> None:
    path = ROOT / "go.work"
    lines = path.read_text(encoding="utf-8").splitlines()

    start = next((i for i, ln in enumerate(lines) if ln.strip() == "use ("), None)
    if start is None:
        die(f"{path}: no 'use (' block found")
    end = next((i for i, ln in enumerate(lines[start:], start) if ln.strip() == ")"), None)
    if end is None:
        die(f"{path}: unclosed 'use (' block")

    entry = f"./{name}"
    entries = [ln.strip() for ln in lines[start + 1 : end] if ln.strip()]
    if entry in entries:
        return
    entries.append(entry)
    entries.sort()

    lines[start : end + 1] = ["use ("] + [f"\t{e}" for e in entries] + [")"]
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"  register {entry} in go.work")


def run_tidy(lab_dir: pathlib.Path) -> None:
    proc = subprocess.run(
        ["go", "mod", "tidy"],
        cwd=str(lab_dir),
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        print(f"warning: 'go mod tidy' failed in {lab_dir}:", file=sys.stderr)
        print(proc.stderr, file=sys.stderr)
        print("run 'go mod tidy' manually after finishing setup", file=sys.stderr)
        return
    print("  go mod tidy ok")


def main() -> int:
    parser = argparse.ArgumentParser(
        prog="new-lab.py",
        description="Scaffold a new lab from templates and register it in go.work.",
    )
    parser.add_argument("name", help="lab directory name, e.g. lab-2")
    parser.add_argument(
        "--db",
        action="store_true",
        help="include Postgres: dev docker-compose, role init SQL, migrations",
    )
    args = parser.parse_args()

    name = args.name.strip().lower()
    if not NAME_RE.match(name):
        die(f"invalid name {name!r} (want [a-z0-9][a-z0-9-]*)")
    dest = ROOT / name
    if dest.exists():
        die(f"directory {name} already exists")

    print(f"scaffolding {name}/ (db={'yes' if args.db else 'no'})")
    for rel, content in file_map(name, args.db).items():
        write(dest / rel, render(content, name))

    add_to_go_work(name)

    run_tidy(dest)

    print(f"\ncreated {name}/")
    print("next steps:")
    print(f"  cd {name}")
    print("  cp .env.example .env")
    print("  make dev        # infra (db) + air")
    if args.db:
        print("  make migrate    # apply migrations")
    return 0


if __name__ == "__main__":
    sys.exit(main())