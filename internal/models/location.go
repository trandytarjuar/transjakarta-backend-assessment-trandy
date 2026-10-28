package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Location struct {
	ID        int64     `json:"id,omitempty"`
	VehicleID string    `json:"vehicle_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// MarshalJSON exposes timestamps as Unix seconds, as required by the public
// MQTT and HTTP contracts, while the application keeps time.Time internally.
func (l Location) MarshalJSON() ([]byte, error) {
	type locationJSON struct {
		ID        int64   `json:"id,omitempty"`
		VehicleID string  `json:"vehicle_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timestamp int64   `json:"timestamp"`
		CreatedAt int64   `json:"created_at,omitempty"`
	}
	encoded := locationJSON{ID: l.ID, VehicleID: l.VehicleID, Latitude: l.Latitude, Longitude: l.Longitude}
	if !l.Timestamp.IsZero() {
		encoded.Timestamp = l.Timestamp.Unix()
	}
	if !l.CreatedAt.IsZero() {
		encoded.CreatedAt = l.CreatedAt.Unix()
	}
	return json.Marshal(encoded)
}

// UnmarshalJSON accepts only the assessment's Unix-second timestamp format.
func (l *Location) UnmarshalJSON(data []byte) error {
	type locationJSON struct {
		ID        int64   `json:"id,omitempty"`
		VehicleID string  `json:"vehicle_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timestamp int64   `json:"timestamp"`
		CreatedAt int64   `json:"created_at,omitempty"`
	}
	var decoded locationJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing data")
	}
	*l = Location{ID: decoded.ID, VehicleID: decoded.VehicleID, Latitude: decoded.Latitude, Longitude: decoded.Longitude}
	if decoded.Timestamp != 0 {
		l.Timestamp = time.Unix(decoded.Timestamp, 0).UTC()
	}
	if decoded.CreatedAt != 0 {
		l.CreatedAt = time.Unix(decoded.CreatedAt, 0).UTC()
	}
	return nil
}

type EventLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type GeofenceEvent struct {
	EventID   string        `json:"-"`
	VehicleID string        `json:"vehicle_id"`
	Event     string        `json:"event"`
	Location  EventLocation `json:"location"`
	Timestamp int64         `json:"timestamp"`
}
