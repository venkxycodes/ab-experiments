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

1. Check whether the user satisfies the experiment's eligibility rules.
2. Select the active experiment for the requested feature.
3. Assign the eligible user to a variant.
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
- `eligibility_rules`
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
  ],
  "eligibility_rules": {
    "country": ["IN"],
    "app_version_min": "4.2.0"
  }
}
```

## 4. Deterministic assignment

Use a stable user identifier as the assignment key. For the simple two-variant case:

```text
bucket = hash(experiment_key + ":" + user_id) % 100
```

The bucket is mapped to the configured cumulative allocation:

```text
0..49  -> control
50..99 -> treatment
```

A plain `user_id % 10` works for a simple internal prototype, but hashing is safer because user IDs may have patterns or be sequential. The important property is that the same user always maps to the same bucket for the same experiment.

Never use random assignment per request. That would make users switch experiences and corrupt the measurement.

## 5. Rollout strategy

A practical rollout sequence is:

1. Create the experiment with control at 100%.
2. Enable a small treatment allocation if operational risk requires it.
3. Move to 50/50 after validating correctness and key guardrails.
4. After the experiment has enough evidence, roll out the winner to 90/10 or 100/0.
5. Complete the experiment and preserve the assignment configuration for analysis.

For allocations in multiples of 10, a ten-bucket model is easy to reason about:

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
POST /v1/experiments/resolve
```

Request:

```json
{
  "subject_id": "user-123",
  "experiment_key": "onboarding_sku_discovery",
  "context": {
    "country": "IN",
    "app_version": "4.2.1"
  }
}
```

Response:

```json
{
  "experiment_key": "onboarding_sku_discovery",
  "eligible": true,
  "variant": "treatment",
  "assignment_id": "a_123",
  "config": {
    "onboarding_flow": "new"
  }
}
```

For an ineligible user:

```json
{
  "experiment_key": "onboarding_sku_discovery",
  "eligible": false,
  "variant": null
}
```

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
