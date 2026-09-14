# AB Experiments

A lightweight, deterministic experimentation service for safely rolling out product variants and measuring outcomes.

## Current implementation

A Go API built with Gin. The service is intentionally application-agnostic:

- The integrating application decides whether a subject is eligible.
- This service stores the experiment registry.
- This service deterministically assigns an eligible subject to a cohort.
- The assignment remains stable for the lifetime of the process.
- Exposure and business outcome events remain the responsibility of the integrating application.

The current adapter is in-memory, so experiments and assignments are lost on restart.

## Run

```bash
go mod tidy
go run ./cmd/api
```

The server listens on `:8080` by default. Set `PORT` to override it.

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
  "bucket": 23,
  "assignment_id": "assignment_..."
}
```

Numeric subject IDs use `subject_id % 100`; non-numeric IDs use a stable FNV-1a hash. Variant weights must sum to 100.

## Design

See [DESIGN.md](./DESIGN.md) for the architecture and invariants.
