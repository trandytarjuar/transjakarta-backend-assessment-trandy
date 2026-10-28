package service

import (
	"math"
	"testing"
)

func TestDistanceAndBoundary(t *testing.T) {
	g := NewGeofenceService(0, 0, 50)
	if d := g.DistanceMeters(0, 0); d != 0 {
		t.Fatalf("center distance=%f", d)
	}
	nearLat := 40.0 / 111195.0
	if !g.IsInside(nearLat, 0) {
		t.Fatal("40m point should be inside")
	}
	farLat := 60.0 / 111195.0
	if g.IsInside(farLat, 0) {
		t.Fatal("60m point should be outside")
	}
	if d := g.DistanceMeters(farLat, 0); math.Abs(d-60) > 0.5 {
		t.Fatalf("distance=%f, want about 60", d)
	}
}
