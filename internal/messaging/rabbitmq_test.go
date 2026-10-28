package messaging

import (
	"encoding/json"
	"strings"
	"testing"
	"transjakarta-fleet/internal/models"
)

func TestGeofenceEventValidationAndJSON(t *testing.T) {
	event := models.GeofenceEvent{
		EventID: "event-1", Event: "geofence_entry", VehicleID: "TJ-001",
		Location: models.EventLocation{Latitude: -6.21462, Longitude: 106.84513}, Timestamp: 1793113200,
	}
	if err := ValidateGeofenceEvent(event); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	jsonBody := string(body)
	for _, field := range []string{`"event":"geofence_entry"`, `"vehicle_id":"TJ-001"`, `"location":{"latitude":-6.21462,"longitude":106.84513}`, `"timestamp":1793113200`} {
		if !strings.Contains(jsonBody, field) {
			t.Fatalf("event JSON %s lacks %s", jsonBody, field)
		}
	}
	if strings.Contains(jsonBody, `"event_id"`) || strings.Contains(jsonBody, `"event_type"`) {
		t.Fatalf("event JSON has fields outside the assessment contract: %s", jsonBody)
	}
	if err := ValidateGeofenceEvent(models.GeofenceEvent{}); err == nil {
		t.Fatal("empty event should fail")
	}
}
