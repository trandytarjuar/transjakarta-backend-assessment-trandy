package repository

import (
	"context"
	"errors"
	"time"
	"transjakarta-fleet/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("location not found")

type LocationRepository interface {
	Save(context.Context, models.Location) (SaveResult, error)
	Latest(context.Context, string) (models.Location, error)
	History(context.Context, string, time.Time, time.Time) ([]models.Location, error)
}

type SaveResult struct {
	Previous *models.Location
	Inserted bool
	Latest   bool
}

type PostgresLocationRepository struct{ pool *pgxpool.Pool }

func NewPostgresLocationRepository(pool *pgxpool.Pool) *PostgresLocationRepository {
	return &PostgresLocationRepository{pool: pool}
}

func (r *PostgresLocationRepository) Save(ctx context.Context, l models.Location) (SaveResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, l.VehicleID); err != nil {
		return SaveResult{}, err
	}
	var previous models.Location
	err = tx.QueryRow(ctx, `SELECT id, vehicle_id, latitude, longitude, timestamp, created_at FROM vehicle_loctions WHERE vehicle_id=$1 ORDER BY timestamp DESC, id DESC LIMIT 1`, l.VehicleID).Scan(&previous.ID, &previous.VehicleID, &previous.Latitude, &previous.Longitude, &previous.Timestamp, &previous.CreatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return SaveResult{}, err
	}
	command, err := tx.Exec(ctx, `INSERT INTO vehicle_loctions(vehicle_id,latitude,longitude,timestamp) VALUES($1,$2,$3,$4) ON CONFLICT (vehicle_id,timestamp) DO NOTHING`, l.VehicleID, l.Latitude, l.Longitude, l.Timestamp)
	if err != nil {
		return SaveResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SaveResult{}, err
	}
	result := SaveResult{Inserted: command.RowsAffected() == 1}
	if !result.Inserted {
		return result, nil
	}
	result.Latest = previous.ID == 0 || l.Timestamp.After(previous.Timestamp)
	if previous.ID != 0 {
		result.Previous = &previous
	}
	return result, nil
}

func scanLocation(row pgx.Row) (models.Location, error) {
	var l models.Location
	err := row.Scan(&l.ID, &l.VehicleID, &l.Latitude, &l.Longitude, &l.Timestamp, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return l, err
}
func (r *PostgresLocationRepository) Latest(ctx context.Context, id string) (models.Location, error) {
	return scanLocation(r.pool.QueryRow(ctx, `SELECT id,vehicle_id,latitude,longitude,timestamp,created_at FROM vehicle_loctions WHERE vehicle_id=$1 ORDER BY timestamp DESC,id DESC LIMIT 1`, id))
}
func (r *PostgresLocationRepository) History(ctx context.Context, id string, start, end time.Time) ([]models.Location, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,vehicle_id,latitude,longitude,timestamp,created_at FROM vehicle_loctions WHERE vehicle_id=$1 AND timestamp >= $2 AND timestamp <= $3 ORDER BY timestamp ASC,id ASC`, id, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.Location, 0)
	for rows.Next() {
		var l models.Location
		if err := rows.Scan(&l.ID, &l.VehicleID, &l.Latitude, &l.Longitude, &l.Timestamp, &l.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	return items, rows.Err()
}
