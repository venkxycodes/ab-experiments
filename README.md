# AB Experiments

A lightweight, deterministic experimentation service for safely rolling out product variants and measuring outcomes.

## Current implementation

A Go API built with Gin. The service is intentionally application-agnostic:

- The integrating application decides whether a subject is eligible.
- This service stores the experiment registry.
- This service deterministically assigns an eligible subject to a cohort.
- The assignment remains stable for the lifetime of the experiment.
- Exposure and business outcome events remain the responsibility of the integrating application.

Storage backends:

- PostgreSQL through GORM for durable experiments and assignments.
- In-memory storage as the default fallback for tests and quick local runs.

## Run with PostgreSQL

Start a local database:

```bash
docker compose up -d postgres
```

Run the API against it:

```bash
go mod tidy
STORAGE_BACKEND=postgres \
DATABASE_URL='host=localhost user=ab_experiments password=ab_experiments dbname=ab_experiments port=5432 sslmode=disable TimeZone=UTC' \
go run ./cmd/api
```

The service automatically creates the `experiments` and `experiment_assignments` tables on startup using GORM AutoMigrate.

Useful database settings:

- `DB_MAX_OPEN_CONNS` defaults to `10`
- `DB_MAX_IDLE_CONNS` defaults to `5`
- `PORT` defaults to `:8080`

If `STORAGE_BACKEND` and `DATABASE_URL` are not set, the service uses in-memory storage.

## API

Create an experiment:

```bash
curl -X POST http://localhost:8080/v1/experiments \
  -H 'Content-Type: application/json' \
  -d '{
    "key": "onboarding_sku_discovery",
    "status": "running",
    "variants": [
      {"key": "control", "weight": 50},
      {"key": "treatment", "weight": 50}
    ]
  }'
```

Resolve a cohort after the application has evaluated eligibility:

```bash
curl -X POST http://localhost:8080/v1/experiments/onboarding_sku_discovery/resolve \
  -H 'Content-Type: application/json' \
  -d '{"subject_id": "123", "eligible": true}'
```

Example response:

```json
{
  "experiment_key": "onboarding_sku_discovery",
  "eligible": true,
  "cohort": "control",
  "bucket": 3,
  "assignment_id": "assignment_..."
}
```

Numeric subject IDs use `subject_id % 10`; non-numeric IDs use a stable FNV-1a hash. Variant weights must sum to 100 and use 10% increments.

## Test

```bash
go test ./...
go vet ./...
```

## Design

See [DESIGN.md](./DESIGN.md) for the architecture and invariants.
