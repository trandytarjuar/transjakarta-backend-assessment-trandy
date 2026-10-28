package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"transjakarta-fleet/internal/config"
	"transjakarta-fleet/internal/database"
	"transjakarta-fleet/internal/handler"
	"transjakarta-fleet/internal/messaging"
	"transjakarta-fleet/internal/repository"
	"transjakarta-fleet/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration invalid", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	conn, ch, err := messaging.Setup(cfg.RabbitMQURL)
	if err != nil {
		slog.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	defer ch.Close()
	repo := repository.NewPostgresLocationRepository(db)
	geo := service.NewGeofenceService(cfg.GeofenceLat, cfg.GeofenceLon, cfg.GeofenceRadius)
	locations := service.NewLocationService(repo, geo, messaging.NewRabbitPublisher(ch))
	if _, err = messaging.StartMQTTSubscriber(ctx, cfg.MQTTBroker, cfg.MQTTTopic, cfg.MQTTUsername, cfg.MQTTPassword, locations); err != nil {
		slog.Error("mqtt subscriber failed", "error", err)
		os.Exit(1)
	}
	app := fiber.New(fiber.Config{ErrorHandler: handler.ErrorHandler, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second})
	handler.New(locations).Register(app)
	go func() {
		<-ctx.Done()
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = app.ShutdownWithContext(shutdownCtx)
	}()
	slog.Info("api started", "address", cfg.HTTPAddr)
	if err = app.Listen(cfg.HTTPAddr); err != nil && ctx.Err() == nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
