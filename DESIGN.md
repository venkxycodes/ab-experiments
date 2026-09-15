# A/B Experimentation Service Design

## 1. Problem

We need to compare two onboarding experiences and measure how many users successfully locate a target SKU:

- Control: the existing onboarding flow
- Treatment: the new onboarding flow

The service should support safe rollouts, stable assignments, accurate exposure measurement, and experiment analysis without making product services responsible for experimentation logic.

## 2. High-level flow

```text
Request
  |
  v
Eligibility evaluation
  |
  +-- ineligible --> normal product experience
  |
  v
Experiment selection
  |
  v
Deterministic variant assignment
  |
  v
Return variant/configuration
  |
  v
Product emits exposure and outcome events
  |
  v
Analytics pipeline computes experiment metrics
```

Eligibility and assignment are intentionally separate:

1. The integrating application evaluates its product-specific eligibility rules.
2. The application asks this service to resolve the active experiment.
3. The service assigns the eligible user to a variant.
4. Persist or derive the assignment consistently.
5. Emit an exposure event only when the assigned experience is actually shown.

An ineligible user must not consume experiment traffic.

## 3. Experiment model

An experiment contains:

- `experiment_id`
- `key`: stable product-facing key, such as `onboarding_sku_discovery`
- `status`: draft, running, paused, or completed
- `variants`: control and treatment
- `allocation`: traffic percentage per variant
- `start_time` and optional `end_time`
- `assignment_version`
- `created_by`

Example:

```json
{
  "key": "onboarding_sku_discovery",
  "variants": [
    {"key": "control", "weight": 50},
    {"key": "treatment", "weight": 50}
  ]
}
```

## 4. Deterministic assignment

Use a stable user identifier as the assignment key. For the simple two-variant case:

```text
bucket = hash(experiment_key + ":" + user_id) % 10
```

The bucket is mapped to the configured cumulative allocation:

```text
0..4 -> control
5..9 -> treatment
```

The V1 implementation uses `user_id % 10` for numeric IDs, matching the ten stable rollout cohorts discussed for this service. Non-numeric IDs use a stable hash. The important property is that the same user always maps to the same bucket for the same experiment.

Never use random assignment per request. That would make users switch experiences and corrupt the measurement.

## 5. Rollout strategy

A practical rollout sequence is:

1. Create the experiment with control at 100%.
2. Enable a small treatment allocation if operational risk requires it.
3. Move to 50/50 after validating correctness and key guardrails.
4. After the experiment has enough evidence, roll out the winner to 90/10 or 100/0.
5. Complete the experiment and preserve the assignment configuration for analysis.

For allocations in multiples of 10, the ten-bucket model is easy to reason about:

```text
bucket = hash(experiment_key + ":" + user_id) % 10
```

For example, 50/50 maps five buckets to each variant; 90/10 maps nine buckets to the winner and one bucket to the other variant.

However, changing allocation can move users between variants unless assignments are persisted or the system uses a rollout design that preserves existing assignments. For experiments where continuity matters, persist the first assignment:

```text
assignment(experiment_id, subject_id) -> variant
```

Then allocation changes affect only users who have not yet been assigned.

## 6. Assignment storage

Two valid modes:

### Derived assignment

Derive the variant from the stable hash every time.

Advantages:

- No assignment database required
- Very low latency
- Easy horizontal scaling

Risks:

- Changing allocation can move existing users
- Changing the hash algorithm or experiment key changes assignments
- Historical assignment must be reconstructed exactly

### Persisted assignment

Write the first assignment to an assignment store and return it on later requests.

Advantages:

- Users remain in the same variant
- Allocation changes can be controlled precisely
- Historical assignment is explicit

Risks:

- Requires storage and write-path reliability
- Needs idempotency and race protection

Recommended default: use persisted assignments for user-facing experiments where users may return across sessions. Derived assignment is acceptable for short-lived or stateless experiments when reassignment is explicitly acceptable.

## 7. API sketch

### Resolve assignment

```http
POST /v1/experiments/:key/resolve
```

The integrating application evaluates eligibility and sends only the result:

```json
{
  "subject_id": "123",
  "eligible": true
}
```

Response:

```json
{
  "experiment_key": "onboarding_sku_discovery",
  "eligible": true,
  "cohort": "treatment",
  "bucket": 73,
  "assignment_id": "assignment_..."
}
```

For an ineligible user, the service returns `eligible: false` without creating an assignment.

### Experiment administration

```text
POST   /v1/experiments
GET    /v1/experiments/:key
PATCH  /v1/experiments/:key
POST   /v1/experiments/:key/pause
POST   /v1/experiments/:key/complete
```

Administrative updates should be audited and validated so that weights sum to 100 and variants cannot be changed accidentally after exposure begins.

## 8. Events and metrics

The product should emit events rather than relying only on assignment responses.

### Exposure event

Emit after the variant is actually rendered:

```json
{
  "event_name": "experiment_exposure",
  "experiment_key": "onboarding_sku_discovery",
  "assignment_id": "a_123",
  "subject_id": "user-123",
  "variant": "treatment",
  "timestamp": "2026-09-14T00:00:00Z"
}
```

### Outcome event

```json
{
  "event_name": "sku_discovered",
  "experiment_key": "onboarding_sku_discovery",
  "assignment_id": "a_123",
  "subject_id": "user-123",
  "variant": "treatment",
  "sku_id": "sku-456",
  "timestamp": "2026-09-14T00:02:00Z"
}
```

The primary metric can be:

```text
SKU discovery rate =
unique users with sku_discovered
--------------------------------
unique users with experiment_exposure
```

Use the exposure population, not all eligible users, as the denominator.

Useful guardrails include onboarding completion, latency, crashes, support contacts, and downstream conversion.

## 9. Consistency and failure handling

- Cache active experiment definitions locally or in Redis, with a short TTL.
- Keep the assignment path synchronous and low latency.
- Make assignment writes idempotent using `(experiment_id, subject_id)` as a unique key.
- Use conditional insert/upsert to prevent concurrent requests assigning different variants.
- If the experimentation service is unavailable, fail open to the control experience for low-risk experiments.
- Never emit exposure for a failed or unavailable assignment.
- Include experiment configuration version in assignment and events.
- Do not change an experiment's variant keys after it starts.
- Protect administrative APIs with authentication, authorization, and audit logging.

## 10. Recommended initial architecture

For V1:

- Stateless Experiment Resolution API
- PostgreSQL for experiment definitions, persisted assignments, and audit history
- Redis for cached active experiment definitions
- Existing event pipeline for exposure and outcome events
- Analytics warehouse for aggregate metrics and reporting
- Simple admin API; UI can come later

The first implementation should optimize for correctness and reproducibility, not statistical sophistication. Sequential testing, confidence intervals, and automated winner selection can be added after the event and assignment data are trustworthy.


## 11. V1 implementation boundary

The Go API deliberately keeps application logic outside the experimentation service.

The integrating application owns:

- deciding whether a user satisfies product-specific eligibility criteria;
- deciding when the assigned experience was actually rendered;
- emitting exposure events;
- emitting business outcome events;
- computing product-specific metrics.

The experimentation service owns only:

- experiment definitions and variant weights;
- experiment status;
- stable subject-to-cohort assignment;
- assignment identity and bucket information.

The resolve request therefore contains an `eligible` boolean:

```json
{
  "subject_id": "123",
  "eligible": true
}
```

The service does not inspect country, app version, subscription state, SKU details, or any other application-specific context.

## 12. Gin API implementation

The current API exposes:

```text
GET  /healthz
POST /v1/experiments
GET  /v1/experiments
GET  /v1/experiments/:key
POST /v1/experiments/:key/resolve
```

The implementation follows the base Go layout:

```text
cmd/api/                         process entrypoint
internal/config/                 environment configuration
internal/router/                 Gin route registration
internal/handler/                HTTP decoding and response mapping
internal/service/                validation and assignment logic
internal/store/                  PostgreSQL and in-memory persistence adapters
model/                           domain types
```

The service uses a ten-slot deterministic bucket space. Numeric subject IDs use `subject_id % 10`, which preserves the same cohort for the same subject and makes 50/50 and 90/10 rollouts straightforward. Assignments are persisted in the in-memory store after the first eligible resolution, so later allocation changes do not move already-assigned subjects within the process.

## 13. Deliberate V1 limitations

- No experiment update or lifecycle administration endpoint yet.
- No exposure/outcome event ingestion.
- No statistical analysis or automatic winner selection.
- No application-specific eligibility rule engine.

PostgreSQL is now the durable `ExperimentStore` adapter. Before production, replace AutoMigrate with versioned schema migrations; keep application-specific eligibility logic outside this service.


## 14. PostgreSQL persistence

The durable adapter uses GORM with PostgreSQL:

- `experiments` stores the registry and variant allocation as JSONB.
- `experiment_assignments` stores one assignment per `(experiment_key, subject_id)`.
- Indexed keys keep registry and assignment lookups lightweight.
- Assignment creation uses `ON CONFLICT DO NOTHING`, then reads the existing row when another replica wins the race.
- Prepared statements and a configured connection pool are enabled.
- AutoMigrate is suitable for this base implementation; production should adopt versioned schema migrations.
