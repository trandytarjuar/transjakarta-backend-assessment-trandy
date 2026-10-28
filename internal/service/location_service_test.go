package service

import (
	"context"
	"testing"
	"time"
	"transjakarta-fleet/internal/models"
	"transjakarta-fleet/internal/repository"
)

type fakeRepo struct {
	results []repository.SaveResult
	saved   []models.Location
}

func (f *fakeRepo) Save(_ context.Context, l models.Location) (repository.SaveResult, error) {
	f.saved = append(f.saved, l)
	result := f.results[0]
	f.results = f.results[1:]
	return result, nil
}
func (f *fakeRepo) Latest(context.Context, string) (models.Location, error) {
	return models.Location{}, nil
}
func (f *fakeRepo) History(context.Context, string, time.Time, time.Time) ([]models.Location, error) {
	return nil, nil
}

type fakeEvents struct{ events []models.GeofenceEvent }

func (f *fakeEvents) PublishGeofenceEntry(_ context.Context, e models.GeofenceEvent) error {
	f.events = append(f.events, e)
	return nil
}
func TestProcessPublishesOnlyOutsideToInside(t *testing.T) {
	now := time.Now()
	outside := models.Location{VehicleID: "TJ-1", Latitude: 0.001, Longitude: 0, Timestamp: now.Add(-time.Second)}
	inside := models.Location{VehicleID: "TJ-1", Latitude: 0, Longitude: 0, Timestamp: now}
	repo := &fakeRepo{results: []repository.SaveResult{{Previous: &outside, Inserted: true, Latest: true}, {Previous: &inside, Inserted: true, Latest: true}}}
	events := &fakeEvents{}
	s := NewLocationService(repo, NewGeofenceService(0, 0, 50), events)
	if err := s.Process(context.Background(), inside); err != nil {
		t.Fatal(err)
	}
	if len(events.events) != 1 {
		t.Fatalf("events=%d, want 1", len(events.events))
	}
	if err := s.Process(context.Background(), models.Location{VehicleID: "TJ-1", Latitude: 0, Longitude: 0, Timestamp: now.Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	if len(events.events) != 1 {
		t.Fatal("inside-to-inside must not publish")
	}
}
func TestValidate(t *testing.T) {
	tests := []models.Location{{}, {VehicleID: "TJ", Latitude: 91, Timestamp: time.Now()}, {VehicleID: "TJ", Longitude: 181, Timestamp: time.Now()}}
	for _, v := range tests {
		if Validate(v) == nil {
			t.Fatalf("expected validation error for %+v", v)
		}
	}
}

func TestProcessSupportsExitAndReentry(t *testing.T) {
	now := time.Now()
	inside := models.Location{VehicleID: "TJ-1", Latitude: 0, Longitude: 0, Timestamp: now}
	outside := models.Location{VehicleID: "TJ-1", Latitude: 0.001, Longitude: 0, Timestamp: now.Add(time.Second)}
	repo := &fakeRepo{results: []repository.SaveResult{
		{Previous: &inside, Inserted: true, Latest: true},
		{Previous: &outside, Inserted: true, Latest: true},
	}}
	events := &fakeEvents{}
	s := NewLocationService(repo, NewGeofenceService(0, 0, 50), events)
	if err := s.Process(context.Background(), outside); err != nil {
		t.Fatal(err)
	}
	if err := s.Process(context.Background(), models.Location{VehicleID: "TJ-1", Latitude: 0, Longitude: 0, Timestamp: now.Add(2 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	if len(events.events) != 1 {
		t.Fatalf("re-entry events=%d, want 1", len(events.events))
	}
}

func TestProcessIgnoresDuplicateOutOfOrderAndUnknownInitialState(t *testing.T) {
	now := time.Now()
	repo := &fakeRepo{results: []repository.SaveResult{
		{Inserted: false},
		{Previous: &models.Location{VehicleID: "TJ-1", Latitude: 0.001, Timestamp: now}, Inserted: true, Latest: false},
		{Inserted: true, Latest: true},
	}}
	events := &fakeEvents{}
	s := NewLocationService(repo, NewGeofenceService(0, 0, 50), events)
	location := models.Location{VehicleID: "TJ-1", Latitude: 0, Longitude: 0, Timestamp: now}
	for i := 0; i < 3; i++ {
		if err := s.Process(context.Background(), location); err != nil {
			t.Fatal(err)
		}
	}
	if len(events.events) != 0 {
		t.Fatalf("events=%d, want 0", len(events.events))
	}
}
