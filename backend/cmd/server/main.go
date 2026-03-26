package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"kubeclaw/backend/internal/app"
	"kubeclaw/backend/internal/config"
	"kubeclaw/backend/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	zapLogger, err := logger.Init(cfg)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer zapLogger.Sync()

	application, err := app.New(cfg)
	if err != nil {
		logger.S().Fatalw("bootstrap app failed", "error", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		logger.S().Fatalw("run app failed", "error", err)
	}
}
