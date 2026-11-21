---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:03:40Z
last_modified: 2026-11-21T17:03:40Z
approved_fingerprint: sha256:15c5efbf4e16864c3b28736a44e2cf3dfec468234b0f0ab555b755c375ce0ebb
source_requirements_approved_at: 2026-11-21T17:02:13Z
source_requirements_fingerprint: sha256:8acc43f9c9b49df9e244047cfa070141d351e282fa3388ce3be7d8688a0f105d
---

# Feature Design

## Architecture

### 1. Anti-Tamper Policy (`internal/schema/`, `internal/guard/`, `internal/cli/guard.go`, `spec/v1/schema/`)
- **Schema (`internal/schema/types.go`, `parse_config.go`, `validate_config.go`, `spec/v1/schema/gradus.schema.json`):**
  - Add `AntiTamper *AntiTamperConfig` (`yaml:"anti_tamper,omitempty" json:"anti_tamper,omitempty"`) to `PyramidConfig` with `Mode string` (`"ask"` | `"deny"`).
  - Register `"anti_tamper"` (`"mode"`) in `pyramidFieldSpec` and `gradus.schema.json`.
  - Reject any non-empty `anti_tamper.mode` other than `"ask"` or `"deny"` in `ValidatePyramidConfig` (exit `2`).
- **Guard evaluation (`internal/guard/evaluator.go`, `internal/cli/guard.go`):**
  - Add `EvaluateWithOptions(action NormalizedAction, rules []resolve.ResolvedRule, opts EvaluateOptions) (EvaluationResult, error)` where `EvaluateOptions.AntiTamperMode` defaults to `DecisionAsk` and switches the anti-tamper floor to `DecisionDeny` when set to `DecisionDeny`.
  - Keep `Evaluate(action, rules)` calling `EvaluateWithOptions` with default options for backward compatibility.
  - Add `LoadAntiTamperMode(rootDir string) string` reading `gradus.yaml`'s `anti_tamper.mode` (fast local read, `< 1ms`, defaulting to `ask`).
  - In `internal/cli/guard.go`, pass `AntiTamperMode` to `guard.EvaluateWithOptions` so native Claude Code (`permissionDecision: "deny"`, exit `0`), native Antigravity (`decision: "deny"`, exit `0`), and generic CLI (`decision: "deny"`, exit `1`) all honor `anti_tamper.mode: deny`.

### 2. `gradusctl doctor` (`internal/cli/doctor.go`, registered in `internal/cli/root.go`)
- Runs a deterministic list of diagnostic checks (`doctorCheck{Name, Status, Message, Fix}`):
  1. `gradus_yaml`: parses and validates `gradus.yaml`.
  2. `gradus_lock`: checks `gradus.lock` presence and valid schema.
  3. `gradusctl_path`: checks `exec.LookPath("gradusctl")` (or warns if not on `PATH` when hooks are configured).
  4. `claude_settings` (when `claude-code` target is configured or `.claude/settings.json` exists): parses `.claude/settings.json`, validates every hook object's `type` against `command | prompt | agent | http | mcp_tool`, and checks that `hooks.PreToolUse` contains a `gradusctl guard` command.
  5. `antigravity_hooks` (when `antigravity` target is configured or `.agents/hooks.json` exists): parses `.agents/hooks.json` and verifies that `PreToolUse` contains a `gradusctl guard` entry.
- Outputs structured JSON (`verdict: "pass" | "fail"`, `checks: [...]`) or human-readable checklist; exits `0` when all checks pass, `1` when any check fails.

### 3. Hygiene Commands & User-Rule Visibility (`internal/state/`, `internal/cli/`)
- **`gradusctl evidence prune [--before YYYY-MM-DD | --all] [--dir .] [--json]` (`internal/cli/evidence.go`):**
  - Deletes matching `.gradus/evidence/*.json` files (by `timestamp` prefix `< before`, or all when `--all` is set) and returns `{"removed": N, "remaining": M}`.
- **`gradusctl lessons prune [--dismissed | --all] [--dir .] [--store-dir ...] [--json]` (`internal/state/lessons.go`, `internal/cli/lessons.go`):**
  - Deletes dismissed (or all) lesson JSON files from the lessons store and returns `{"removed": N, "remaining": M}`.
- **`gradusctl status` user-rule visibility (`internal/cli/status.go`):**
  - Add optional `--user-dir` flag (default `~/.gradus/levels/user`, skipped in CI) and populate `UserRules []string` (`json:"user_rules,omitempty"`) in `statusReport` using fast local rule file parsing, keeping `status --brief` `< 100ms`.

## Options Considered

- **Chosen: Keep `Evaluate(action, rules)` unchanged and add `EvaluateWithOptions(action, rules, opts)`.**
  - Preserves 100% source compatibility with existing Phase 0 unit tests and benchmarks while enabling `anti_tamper.mode: deny`.
- **Alternative rejected: Change `Evaluate` signature across all callers.**
  - Unnecessary churn with zero functional benefit.

## Simplicity And Elegance Review

- `gradusctl doctor` is read-only and deterministic (no network calls, no silent mutations), surfacing exact root causes like unknown hook types (`"commbnd"`) with copy-pasteable fix commands (`rm .claude/settings.json && gradusctl compile`).
- `evidence prune` and `lessons prune` require explicit scope flags (`--all`, `--before`, or `--dismissed`) so users never accidentally delete state.

## Failure Modes And Tradeoffs

- **Missing scope flag on `prune`:** Fails closed with exit `2` and usage guidance rather than deleting anything by default.
- **`gradusctl` not in `PATH` during `go test`:** `doctor` checks `exec.LookPath("gradusctl")`; in unit tests where `gradusctl` may not be installed in system `PATH`, `doctor` treats missing `PATH` as a warning unless `--strict-path` is passed (or checks `os.Executable()` fallback), while malformed hook JSON is always a hard failure (`exit 1`).

## Verification Plan

- Unit tests in `internal/guard` and `internal/cli` verifying `anti_tamper.mode: ask` vs `deny` across generic, native Claude Code, and native Antigravity payloads.
- CLI integration tests in `internal/cli` verifying `gradusctl doctor` on healthy repos, corrupted `.claude/settings.json` (`"type": "commbnd"`), and corrupted `.agents/hooks.json`.
- CLI integration tests in `internal/cli` verifying `evidence prune`, `lessons prune`, and `status --json` `user_rules`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC3`: `internal/cli/doctor.go` checking `gradus.yaml`, `gradus.lock`, `.claude/settings.json` hook types & guard entry, and `.agents/hooks.json` |
| `R2` | `R2.AC1`–`R2.AC3`: `AntiTamperConfig` in `internal/schema/`, `EvaluateWithOptions` in `internal/guard/evaluator.go`, and `internal/cli/guard.go` |
| `R3` | `R3.AC1`–`R3.AC3`: `evidence prune`, `lessons prune`, and `statusReport.UserRules` in `internal/cli/` and `internal/state/` |
| `NFR1` | Optional `anti_tamper` defaulting to `ask`; existing tests unchanged |
| `NFR2` | Local offline reads `< 100ms`, canonical English, `CGO_ENABLED=0` |
| `C1` | `anti_tamper.mode` registered in `spec/v1/schema/gradus.schema.json` and `internal/schema/parse_config.go` |
