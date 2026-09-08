package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	httpdelivery "github.com/rgu-labs/term5-distributed-systems/lab-1/internal/delivery/http"
	tcpdelivery "github.com/rgu-labs/term5-distributed-systems/lab-1/internal/delivery/tcp"
	"github.com/rgu-labs/term5-distributed-systems/lab-1/internal/service"
	"github.com/rgu-labs/term5-distributed-systems/lib/config"
	httpserver "github.com/rgu-labs/term5-distributed-systems/lib/http"
	"github.com/rgu-labs/term5-distributed-systems/lib/http/middleware"
	"github.com/rgu-labs/term5-distributed-systems/lib/log"
	"github.com/rgu-labs/term5-distributed-systems/lib/shutdown"
	tcpserver "github.com/rgu-labs/term5-distributed-systems/lib/tcp"
)

const (
	serverTypeHTTP = "http"
	serverTypeTCP  = "tcp"

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
	sm := shutdown.NewManager()
	blurSvc := service.NewBlur()

	switch cfg.ServerType {
	case serverTypeHTTP:
		bootstrapHTTPServer(cfg, ctx, sm, blurSvc)
	case serverTypeTCP:
		bootstrapTCPServer(cfg, ctx, sm, blurSvc)
	default:
		log.Error("invalid server type", "type", cfg.ServerType)
		os.Exit(1)
	}

	sm.RunWithTimeout(ctx, 15*time.Second)
}

func logLevel(env string) log.Level {
	if env == envDev {
		return log.LevelDebug
	}
	return log.LevelInfo
}

func bootstrapHTTPServer(cfg Config, ctx context.Context, sm *shutdown.Manager, blurSvc *service.BlurService) {
	mux, srv, _ := httpserver.New(
		httpserver.Config{Addr: fmt.Sprintf(":%d", cfg.Port)},
		httpserver.Default(),
	)

	mux.Handle(
		"POST /api/blur",
		middleware.LimitBody(10<<20)(httpdelivery.NewBlurHandler(blurSvc)),
	)

	sm.Register("http", srv.Shutdown)

	go func() {
		log.Info("http server started", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()
}

func bootstrapTCPServer(cfg Config, ctx context.Context, sm *shutdown.Manager, blurSvc *service.BlurService) {
	srv, _ := tcpserver.New(
		tcpserver.Config{Addr: fmt.Sprintf(":%d", cfg.Port)},
		tcpserver.Default(),
	)

	sm.Register("tcp", srv.Shutdown)

	go func() {
		log.Info("tcp server started", "addr", srv.Addr(), "env", cfg.Env)
		srv.Start(tcpdelivery.NewBlurHandler(blurSvc))
	}()
}
