# AB Experiments

A lightweight, deterministic experimentation service for safely rolling out product variants and measuring outcomes.

## Example

For a new onboarding flow, compare how effectively users locate a particular SKU:

- Control: existing onboarding flow
- Treatment: new onboarding flow
- Eligibility: decide whether the user can participate
- Assignment: deterministically assign eligible users to a variant
- Measurement: record exposure and outcome events

## Design

See [DESIGN.md](./DESIGN.md) for the architecture, assignment model, rollout strategy, APIs, and failure considerations.

## Core principle

Eligibility and assignment are separate decisions. A user is assigned only after eligibility is confirmed, and the assignment remains stable for the lifetime of the experiment.
