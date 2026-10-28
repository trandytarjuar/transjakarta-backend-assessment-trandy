-- Upgrade databases created by the earlier implementation without losing data.
DO $$
BEGIN
    IF to_regclass('public.vehicle_locations') IS NOT NULL
       AND to_regclass('public.vehicle_loctions') IS NULL THEN
        ALTER TABLE vehicle_locations RENAME TO vehicle_loctions;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'vehicle_loctions'
          AND column_name = 'recorded_at'
    ) THEN
        ALTER TABLE vehicle_loctions RENAME COLUMN recorded_at TO timestamp;
    END IF;
END $$;

DELETE FROM vehicle_loctions a
USING vehicle_loctions b
WHERE a.vehicle_id = b.vehicle_id
  AND a.timestamp = b.timestamp
  AND a.id > b.id;

CREATE UNIQUE INDEX IF NOT EXISTS uq_vehicle_loctions_vehicle_timestamp
    ON vehicle_loctions (vehicle_id, timestamp);

CREATE INDEX IF NOT EXISTS idx_vehicle_loctions_vehicle_timestamp
    ON vehicle_loctions (vehicle_id, timestamp DESC, id DESC);
