---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T13:54:43Z
last_modified: 2026-11-23T13:54:43Z
approved_fingerprint: sha256:61a300d0fbca174cde82c47ff984b1f1b7fa3354a717711eb94acf3c95719306
---

# Requirements Document

## Introduction

Phase 2 ("Business unit") introduces abstract **Obligations** and concrete **Bindings** (`docs/design/gradus-design.md` §19). Upper levels declare *what* governance process must be satisfied (e.g., `OBL-SDD-01: New behavior requires an approved specification before code` with `allowed_providers: [walden, spec-kit]`), while lower levels or `gradus.yaml` declare *how* each obligation is satisfied in a repository scope via a `binding` (`provider`, `requires`, `scope`). This feature defines the `gradus/v1` schema, validation, pyramid resolution invariants, and `ResolvedView`/`handbook` reporting for obligations and bindings (provider execution and constitution injection belong to `F17` adapters).

## Requirements

### R1 Obligation and Binding Schema and Validation

**User Story:** As a governance author, I want to declare abstract `obligations` in `level.yaml` (or `gradus.yaml`) and concrete `bindings` in lower levels or `gradus.yaml`, so that process requirements are separated from specific tooling providers.

#### Acceptance Criteria

1. `R1.AC1` WHEN a `level.yaml` or `gradus.yaml` declares `obligations` (each with `id`, `rule`, optional `allowed_providers`, and optional `scope`) and `bindings` (each with `obligation`, `provider`, optional `requires`, and optional `scope`), the system SHALL parse and preserve them in `schema.LevelConfig` and `schema.PyramidConfig` while maintaining `gradus/v1` JSON Schema validity.
   - Acceptance check: Parsing valid `level.yaml` and `gradus.yaml` files containing `obligations` and `bindings` succeeds and passes `spec/v1` JSON Schema validation.
2. `R1.AC2` IF an obligation omits `id` or `rule`, declares duplicate `id` values within the same config, or a binding omits `obligation` or `provider`, THEN configuration parsing SHALL fail with exit code `2` and a clear diagnostic.
   - Acceptance check: Invalid obligation or binding declarations fail `ValidatePyramidConfig` / `ValidateLevelConfig` and cause CLI commands to exit with code `2`.

### R2 Pyramid Resolution and Invariant Enforcement

**User Story:** As a platform engineer, I want `gradusctl resolve` and `gradusctl verify` to validate obligation-binding compatibility across the pyramid, so that lower levels cannot satisfy an obligation with a disallowed provider or leave declared obligations unbound.

#### Acceptance Criteria

1. `R2.AC1` WHEN `resolve.Resolve` evaluates a pyramid containing `obligations` and `bindings`, the system SHALL match each binding to its target obligation, verify that `binding.provider` is permitted by `obligation.allowed_providers` (when non-empty), and mark the obligation as satisfied in `ResolvedView`.
   - Acceptance check: A pyramid where `team` declares `OBL-SDD-01` (`allowed_providers: [walden, spec-kit]`) and `project` (or `gradus.yaml`) binds `OBL-SDD-01` to `walden` resolves cleanly with zero conflicts and `satisfied: true`.
2. `R2.AC2` IF a binding references a `provider` not in `allowed_providers`, references an unknown `obligation` ID, or a lower level attempts to widen `allowed_providers` for an inherited obligation, THEN `resolve.Resolve` SHALL emit a blocking `Conflict` (`kind: "obligation"`) causing `gradusctl verify` and `gradusctl compile` to exit with code `1`.
   - Acceptance check: Binding `OBL-SDD-01` to a disallowed provider `custom` produces an `obligation` conflict and causes `gradusctl verify` to fail with exit code `1`.
3. `R2.AC3` IF an active `obligation` has no matching valid `binding` in the resolved pyramid, THEN `resolve.Resolve` SHALL emit a blocking `Conflict` (`kind: "obligation"`) identifying the unbound obligation ID.
   - Acceptance check: Declaring `OBL-SDD-01` without any binding produces an `obligation` conflict in `resolve` and `verify`.

### R3 ResolvedView and Handbook Visibility

**User Story:** As a developer or downstream adapter, I want resolved obligations and bindings exposed in `gradusctl resolve --json` and `gradusctl handbook`, so that both tooling adapters and onboarding documentation reflect active process bindings.

#### Acceptance Criteria

1. `R3.AC1` WHEN `resolve.Resolve` completes on a pyramid with obligations or bindings, `ResolvedView` (`gradusctl resolve --json`) SHALL include `obligations` (with `id`, `rule`, `level`, `allowed_providers`, `bindings`, `satisfied`) and `bindings`, and update `view_hash` deterministically.
   - Acceptance check: `gradusctl resolve --json` outputs the `obligations` and `bindings` arrays deterministically across consecutive runs.
2. `R3.AC2` WHEN `gradusctl handbook` renders a pyramid containing obligations or bindings, the generated Markdown SHALL include an `Obligations & Bindings` section summarizing each obligation, its bound provider(s), `requires`, and `scope`.
   - Acceptance check: `gradusctl handbook` output includes the obligation ID, rule text, and bound provider name.

## Non-Functional Requirements

- `NFR1` **Deterministic Ordering:** `ResolvedView.Obligations` and `ResolvedView.Bindings` MUST be sorted deterministically so `view_hash` is invariant across runs and operating systems (`E1`).
- `NFR2` **Backward Compatibility:** Existing pyramids without `obligations` or `bindings` MUST resolve with identical `view_hash` (`omitempty` JSON fields) and zero new conflicts (`C1`).

## Constraints And Dependencies

- `C1` Must preserve `gradus/v1` backward compatibility across `spec/v1/schema/gradus.schema.json`, `level.schema.json`, and `resolved-view.schema.json`.
- `C2` Must build with `CGO_ENABLED=0` and reuse `internal/schema`, `internal/resolve`, and `internal/compile/handbook.go` without new third-party dependencies.

## Out Of Scope

- Provider adapters (`walden`, `spec-kit`), constitution managed-section injection, and provider evidence verification (`gradusctl adapter`) — covered by `F17` (`adapters-walden-and-speckit`).
- Toolchain `required` / `recommended` / `forbidden` declarations — covered by `F18` (`toolchain-governance`).
