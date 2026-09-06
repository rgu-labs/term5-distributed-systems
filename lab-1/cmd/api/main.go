package main

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
