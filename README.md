# Transjakarta Fleet Management

Production-minded reference backend for receiving vehicle GPS telemetry over MQTT, storing it in PostgreSQL, detecting entry into a 50 m geofence, and dispatching alerts through RabbitMQ. The REST API uses **Fiber v2 exclusively**.

## Architecture

`mock publisher → Mosquitto → API subscriber → PostgreSQL → geofence service → RabbitMQ → worker`

The API process subscribes to `/fleet/vehicle/+/location` using username/password authentication. Mosquitto rejects anonymous clients. The API strictly validates the topic and JSON payload, checks that the topic vehicle matches `vehicle_id`, and persists valid samples in `vehicle_loctions`. RabbitMQ uses durable direct exchange `fleet.events`, routing key `geofence.entry`, and durable queue `geofence_alerts`.

## Run

Requirements: Docker with Compose.

```bash
cp .env.example .env
docker compose up --build -d
docker compose ps
curl http://localhost:18080/health
```

Edit `.env` before startup. Docker Compose reads it automatically. The checked-in `.env.example` documents every required variable; `.env` is ignored to prevent committing local credentials. On this development machine, `.env` uses API port `18080` because `8080` is already occupied.

The publisher emits a `TJ-001` point every two seconds. It travels toward and through the configured center, then resets outside. RabbitMQ management is at `http://localhost:15672` (`fleet` / `fleet`). Stop with `docker compose down`; add `-v` only when you intentionally want to delete database data.

## API

- `GET /health`
- `GET /vehicles/TJ-001/location`
- `GET /vehicles/TJ-001/history?start=1767225600&end=1798761599`

Successful responses use `{ "data": ... }` (history also includes `count`). Errors use `{ "error": { "code": "...", "message": "..." } }`. All public timestamps and the history `start`/`end` parameters are Unix seconds. PostgreSQL stores them as `TIMESTAMPTZ`. A missing latest location returns 404; malformed ranges return 400; server failures return 500.

MQTT payload example:

```json
{"vehicle_id":"TJ-001","latitude":-6.21462,"longitude":106.84513,"timestamp":1793113200}
```

Geofence event example:

```json
{"vehicle_id":"TJ-001","event":"geofence_entry","location":{"latitude":-6.21462,"longitude":106.84513},"timestamp":1793113200}
```

## Configuration

Environment variables are documented in `.env.example`. Application settings include `DATABASE_URL`, `MQTT_BROKER`, `MQTT_TOPIC`, `MQTT_USERNAME`, `MQTT_PASSWORD`, `RABBITMQ_URL`, `GEOFENCE_LAT`, `GEOFENCE_LON`, `GEOFENCE_RADIUS_METERS`, and publisher-only `VEHICLE_ID`. Compose also uses the PostgreSQL/RabbitMQ credentials and configurable host ports. Compose service URLs must use internal hostnames such as `postgres`, `mosquitto`, and `rabbitmq`.

MQTT credentials are required. For local testing, subscribe with:

```powershell
docker compose exec mosquitto mosquitto_sub -h localhost -u fleet -P fleet -t "/fleet/vehicle/+/location" -v
```

## Development and verification

```bash
go mod tidy
go test ./...
go vet ./...
docker compose config
```

Migrations in `migrations/` are applied by PostgreSQL on first volume initialization. `002_align_assessment_schema.sql` upgrades databases made by the earlier implementation; for an existing volume, apply it once with `docker compose exec -T postgres psql -U fleet -d fleet -f /docker-entrypoint-initdb.d/002_align_assessment_schema.sql`.

## Design notes

- Coordinates and required timestamps are validated before storage.
- A PostgreSQL advisory transaction lock serializes writes for each vehicle. The unique `(vehicle_id, timestamp)` index makes MQTT redelivery idempotent.
- The latest persisted timestamp is the durable geofence state. Older out-of-order samples remain available in history but cannot change state or emit an event. Leaving and entering again emits a new event, while remaining inside does not.
- MQTT QoS 1, automatic MQTT reconnection, persistent RabbitMQ messages, durable topology, and manual worker acknowledgements favor reliable delivery.
- Credentials in Compose are local-development defaults and must be replaced in a real deployment.
