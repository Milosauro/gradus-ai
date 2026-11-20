---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T10:00:35Z
last_modified: 2026-11-20T10:00:35Z
approved_fingerprint: sha256:29a9656f15f4a3621867f4e44e0132706237edf16fe61ad33946bbfe40ffa478
source_requirements_approved_at: 2026-11-20T09:48:28Z
source_requirements_fingerprint: sha256:dbd6945d30689f079b80581f0d2bfed47321988dfd925c8c6273b26fc2161fd5
---

# Feature Design

## Architecture

Feature **F4 (`verify-check-evidence`)** introduces three internal packages (`internal/check`, `internal/evidence`, `internal/hooks`) and four CLI commands (`verify`, `check`, `evidence`, `hooks`):

1. **`internal/cli/verify.go` (`gradusctl verify [--dir] [--json]`):**
   - Calls `resolve.Resolve(dir, resolve.Options{})`:
     - Returns exit code `2` on invalid `gradus.yaml`, tampered `locked: true` level vs `gradus.lock`, or unreachable `fail-closed` source.
     - Collects blocking conflicts (`len(view.Conflicts) > 0`) and expired-exception notes.
   - Calls `compile.PlanCompile(dir, resolved.View, cfg.Targets, skills.FS)` and compares every planned file against on-disk bytes (`os.ReadFile`):
     - Missing or mismatched generated files are recorded as `DriftItem{Path, Reason}` (`missing` | `modified`).
     - Also checks for stale `gradus-*.md` files in `.claude/rules/` and `.agents/rules/` not in `PlanCompile`.
   - Emits `VerifyReport{Verdict, ViewHash, LockHash, Conflicts, Drift, ExpiredExceptions, Notes}` as text or `--json`, returning `0` if clean, `1` if conflicts/drift/expired exceptions exist, `2` on technical errors.

2. **`internal/check` (`engine.go`, `builtin.go`, `external.go`) & `internal/cli/check.go`:**
   - **File discovery (`--staged` / `--base <ref>`):**
     - `--staged`: runs `git diff --cached --name-only --diff-filter=ACMR`.
     - `--base <ref>`: runs `git diff --name-only --diff-filter=ACMR <ref>...HEAD`.
     - If neither flag is given, scans all tracked/unignored files via `git ls-files`.
   - **Rule matching:** Filters `view.Rules` where `Enforcement == "check"` and `!Waived` and at least one changed file matches `r.Scope` via `doublestar.Match`.
   - **Built-in checker (`builtin.go`):** Handles `checker.name` == `"pattern"` or `"regex"`. Compiles `checker.Pattern` (from `Rule.Checker.Pattern` or `Rule.Checker.Config`), scans matched files line-by-line, and emits `Finding{RuleID, Severity, Level, Source, Path, Line, Message, Confidence: 1.0, Status: "open"}`.
   - **External checker plugin protocol (`external.go`):**
     - Resolves `checker.name` via `exec.LookPath`. If missing from `PATH`, returns a fatal error mapped to exit code `2` (never `0`, D-73).
     - Runs `<checker.name>` (passing `--config <checker.config>` if configured, or tool-appropriate flags + scoped file paths via `os/exec` with zero shell interpolation).
     - Exit code `0` -> pass; exit code `1` -> violation findings recorded; exit code `>= 2` or execution failure -> error mapped to exit code `2`.
   - Optionally writes evidence to configured sinks (`--write-evidence`).

3. **`internal/evidence` (`sink.go`) & `internal/cli/evidence.go`:**
   - `Record` struct matches `spec/v1/schema/evidence.schema.json` (§9.9): `schema`, `id` (`ev-YYYYMMDD-HHMMSS-<rand>`), `kind` (`verify|check|review|semantic-conflicts|exception-use`), `timestamp`, `commit` (`git rev-parse --short HEAD`), `base`, `lock_hash`, `view_hash`, `producer`, `findings`, `verdict` (`pass|fail|error`). Zero personal activity metrics (P16).
   - **5 Sinks:**
     - `none`: no-op.
     - `local`: writes `.gradus/evidence/<id>.json` and prunes records older than retention (default `30d`).
     - `ci-artifact`: writes `<sink.path>/<id>.json` (defaulting to `.gradus/ci-artifacts/`).
     - `repo`: writes `<sink.path>/<id>.json` inside the repository.
     - `command`: executes `sh -c <sink.command>` passing serialized JSON on `stdin`.
   - `evidence list [--dir] [--json]`: lists `.gradus/evidence/*.json` sorted newest-first.
   - `evidence write --kind <kind> --verdict <verdict> ...`: constructs and writes a record to active sinks (using `evidence.ci.sinks` when `CI` is set and configured).

4. **`internal/hooks` (`manager.go`) & `internal/cli/hooks.go`:**
   - `DetectManager(rootDir, configured)`: if `configured == "auto"` or `""`, checks `.pre-commit-config.yaml` -> `pre-commit`, `lefthook.yml` -> `lefthook`, `.husky` -> `husky`, else `native`.
   - `RenderSnippet(manager)`: emits the exact YAML/shell snippet invoking `gradusctl verify` and `gradusctl check --staged`.
   - `InstallHooks(rootDir, manager, allowNative)`: idempotently updates `.pre-commit-config.yaml`, `lefthook.yml`, or `.husky/pre-commit`; for `native`, writes `.githooks/pre-commit` and sets `git config core.hooksPath .githooks` ONLY when `allowNative` (`--native`) is explicitly true (D-72).

## Options Considered

- **Built-in vs external checker protocol:** Separating `builtin.go` (`pattern`/`regex`) from `external.go` (`os/exec` subprocess) avoids requiring external binaries for simple regex rules while keeping `gitleaks`/`semgrep` integration clean and enforcing exit code `2` on missing binaries.
- **Drift verification reuse:** Reusing `compile.PlanCompile` directly inside `gradusctl verify` eliminates any divergence between what `compile` writes and what `verify` expects.

## Simplicity And Elegance Review

- All 4 commands share `resolve.Resolve` and `ExitErrorWithCode`.
- `evidence.WriteRecord` is a single function called by `gradusctl evidence write` and optionally by `verify`/`check`.

## Failure Modes And Tradeoffs

- **Missing `git` repository during `check --staged`:** Returns exit code `2` with a clear diagnostic rather than silently passing.
- **External checker exit code conventions:** Some external tools use exit `1` for violations and `>= 2` for errors; `external.go` treats `1` as violation and any other non-zero exit as error (`2`) unless configured otherwise.

## Verification Plan

- **`internal/cli` `TestVerifyCommand`:** Clean pass (`0`), manual edit to `AGENTS.md` -> drift (`1`), tampered locked level -> (`2`).
- **`internal/check` `TestCheckEngine`:** `--staged` diff filtering, built-in `regex` match (`1`), missing external checker binary -> (`2`), external checker script exit `1` -> (`1`).
- **`internal/evidence` `TestEvidenceSinks`:** All 5 sinks (`none`, `local` + retention pruning, `ci-artifact`, `repo`, `command` via stdin), `CI` sink selection, and `evidence list`.
- **`internal/hooks` `TestHooksManager`:** Auto-detection order, idempotent `install` for `lefthook`/`pre-commit`/`husky`, and `native` requiring `--native` consent.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC3`: `internal/cli/verify.go` (`resolve.Resolve` + `compile.PlanCompile` byte comparison) |
| `R2` | `R2.AC1`–`R2.AC4`: `internal/check/{engine,builtin,external}.go` & `internal/cli/check.go` |
| `R3` | `R3.AC1`–`R3.AC3`: `internal/evidence/sink.go` & `internal/cli/evidence.go` |
| `R4` | `R4.AC1`–`R4.AC2`: `internal/hooks/manager.go` & `internal/cli/hooks.go` |
| `NFR1` | Strict exit code `2` on missing checkers, git errors, and lock mismatches |
| `NFR2` | `os/exec` invocations in `internal/check` and `internal/hooks` |
| `NFR3` | `internal/evidence/sink.go` schema omitting all user-activity metrics (P16) |
