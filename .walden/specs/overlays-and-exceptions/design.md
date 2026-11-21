---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T16:44:54Z
last_modified: 2026-11-21T16:44:54Z
approved_fingerprint: sha256:e055198d41f2b96c266157489cc2c5c6a5ddeed4ed1db304b5989df1f12fb562
source_requirements_approved_at: 2026-11-21T16:37:47Z
source_requirements_fingerprint: sha256:b38dc9c305e3413c41af2617d6819b1074f1558d2b69b69cee8f02bdaab4da0e
---

# Feature Design

## Architecture

### 1. Explicit Overlay Activation (`internal/schema/`, `internal/resolve/`, `spec/v1/schema/`)
- **Schema validation (`internal/schema/validate_config.go`, `spec/v1/schema/overlay.schema.json`):**
  - Extend allowed `activation` values on `OverlayRef` and `OverlayConfig` to `""`, `"always"`, `"explicit"`, `"disabled"`, and `"profile"`.
- **Resolution (`internal/resolve/resolve.go`):**
  - Treat `ov.Activation == "" || ov.Activation == "always" || ov.Activation == "explicit"` as `active = true`.
  - Treat `ov.Activation == "disabled"` (or `"profile"` when not matched) as `active = false` (recorded in `ResolvedView.Overlays` with `Active: false`, rules not loaded into `ResolvedView.Rules`).
  - Existing `CheckInheritedConstraints` enforces `constraints.allowed_overlays` across active overlays.

### 2. Exception Policy & Break-Glass Resolution (`internal/resolve/`)
- **Extend `exceptionRecord` & `ExceptionView` (`internal/resolve/resolve.go`, `internal/resolve/types.go`):**
  - Add `PostReviewDue string` (`yaml:"post_review_due,omitempty"` / `json:"post_review_due,omitempty"`) and `Created string` (`json:"created,omitempty"`), `Reason string` (`json:"reason,omitempty"`).
- **Pending vs Active vs Expired vs Policy Conflicts (`loadExceptions` + `validateExceptionPolicies`):**
  - `kind == "exception"` with `len(approved_by) == 0` is recorded as a non-waiving pending note (does NOT waive the target rule).
  - `expires < todayUTC` is recorded as an expired exception note (already handled as `ExpiredExceptions` in `gradusctl verify`, exit `1`).
  - For active exceptions (`exception` with approvers, or `break-glass`), look up the target rule's owning level (`levelConfigs[rule.Level]` and `cfg.Levels` `Offline`):
    - **Max duration:** Parse `exceptions.max_duration` (`<N>d` or `<N>h`, e.g., `30d`, `72h`) and compare against `expires - created`; if exceeded, emit `Conflict{Kind: "exception", RuleID, Level, Message}`.
    - **Break-glass rules:** If `kind == "break-glass"`:
      - Require non-empty `reason` and valid `YYYY-MM-DD` `post_review_due`.
      - If `post_review_due < todayUTC`, emit `Conflict{Kind: "exception", RuleID, Level, Message}` (overdue post-review).
      - If target level sets `offline.mode == "fail-closed"` OR `(levelConfig.Exceptions != nil && !levelConfig.Exceptions.BreakGlass)`, emit `Conflict{Kind: "exception", RuleID, Level, Message}`.

### 3. CLI Exception State & Subcommands (`internal/state/exceptions.go`, `internal/cli/exception.go`)
- **State module (`internal/state/exceptions.go`):**
  - `ExceptionRecord` struct matching `spec/v1/schema/exception.schema.json`.
  - `ListExceptions(rootDir, now)` computing state (`pending` | `active` | `expired`), `WriteException(rootDir, rec)`, `ApproveException(rootDir, id, approver)`, `ExpireException(rootDir, id, now)` (sets `expires` to `now.AddDate(0, 0, -1).Format("2006-01-02")`).
- **CLI command (`internal/cli/exception.go`, registered in `internal/cli/root.go`):**
  - `gradusctl exception request --rule <id> --reason <text> --expires <YYYY-MM-DD> [--scope ...] [--dir .]`
  - `gradusctl exception approve <id> --by <approver> [--dir .]`
  - `gradusctl exception break-glass --rule <id> --reason <text> --expires <YYYY-MM-DD> --post-review-due <YYYY-MM-DD> --by <actor> [--scope ...] [--dir .]` (validates target level policy before writing; returns exit `1` without writing if disallowed or exceeding `max_duration`)
  - `gradusctl exception list [--dir .] [--json]`
  - `gradusctl exception expire <id> [--dir .]`

## Options Considered

- **Chosen: Store exception records directly as YAML files in `.gradus/exceptions/<id>.yaml` owned by `internal/state/exceptions.go` and validated during `resolve.Resolve`.**
  - Matches `spec/v1/schema/exception.schema.json` and existing `loadExceptions` in `internal/resolve/resolve.go` with zero migration needed.
- **Alternative rejected: Introduce a separate `.gradus/exceptions.db` index file.**
  - Rejected because YAML files per exception are git-reviewable in pull requests and already specified by `gradus/v1`.

## Simplicity And Elegance Review

- Reuses existing `schema.ExceptionPolicy` (`max_duration`, `approvers`, `break_glass`) and `spec/v1/schema/exception.schema.json`.
- Keeps `gradusctl exception` strictly as the state owner for `.gradus/exceptions/*.yaml`, consistent with Golden Rule #3.
- Duration parser supports clean `<N>d` / `<N>h` strings using Go stdlib only.

## Failure Modes And Tradeoffs

- **Disallowed `break-glass`:** Both `gradusctl exception break-glass` (at creation time) and `gradusctl resolve`/`verify` (at evaluation time) enforce `break_glass: false` and `fail-closed` so manual YAML tampering cannot bypass the policy.
- **Invalid dates (`expires`, `post_review_due`):** Rejected with exit `2` at CLI creation and flagged during resolution.
- **Clock skew on `expire`:** `exception expire <id>` sets `expires` to yesterday UTC (`now.UTC().AddDate(0, 0, -1)`), ensuring immediate deterministic expiration across all timezones.

## Verification Plan

- Unit tests in `internal/resolve` verifying `activation: explicit` vs `disabled`, pending unapproved exceptions, `max_duration` violation, `break-glass` policy checks (`break_glass: false`, `fail-closed`, overdue `post_review_due`).
- CLI integration tests in `internal/cli` verifying `gradusctl exception request`, `approve`, `break-glass`, `list --json`, `expire`, and `gradusctl verify` exit codes.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `activation: explicit | always | disabled | profile` validation in `internal/schema/validate_config.go` and overlay activation + `allowed_overlays` checks in `internal/resolve/resolve.go` |
| `R2` | `R2.AC1`–`R2.AC3`: Pending unapproved exception handling, `max_duration` span check, and `break-glass` validation (`reason`, `post_review_due`, `break_glass: false`, `fail-closed`) in `internal/resolve/resolve.go` |
| `R3` | `R3.AC1`–`R3.AC4`: `internal/state/exceptions.go` and `internal/cli/exception.go` implementing `request`, `approve`, `break-glass`, `list`, and `expire` |
| `NFR1` | Backward-compatible YAML decoding for existing `.gradus/exceptions/*.yaml` records |
| `NFR2` | Canonical English messages/code and `CGO_ENABLED=0` static build |
| `C1` | Conformance with `spec/v1/schema/exception.schema.json` and `spec/v1/schema/overlay.schema.json` |
