package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"log/slog"
	"math"
	"strings"
	"time"
	"transjakarta-fleet/internal/models"
)

const Exchange = "fleet.events"
const Queue = "geofence_alerts"
const RoutingKey = "geofence.entry"

type RabbitPublisher struct{ ch *amqp091.Channel }

func Setup(url string) (*amqp091.Connection, *amqp091.Channel, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if err = ch.ExchangeDeclare(Exchange, "direct", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}
	q, err := ch.QueueDeclare(Queue, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}
	if err = ch.QueueBind(q.Name, RoutingKey, Exchange, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}
func NewRabbitPublisher(ch *amqp091.Channel) *RabbitPublisher { return &RabbitPublisher{ch} }

func ValidateGeofenceEvent(e models.GeofenceEvent) error {
	if strings.TrimSpace(e.EventID) == "" || e.Event != "geofence_entry" || strings.TrimSpace(e.VehicleID) == "" {
		return fmt.Errorf("missing or invalid event identity")
	}
	if math.IsNaN(e.Location.Latitude) || math.IsInf(e.Location.Latitude, 0) || e.Location.Latitude < -90 || e.Location.Latitude > 90 ||
		math.IsNaN(e.Location.Longitude) || math.IsInf(e.Location.Longitude, 0) || e.Location.Longitude < -180 || e.Location.Longitude > 180 || e.Timestamp <= 0 {
		return fmt.Errorf("invalid event location or timestamp")
	}
	return nil
}

func (p *RabbitPublisher) PublishGeofenceEntry(ctx context.Context, e models.GeofenceEvent) error {
	if err := ValidateGeofenceEvent(e); err != nil {
		return err
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, Exchange, RoutingKey, true, false, amqp091.Publishing{ContentType: "application/json", DeliveryMode: amqp091.Persistent, MessageId: e.EventID, Timestamp: time.Unix(e.Timestamp, 0).UTC(), Body: body})
}
func Consume(ctx context.Context, ch *amqp091.Channel) error {
	if err := ch.Qos(10, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(Queue, "geofence-worker", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}
			var e models.GeofenceEvent
			if err := json.Unmarshal(d.Body, &e); err != nil {
				slog.Error("invalid geofence event", "error", err)
				_ = d.Nack(false, false)
				continue
			}
			e.EventID = d.MessageId
			if err := ValidateGeofenceEvent(e); err != nil {
				slog.Error("invalid geofence event", "error", err)
				_ = d.Nack(false, false)
				continue
			}
			slog.Info("geofence entry alert processed", "message_id", d.MessageId, "vehicle_id", e.VehicleID, "latitude", e.Location.Latitude, "longitude", e.Location.Longitude)
			if err := d.Ack(false); err != nil {
				return fmt.Errorf("ack geofence event: %w", err)
			}
		}
	}
}
