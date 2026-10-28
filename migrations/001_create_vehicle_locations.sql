CREATE TABLE IF NOT EXISTS vehicle_loctions (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id VARCHAR(64) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_vehicle_loctions_vehicle_timestamp
    ON vehicle_loctions (vehicle_id, timestamp);

CREATE INDEX IF NOT EXISTS idx_vehicle_loctions_vehicle_timestamp
    ON vehicle_loctions (vehicle_id, timestamp DESC, id DESC);
