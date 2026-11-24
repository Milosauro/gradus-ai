---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T15:16:56Z
last_modified: 2026-11-24T15:16:56Z
approved_fingerprint: sha256:dcc9791030e28303faaaae9b5eec24abf2dc9cb712387bf18c736f5a3c833662
source_requirements_approved_at: 2026-11-24T15:10:10Z
source_requirements_fingerprint: sha256:4cde006c64e91b18e6ace3ae89966ef23bff5dfc0077abd3be896e8f8b0130e6
---

# Design: Action Hardening, Check Coverage, Maintenance & Design 1.1 (`v0.3.2`)

## Architecture
This consolidation release addresses four independent layers without altering `gradus/v1` wire compatibility:
1. **Runtime Guard, Redaction & Exception Validation (`internal/guard`, `internal/state`, `internal/cli`):**
   - Extend `internal/guard/antitamper.go` with regexes for `git -c core.hooksPath=...`, `git push --no-verify`, hook-bypass env vars (`HUSKY=0`, `LEFTHOOK=0`, `SKIP=...`), and protected globs `lefthook-local.yml` and `.lefthook-local.yml`.
   - Extend `internal/state/redact.go` with patterns for truncated PEM private keys (header through end of key body/text) and unspaced 10-digit Italian mobile numbers (`\b3\d{9}\b`).
   - Validate `max_exception_duration` in `newExceptionRequestCmd` (`internal/cli/exception.go`) by resolving the target rule's owning level config before writing the exception record.
2. **Check Engine Edge-Case Tests (`internal/check/engine_test.go`):**
   - Add `TestCheckEngineEdgeCasesV032` covering NUL-byte binary file skipping/handling, large file scanning, complex glob patterns (`**/*.go`, nested paths), and missing/failing external checker commands returning `CheckError` / technical error status.
3. **CI Action, Dependency Maintenance & `PRJ-CI-001` (`action.yml`, `ci.yml`, `release.yml`, `go.mod`, `PRJ-CI-001.md`):**
   - Point `actions/setup-go` in `.github/actions/gradus/action.yml` to `go-version-file: '${{ github.action_path }}/../../../go.mod'` and default `version` to `'v0.3.2'`.
   - Add a `test-action-non-go` job to `.github/workflows/ci.yml` that runs `./.github/actions/gradus` against a temporary non-Go project directory (no `go.mod`).
   - Update all pinned action SHAs in `.github/workflows/ci.yml`, `.github/workflows/release.yml`, and `.github/actions/gradus/action.yml` to current non-deprecated releases.
   - Replace `gopkg.in/yaml.v3` imports with `go.yaml.in/yaml/v3` across the codebase and run `go mod tidy`.
   - Promote `.gradus/levels/project/rules/PRJ-CI-001.md` to `enforcement: check` with a regex pattern matching unpinned `uses:` lines (not followed by a 40-hex SHA) plus `Compliant` and `Non-compliant` examples, and recompile managed targets.
4. **Design 1.1 & Documentation (`docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/plan/phase-2-plan.md`, `README.md`, `CHANGELOG.md`):**
   - Update `docs/design/gradus-design.md` to Version `1.1` as a living specification (behavioral semantics, ADR alignment, Decision Log additions, normative precedence).
   - Align `spec/v1/SPEC.md` with the normative precedence rule and Phase 2 active keys.
   - Add `docs/plan/phase-2-plan.md` matching the structure of `phase-0-plan.md` and `phase-1-plan.md`.
   - Update `README.md` and `CHANGELOG.md` for `v0.3.2`, documenting `--by` + `CODEOWNERS`, `adapter verify` managed-block behavior, `v0.3.0` → `v0.3.1+` `view_hash` recompile note, and transparent validation status.

## Failure Modes And Tradeoffs
- **Unspaced 10-digit phone redaction:** Restricting to Italian mobile prefix `\b3\d{9}\b` avoids false positives on Unix timestamps (currently `17...`) or random 10-digit IDs while catching realistic unspaced mobile numbers like `3331234567`.
- **Truncated PEM redaction:** Matching `-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*` after the full-block pattern ensures complete PEM blocks are redacted cleanly without eating trailing prose, while truncated blocks without `-----END` are still redacted.
- **`PRJ-CI-001` regex check:** Go's RE2 engine does not support negative lookahead, so the regex matches `uses:\s*[^#\s]+@` followed by a ref that is clearly a short/non-SHA tag (e.g. `v\d` or branch name rather than 40 hex chars, specifically `@(?:v[0-9]|main|master|[a-zA-Z0-9._-]{1,39}(?:\s|$))`). We verify this via `gradusctl test` and `gradusctl check` against all repo workflows.

## Simplicity And Elegance Review
- Each fix reuses existing abstractions (`AntiTamperShellReason`, `redactionPatterns`, `check.Engine`, `go.yaml.in/yaml/v3` drop-in API) with zero new dependencies or schema breaks.

## Options Considered
- **Option A (Chosen):** Point `.github/actions/gradus/action.yml` to `${{ github.action_path }}/../../../go.mod`, extend existing guard/redaction/exception helpers in place, migrate `gopkg.in/yaml.v3` to `go.yaml.in/yaml/v3`, and update `gradus-design.md` to Version 1.1 as a living spec.
- **Option B (Rejected):** Hardcode a Go version in `action.yml` (drifts from `go.mod`) or split Design 1.1 into a separate docs-only release (leaves `v0.3.2` documentation out of sync).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/guard/antitamper.go`, `internal/state/redact.go`, `internal/cli/exception.go` (`R1.AC1`–`R1.AC4`) |
| `R2` | `internal/check/engine_test.go` (`R2.AC1`) |
| `R3` | `.github/actions/gradus/action.yml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `go.mod`, `.gradus/levels/project/rules/PRJ-CI-001.md` (`R3.AC1`–`R3.AC3`) |
| `R4` | `docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/plan/phase-2-plan.md`, `README.md`, `CHANGELOG.md` (`R4.AC1`, `R4.AC2`) |
| `NFR1` | Static `CGO_ENABLED=0` binary build and `git` via `os/exec` |
| `NFR2` | Canonical English across all repository files |
| `C1` | Backward-compatible `gradus/v1` schemas and CLI exit codes |
| `C2` | Full Go test suite regression pass |

## Verification Plan
- Unit and CLI tests for anti-tamper, redaction, exception duration validation, and `internal/check` edge cases.
- `go mod tidy -diff`, `go test -count=1 ./...`, `CGO_ENABLED=0 go build`, `gradusctl verify --json`, `gradusctl check --json`, `gradusctl test --json`, `walden validate --all --json`.
