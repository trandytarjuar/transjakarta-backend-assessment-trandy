package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"transjakarta-fleet/internal/config"
	"transjakarta-fleet/internal/messaging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration invalid", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	conn, ch, err := messaging.Setup(cfg.RabbitMQURL)
	if err != nil {
		slog.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	defer ch.Close()
	slog.Info("geofence worker started")
	if err = messaging.Consume(ctx, ch); err != nil && ctx.Err() == nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
