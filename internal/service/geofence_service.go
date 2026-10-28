package service

import "math"

type GeofenceService struct{ Latitude, Longitude, RadiusMeters float64 }

func NewGeofenceService(lat, lon, radius float64) *GeofenceService {
	return &GeofenceService{lat, lon, radius}
}
func (g *GeofenceService) DistanceMeters(lat, lon float64) float64 {
	const earth = 6371000.0
	rad := math.Pi / 180
	dLat := (lat - g.Latitude) * rad
	dLon := (lon - g.Longitude) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(g.Latitude*rad)*math.Cos(lat*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earth * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
func (g *GeofenceService) IsInside(lat, lon float64) bool {
	return g.DistanceMeters(lat, lon) <= g.RadiusMeters
}
