package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
	"transjakarta-fleet/internal/models"
	"transjakarta-fleet/internal/service"
)

func VehicleIDFromTopic(topic string) (string, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 || parts[0] != "" || parts[1] != "fleet" || parts[2] != "vehicle" || parts[3] == "" || parts[4] != "location" {
		return "", fmt.Errorf("invalid vehicle location topic %q", topic)
	}
	return parts[3], nil
}

func DecodeLocationPayload(payload []byte) (models.Location, error) {
	var location models.Location
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&location); err != nil {
		return location, fmt.Errorf("decode location payload: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return location, fmt.Errorf("decode location payload: trailing data")
	}
	if err := service.Validate(location); err != nil {
		return location, err
	}
	return location, nil
}

func StartMQTTSubscriber(ctx context.Context, broker, topic, username, password string, locations *service.LocationService) (mqtt.Client, error) {
	handler := func(_ mqtt.Client, msg mqtt.Message) {
		vehicleID, err := VehicleIDFromTopic(msg.Topic())
		if err != nil {
			slog.Warn("rejected invalid mqtt topic", "topic", msg.Topic())
			return
		}
		l, err := DecodeLocationPayload(msg.Payload())
		if err != nil {
			slog.Warn("rejected invalid mqtt json", "error", err)
			return
		}
		if l.VehicleID != vehicleID {
			slog.Warn("rejected vehicle/topic mismatch", "topic_vehicle", vehicleID, "payload_vehicle", l.VehicleID)
			return
		}
		if err := locations.Process(ctx, l); err != nil {
			slog.Error("location processing failed", "vehicle_id", l.VehicleID, "error", err)
		}
	}
	initialSubscription := make(chan error, 1)
	var initialOnce sync.Once
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("fleet-api-subscriber").
		SetUsername(username).
		SetPassword(password).
		SetAutoReconnect(true).
		SetConnectRetry(true)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) { slog.Error("mqtt connection lost", "error", err) })
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		token := client.Subscribe(topic, 1, handler)
		token.Wait()
		err := token.Error()
		initialOnce.Do(func() { initialSubscription <- err })
		if err != nil {
			slog.Error("mqtt subscription failed", "topic", topic, "error", err)
			return
		}
		slog.Info("mqtt subscription active", "topic", topic)
	})
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	select {
	case err := <-initialSubscription:
		if err != nil {
			client.Disconnect(250)
			return nil, fmt.Errorf("subscribe: %w", err)
		}
	case <-ctx.Done():
		client.Disconnect(250)
		return nil, ctx.Err()
	case <-time.After(10 * time.Second):
		client.Disconnect(250)
		return nil, fmt.Errorf("subscribe: timed out")
	}
	go func() {
		<-ctx.Done()
		client.Unsubscribe(topic)
		client.Disconnect(uint(time.Second / time.Millisecond))
	}()
	return client, nil
}
