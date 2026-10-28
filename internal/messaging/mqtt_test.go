package messaging

import "testing"

func TestVehicleIDFromTopic(t *testing.T) {
	id, err := VehicleIDFromTopic("/fleet/vehicle/TJ-001/location")
	if err != nil || id != "TJ-001" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	invalid := []string{"fleet/vehicle/TJ-001/location", "/fleet/vehicles/TJ-001/location", "/fleet/vehicle//location", "/vehicles/TJ-001/location"}
	for _, topic := range invalid {
		if _, err := VehicleIDFromTopic(topic); err == nil {
			t.Fatalf("topic %q should fail", topic)
		}
	}
}

func TestDecodeLocationPayload(t *testing.T) {
	payload := []byte(`{"vehicle_id":"TJ-001","latitude":-6.2,"longitude":106.8,"timestamp":1793113200}`)
	location, err := DecodeLocationPayload(payload)
	if err != nil || location.VehicleID != "TJ-001" {
		t.Fatalf("location=%+v err=%v", location, err)
	}
	invalid := [][]byte{
		[]byte(`{"vehicle_id":"TJ-001","latitude":91,"longitude":106.8,"timestamp":1767225600}`),
		[]byte(`{"vehicle_id":"TJ-001","latitude":-6.2,"longitude":106.8}`),
		[]byte(`{"vehicle_id":"TJ-001","latitude":-6.2,"longitude":106.8,"timestamp":1767225600,"unknown":true}`),
		[]byte(`{"vehicle_id":"TJ-001","latitude":-6.2,"longitude":106.8,"timestamp":"2026-01-01T00:00:00Z"}`),
		[]byte(`not-json`),
	}
	for _, body := range invalid {
		if _, err := DecodeLocationPayload(body); err == nil {
			t.Fatalf("payload %s should fail", body)
		}
	}
}
