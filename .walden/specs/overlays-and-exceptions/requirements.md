---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T16:37:47Z
last_modified: 2026-11-21T16:37:47Z
approved_fingerprint: sha256:b38dc9c305e3413c41af2617d6819b1074f1558d2b69b69cee8f02bdaab4da0e
---

# Requirements Document

## Introduction

`overlays-and-exceptions` (`F10`) implements explicit overlay activation (`activation: explicit | always | disabled`) with lockfile and `allowed_overlays` verification, policy-checked exception and break-glass resolution, and the `gradusctl exception` CLI workflow (`request`, `approve`, `break-glass`, `list`, `expire`).

## Requirements

### R1 Explicit Overlay Activation and Constraints

**User Story:** As a team lead, I want overlays to support explicit activation in `gradus.yaml` while respecting upper-level `allowed_overlays` constraints and lockfile verification, so that cross-cutting stack or domain rules apply deterministically only where declared.

#### Acceptance Criteria

1. `R1.AC1` WHEN an overlay in `gradus.yaml` sets `activation: explicit` (or `always` or omitted), `gradusctl resolve` SHALL mark it `active: true`, merge its rules at its declared `strength`, and enforce `constraints.allowed_overlays` from upper levels (reporting a conflict and exit code `1` if disallowed).
   - Acceptance check: An overlay with `activation: explicit` listed in `allowed_overlays` is active and its rules appear immediately after its `strength` level; an unlisted overlay fails `resolve` and `verify` with exit code `1`.
2. `R1.AC2` WHEN an overlay in `gradus.yaml` sets `activation: disabled` (or `activation: profile` without matching profile rules), `gradusctl resolve` SHALL record it in `ResolvedView.Overlays` with `active: false` and exclude its rules from `ResolvedView.Rules`.
   - Acceptance check: A disabled overlay appears in `resolve --json` with `"active": false` and none of its rules are compiled or enforced.

### R2 Exception Policy and Break-Glass Enforcement

**User Story:** As a security or platform owner, I want exceptions and hotfix-style break-glass waivers validated deterministically against `level.yaml` exception policies and expiry/post-review dates, so that unapproved, overly long, expired, or disallowed waivers fail closed.

#### Acceptance Criteria

1. `R2.AC1` WHEN an exception in `.gradus/exceptions/*.yaml` has an empty `approved_by` list and `kind: exception`, `gradusctl resolve` SHALL treat it as pending without waiving the target rule.
   - Acceptance check: A requested exception with `approved_by: []` does not set `waived: true` on the target rule until approved.
2. `R2.AC2` WHEN an exception targets a rule whose level defines `exceptions.max_duration` and the exception's span (`expires - created`) exceeds `max_duration`, `gradusctl resolve` and `gradusctl verify` SHALL report an exception policy conflict and exit with code `1`.
   - Acceptance check: A 60-day exception against a level with `max_duration: 30d` produces a `Conflict{Kind: "exception"}` and exit code `1`.
3. `R2.AC3` WHEN an exception has `kind: break-glass`, `gradusctl resolve` and `gradusctl verify` SHALL report a blocking conflict (exit `1`) if `reason` is empty, `post_review_due` is missing/invalid/overdue, or the target level sets `exceptions.break_glass: false` or `offline.mode: fail-closed`.
   - Acceptance check: A `break-glass` exception against a level with `break_glass: false` or an overdue `post_review_due` fails `resolve` and `verify` with exit code `1`.

### R3 `gradusctl exception` CLI Workflow

**User Story:** As a developer or approver, I want dedicated `gradusctl exception` subcommands to request, approve, create break-glass waivers, list, and expire exceptions, so that all `.gradus/exceptions/*.yaml` state changes are owned by the CLI.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl exception request --rule <id> --reason <text> --expires <YYYY-MM-DD>` is invoked, the CLI SHALL write a valid pending exception YAML file under `.gradus/exceptions/` with `kind: exception` and `approved_by: []`.
   - Acceptance check: Running `exception request` creates `.gradus/exceptions/<id>.yaml` matching `exception.schema.json` with empty `approved_by`.
2. `R3.AC2` WHEN `gradusctl exception approve <id> --by <approver>` is invoked, the CLI SHALL append the approver to `approved_by` in `.gradus/exceptions/<id>.yaml`.
   - Acceptance check: After `exception approve`, the target rule resolves with `waived: true` and includes `<id>` in `waiver_ids`.
3. `R3.AC3` WHEN `gradusctl exception break-glass --rule <id> --reason <text> --expires <YYYY-MM-DD> --post-review-due <YYYY-MM-DD> --by <actor>` is invoked, the CLI SHALL validate the target level's `exceptions` policy and write a `kind: break-glass` record (or refuse with exit `1` when disallowed).
   - Acceptance check: `exception break-glass` succeeds when allowed by the level policy and exits `1` without writing when `break_glass: false`.
4. `R3.AC4` WHEN `gradusctl exception list [--json]` or `gradusctl exception expire <id>` is invoked, the CLI SHALL report all exceptions with their computed state (`pending`, `active`, `expired`) or set `expires` to a past date to immediately revoke an exception.
   - Acceptance check: `exception expire <id>` causes subsequent `exception list --json` to report status `expired` and stops waiving the rule.

## Non-Functional Requirements

- `NFR1` **Backward Compatibility:** Existing Phase 0 exception YAML records with `approved_by` populated and `kind: exception` MUST continue to work unchanged.
- `NFR2` **Canonical English & Static Build:** All code, schemas, and CLI messages MUST be written in canonical English (`PRJ-LANG-001`) and build with `CGO_ENABLED=0` (`PRJ-GO-001`).

## Constraints And Dependencies

- `C1` Exception records MUST conform to `spec/v1/schema/exception.schema.json` and `overlay` activation values MUST stay aligned with `internal/schema/validate_config.go` and `spec/v1/schema/overlay.schema.json`.

## Out Of Scope

- Conditional profile-expression overlay activation (Phase 3).
- Remote HTTP/API approval workflows for exceptions (deferred per §25).
