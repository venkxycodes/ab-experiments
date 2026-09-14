# Agent instructions

This repository is a small A/B experimentation service and design exercise. Use the Codex productivity skills in `.agents/skills/` when they materially improve the work.

## Working principles

- Preserve experiment correctness, assignment stability, and measurement semantics.
- Prefer small, focused changes over drive-by refactors.
- Read the relevant design and surrounding files before changing them.
- Keep eligibility, assignment, exposure, and outcome semantics explicit.
- Verify behavior with tests or documented reasoning before considering work complete.
- Do not invent statistical guarantees without evidence.
- Use `$grilling` when requirements or design decisions are unclear.
- Use `$codebase-design` and `$design-an-interface` when shaping modules or APIs.
- Use `$architecture-review` for read-only simplification reviews.
- Use `$adversarial-review` before considering a design or implementation complete.
- Use `$code-simplification` and `$deslop` after implementation when applicable.

## Project context

- Product: deterministic A/B experimentation service.
- Current scenario: compare old and new onboarding flows for SKU discovery.
- Core invariant: eligibility is evaluated before variant assignment.
- Assignment must be stable for a subject within an experiment.
- Exposure is recorded only when the assigned experience is actually shown.
- Outcome metrics use exposed users as the denominator.

## Current verification

This repository currently contains design documentation only. When implementation is added, document and run the relevant test, lint, and build commands here.
