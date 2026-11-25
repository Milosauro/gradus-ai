---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:56:09Z
last_modified: 2026-11-25T09:56:09Z
approved_fingerprint: sha256:2db72fdc92eb8a7c672c2fbd91bd0b84341226cb153d8b27348d835b8cbd08c9
---

# Requirements Document

## Introduction

Feature `F28` (`maturity-assessment-and-commit-trailers`) completes Phase 3 ("Organization", `v0.4.0`) by delivering Design capabilities `X-13` (process-only maturity assessment via `gradusctl assess` and the embedded `gradus-assess` skill, strictly excluding individual evaluation per P16) and `X-05` / `D-65` (optional git commit attribution trailers via `gradusctl trailers generate` and `gradusctl trailers verify`), plus Phase 3 release documentation updates.

## Requirements

### R1 Process Maturity Assessment Engine and `gradus-assess` Skill

**User Story:** As an engineering leader or platform team, I want a deterministic assessment of our repository's governance process maturity (and a guided `gradus-assess` skill) that evaluates configuration and process controls without ever evaluating individuals.

#### Acceptance Criteria

1. `R1.AC1` WHEN `assess.Evaluate(rootDir)` inspects a Gradus repository, it SHALL compute deterministic process maturity scores across five process dimensions (`pyramid_integrity`, `enforcement_depth`, `change_governance`, `obligations_and_toolchain`, `evidence_and_audit`), assign a maturity level (`L1`–`L5`), emit concrete process recommendations, and contain zero individual developer metrics.
   - Acceptance check: `TestProcessMaturityAssessment` verifies higher maturity configurations score above minimal setups and output contains zero individual metrics.
2. `R1.AC2` WHEN `skills.FS` is compiled for agent targets, it SHALL include `gradus-assess/SKILL.md` (passing `TestEmbeddedSkills` with explicit prohibitions against evaluating individuals) and a Skill-TDD scenario under `evals/scenarios/`.
   - Acceptance check: `go test -v -count=1 ./skills` passes with `gradus-assess` included in `canonicalSkills`.

### R2 Optional Commit Attribution Trailers (`X-05`, `D-65`)

**User Story:** As a compliance or release engineer, I want `gradusctl trailers generate` and `gradusctl trailers verify` to produce and validate optional process-attribution git trailers (`Gradus-Lock-Hash`, `Gradus-View-Hash`, `Gradus-Agent`, `Gradus-Model-Tier`, `Gradus-Review-ID`) using the system `git` binary.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl trailers generate` runs on a Gradus repository with optional `--agent`, `--model-tier`, and `--review-id` flags, it SHALL emit canonical `Key: Value` git trailer lines bound to the repository's current `lock_hash` and `view_hash` (rejecting invalid `--model-tier` with exit code `2`).
   - Acceptance check: `TestAssessAndTrailersCLI` verifies generated trailer lines and exit `2` on invalid `--model-tier`.
2. `R2.AC2` WHEN `gradusctl trailers verify --ref <ref>` inspects a git commit message via `os/exec` `git`, it SHALL parse `Gradus-*` trailers, validate `Gradus-Model-Tier` values, verify `Gradus-Lock-Hash` matches the current `gradus.lock` when `--require-lock-hash` is set, and exit `1` on mismatch or missing required trailers.
   - Acceptance check: `TestAssessAndTrailersCLI` verifies matching commit trailers pass (exit `0`) and missing/mismatched trailers fail (exit `1`).

### R3 Phase 3 Closure Documentation and Dogfooding Compilation

**User Story:** As a user or maintainer preparing `v0.4.0`, I want `README.md`, `CHANGELOG.md`, `docs/plan/phase-3-plan.md`, and `gradus-ai`'s compiled skills updated cleanly for Phase 3 completion.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradus-ai` is compiled and verified after adding `gradus-assess` and Phase 3 docs (`README.md`, `CHANGELOG.md`, `docs/plan/phase-3-plan.md`), `gradusctl verify --json`, `gradusctl check --json`, and `gradusctl test --json` SHALL all pass with exit code `0`.
   - Acceptance check: `TestAssessAndTrailersCLI` and repository `gradusctl verify`/`check`/`test` exit `0`.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary invoking `git` via `os/exec` (`PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` Commit trailers SHALL remain strictly opt-in and non-breaking for repositories that do not use them.
2. `C2` All existing unit and integration tests SHALL pass without regressions.
