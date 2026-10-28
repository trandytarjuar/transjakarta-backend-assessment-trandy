package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math"
	"strings"
	"time"
	"transjakarta-fleet/internal/models"
	"transjakarta-fleet/internal/repository"
)

var ErrInvalidLocation = errors.New("invalid location")

type EventPublisher interface {
	PublishGeofenceEntry(context.Context, models.GeofenceEvent) error
}
type LocationService struct {
	repo     repository.LocationRepository
	geofence *GeofenceService
	events   EventPublisher
}

func NewLocationService(r repository.LocationRepository, g *GeofenceService, e EventPublisher) *LocationService {
	return &LocationService{r, g, e}
}
func Validate(l models.Location) error {
	if strings.TrimSpace(l.VehicleID) == "" {
		return fmt.Errorf("%w: vehicle_id is required", ErrInvalidLocation)
	}
	if math.IsNaN(l.Latitude) || math.IsInf(l.Latitude, 0) || l.Latitude < -90 || l.Latitude > 90 {
		return fmt.Errorf("%w: latitude out of range", ErrInvalidLocation)
	}
	if math.IsNaN(l.Longitude) || math.IsInf(l.Longitude, 0) || l.Longitude < -180 || l.Longitude > 180 {
		return fmt.Errorf("%w: longitude out of range", ErrInvalidLocation)
	}
	if l.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp is required", ErrInvalidLocation)
	}
	return nil
}
func (s *LocationService) Process(ctx context.Context, l models.Location) error {
	if err := Validate(l); err != nil {
		return err
	}
	result, err := s.repo.Save(ctx, l)
	if err != nil {
		return err
	}
	if !result.Inserted || !result.Latest || result.Previous == nil {
		return nil
	}
	inside := s.geofence.IsInside(l.Latitude, l.Longitude)
	wasInside := s.geofence.IsInside(result.Previous.Latitude, result.Previous.Longitude)
	if inside && !wasInside {
		return s.events.PublishGeofenceEntry(ctx, models.GeofenceEvent{
			EventID: uuid.NewString(), VehicleID: l.VehicleID, Event: "geofence_entry",
			Location: models.EventLocation{Latitude: l.Latitude, Longitude: l.Longitude}, Timestamp: l.Timestamp.Unix(),
		})
	}
	return nil
}
func (s *LocationService) Latest(ctx context.Context, id string) (models.Location, error) {
	return s.repo.Latest(ctx, id)
}
func (s *LocationService) History(ctx context.Context, id string, start, end time.Time) ([]models.Location, error) {
	if strings.TrimSpace(id) == "" || start.IsZero() || end.IsZero() || start.After(end) {
		return nil, fmt.Errorf("%w: invalid history range", ErrInvalidLocation)
	}
	return s.repo.History(ctx, id, start, end)
}
