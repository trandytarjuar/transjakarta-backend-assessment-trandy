package main

import (
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
	"transjakarta-fleet/internal/config"
	"transjakarta-fleet/internal/models"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration invalid", "error", err)
		os.Exit(1)
	}
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID("mock-gps-publisher").
		SetUsername(cfg.MQTTUsername).
		SetPassword(cfg.MQTTPassword).
		SetAutoReconnect(true).
		SetConnectRetry(true)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		slog.Error("mqtt connection failed", "error", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	step := 0
	topic := "/fleet/vehicle/" + cfg.PublisherID + "/location"
	slog.Info("mock publisher started", "topic", topic)
	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			distance := 120.0 - float64(step%13)*15.0
			bearing := math.Pi / 2
			lat := cfg.GeofenceLat + (distance*math.Cos(bearing))/111320
			lon := cfg.GeofenceLon + (distance*math.Sin(bearing))/(111320*math.Cos(cfg.GeofenceLat*math.Pi/180))
			payload, err := json.Marshal(models.Location{VehicleID: cfg.PublisherID, Latitude: lat, Longitude: lon, Timestamp: now.UTC()})
			if err != nil {
				slog.Error("gps serialization failed", "error", err)
				continue
			}
			t := client.Publish(topic, 1, false, payload)
			if !t.WaitTimeout(5 * time.Second) {
				slog.Warn("mqtt publish timed out; client will retry after reconnect", "vehicle_id", cfg.PublisherID)
				continue
			}
			if t.Error() != nil {
				slog.Error("mqtt publish failed", "error", t.Error())
			} else {
				slog.Info("gps published", "vehicle_id", cfg.PublisherID, "distance_hint_m", distance)
			}
			step++
		}
	}
}
