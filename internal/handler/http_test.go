package handler

import (
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"testing"
	"time"
	"transjakarta-fleet/internal/models"
	"transjakarta-fleet/internal/repository"
	"transjakarta-fleet/internal/service"
)

type repoStub struct{}

func (repoStub) Save(context.Context, models.Location) (repository.SaveResult, error) {
	return repository.SaveResult{}, nil
}
func (repoStub) Latest(_ context.Context, id string) (models.Location, error) {
	return models.Location{ID: 1, VehicleID: id, Latitude: -6.2, Longitude: 106.8, Timestamp: time.Now()}, nil
}
func (repoStub) History(_ context.Context, id string, start, end time.Time) ([]models.Location, error) {
	return []models.Location{{ID: 1, VehicleID: id, Timestamp: start}}, nil
}

type noEvents struct{}

func (noEvents) PublishGeofenceEntry(context.Context, models.GeofenceEvent) error { return nil }
func testApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	New(service.NewLocationService(repoStub{}, service.NewGeofenceService(0, 0, 50), noEvents{})).Register(app)
	return app
}
func TestHealth(t *testing.T) {
	res, err := testApp().Test(httptest.NewRequest("GET", "/health", nil))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("status=%d err=%v", res.StatusCode, err)
	}
}
func TestHistoryValidation(t *testing.T) {
	res, err := testApp().Test(httptest.NewRequest("GET", "/vehicles/TJ-1/history?start=nope&end=nope", nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var body map[string]any
	if err = json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == nil {
		t.Fatal("expected JSON error")
	}
}

func TestUnixTimestampContract(t *testing.T) {
	res, err := testApp().Test(httptest.NewRequest("GET", "/vehicles/TJ-1/location", nil))
	if err != nil {
		t.Fatal(err)
	}
	var latest struct {
		Data struct {
			Timestamp int64 `json:"timestamp"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&latest); err != nil || latest.Data.Timestamp <= 0 {
		t.Fatalf("latest=%+v err=%v", latest, err)
	}

	res, err = testApp().Test(httptest.NewRequest("GET", "/vehicles/TJ-1/history?start=1715000000&end=1715009999", nil))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("status=%d err=%v", res.StatusCode, err)
	}
}
