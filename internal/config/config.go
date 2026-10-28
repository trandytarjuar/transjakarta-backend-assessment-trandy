package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	MQTTBroker     string
	MQTTTopic      string
	MQTTUsername   string
	MQTTPassword   string
	RabbitMQURL    string
	GeofenceLat    float64
	GeofenceLon    float64
	GeofenceRadius float64
	PublisherID    string
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr: env("HTTP_ADDR", ":8080"), DatabaseURL: env("DATABASE_URL", "postgres://fleet:fleet@localhost:5432/fleet?sslmode=disable"),
		MQTTBroker: env("MQTT_BROKER", "tcp://localhost:1883"), MQTTTopic: env("MQTT_TOPIC", "/fleet/vehicle/+/location"),
		MQTTUsername: env("MQTT_USERNAME", ""), MQTTPassword: env("MQTT_PASSWORD", ""),
		RabbitMQURL: env("RABBITMQ_URL", "amqp://fleet:fleet@localhost:5672/"), PublisherID: env("VEHICLE_ID", "TJ-001"),
	}
	var err error
	if c.GeofenceLat, err = floatEnv("GEOFENCE_LAT", -6.21462); err != nil {
		return c, err
	}
	if c.GeofenceLon, err = floatEnv("GEOFENCE_LON", 106.84513); err != nil {
		return c, err
	}
	if c.GeofenceRadius, err = floatEnv("GEOFENCE_RADIUS_METERS", 50); err != nil {
		return c, err
	}
	if c.MQTTUsername == "" || c.MQTTPassword == "" {
		return c, fmt.Errorf("MQTT_USERNAME and MQTT_PASSWORD are required")
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func floatEnv(key string, fallback float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", key, err)
	}
	return n, nil
}

func Retry(ctxDone <-chan struct{}, attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		select {
		case <-ctxDone:
			return fmt.Errorf("startup cancelled: %w", err)
		case <-time.After(delay):
		}
	}
	return err
}
