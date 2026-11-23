---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T17:03:14Z
last_modified: 2026-11-23T17:03:14Z
approved_fingerprint: sha256:40c7912ba8ba8bafe66a5abf4def02b9003d2999d2a62f77e8d422bcd8e97f41
source_requirements_approved_at: 2026-11-23T17:01:17Z
source_requirements_fingerprint: sha256:30d721f3e8dfde9c2040ceb476beade3a525d4b555047b5d5bb019bee48c7016
---

# Feature Design

## Architecture

This consolidation release hardens five existing subsystems without adding third-party dependencies:

1. **Human-Only Governance Guard & Extended Anti-Tamper (`internal/guard/antitamper.go`):**
   - Add `humanOnlyGradusCmdRe` matching shell invocations of `gradusctl` subcommands reserved for humans (`exception approve`, `exception break-glass`, `conflicts dismiss`, `lessons update`, `hooks install`).
   - Add `gitCommitNoVerifyRe` matching `git` `commit` commands containing `--no-verify` or standalone `-n`.
   - Extend `protectedPathGlobs` and `protectedShellTargetsRe` to cover `.pre-commit-config.yaml`, `lefthook.yml`, `.lefthook.yml`, `.husky/**`, `.github/workflows/**`, `opencode.json`, and `.codex/config.toml`.
   - Add `install` to `mutatingShellVerbRe`.
   - All matches return `anti_tamper: true` and respect `anti_tamper.mode` (`ask` by default, `deny` in strict mode, `NFR2`).

2. **Authorized Approver Validation (`internal/cli/exception.go`):**
   - In `gradusctl exception approve <id> --by <approver>`, resolve the pyramid to locate the target rule's level, collect authorized approvers from `level.yaml` (`exceptions.approvers`, `approvers`, `owners`) and `gradus.yaml` (`levels[].approvers`).
   - If the authorized set is non-empty and `<approver>` is not listed, fail closed with exit code `2` before modifying `.gradus/exceptions/<id>.yaml`.

3. **User-Free `view_hash` & `--update` Lock Upgrades (`internal/resolve/resolve.go`, `internal/cli/resolve.go`):**
   - In `resolve.Resolve`, compute `view.ViewHash` over a deterministic projection that strips `user`-level entries from `Levels`, `Rules`, `InactiveRules`, and sets `Scope.IncludesUserLevel = false`, ensuring `resolve --write`, `compile`, and `verify` always compute the exact same project `view_hash` regardless of `~/.gradus/levels/user` (`R2.AC1`).
   - Add `AllowLockUpdate bool` to `resolve.Options` and `--update` flag on `gradusctl resolve`.
   - During level resolution in `resolve.Resolve`: when `lvl.Locked && lockEntry != nil`, allow skipping `VerifyLockedEntry` **only** when `opts.AllowLockUpdate && lvl.Source.Git != "" && lockEntry.Ref != "" && lockEntry.Ref != lvl.Source.Ref`. If the ref did not change (or `--update` is false), `VerifyLockedEntry` still fails closed with exit code `2` (`R2.AC2`, `R2.AC3`).

4. **Reserved-Key Cleanup, Supply Chain & Docs (`internal/schema/unknown.go`, `go.mod`, `.goreleaser.yaml`, `.github/actions/gradus/action.yml`, `internal/compile/handbook.go`, `README.md`, `CHANGELOG.md`, `.gradus/levels/project/rules/PRJ-GO-001.md`):**
   - Remove active Phase 1/2 keys (`obligations`, `bindings`, `toolchain`, `signature`, `min_version`, `charter`, `context_budget`, `exceptions`, `anti_tamper`) from `reservedPyramidKeys` / `reservedLevelKeys` in `internal/schema/unknown.go`.
   - Run `go mod tidy` on `go.mod` and replace `go mod tidy` with `go mod tidy -diff` in `.goreleaser.yaml`.
   - Update `.github/actions/gradus/action.yml` to use `go-version-file: go.mod`, default `version: "v0.3.1"` (no `@latest`), and pass inputs via `env:`.
   - Fix `handbook.go` heading numbering (`## 1.`, `## 2.`, `## 3.`, `## 4.`).
   - Add `### Compliant` and `### Non-compliant` examples to `.gradus/levels/project/rules/PRJ-GO-001.md`, update `README.md` to Phase 2 (`v0.3.1`) with explicit live-validation scope, and document `v0.3.1` in `CHANGELOG.md`.

## Options Considered

- **Chosen:** Project-only `view_hash` projection inside `resolve.Resolve` + explicit `--update` flag on `gradusctl resolve`.
  - Keeps a single `ViewHash` field across `ResolvedView`, `gradus.lock`, and managed markers (`AGENTS.md`/`CLAUDE.md`) while keeping `user` rules available in `ResolvedView.Rules` for local `compile`/`guard`.
- **Alternative:** Maintain two separate hashes (`project_view_hash` and `local_view_hash`).
  - Rejected: complicates lockfile and marker schemas unnecessarily when `gradus.lock` by design never includes personal user rules.

## Simplicity And Elegance Review

- Reuses existing `anti_tamper.mode` machinery for human-only `gradusctl` subcommands and `git commit --no-verify` so both Claude Code and Antigravity inherit the protection automatically without hook schema changes.
- Keeping `--update` strictly gated on `lockEntry.Ref != lvl.Source.Ref` guarantees that tampering at a pinned ref is still caught even if `--update` is passed.

## Failure Modes And Tradeoffs

- **Authorized approver list omitted on personal repos:** When a level declares no `exceptions.approvers`, `approvers`, or `owners`, `exception approve --by <name>` continues to accept `<name>` so minimal personal setups do not break (`C1`).
- **PR review remains the ultimate approval boundary:** Documentation explicitly notes that local CLI/hook enforcement is complemented by PR review + `CODEOWNERS` on `.gradus/exceptions/`.

## Verification Plan

- Unit tests in `internal/guard/guard_test.go` covering human-only `gradusctl` subcommands, `git commit --no-verify`/`-n`, `.pre-commit-config.yaml`, `.github/workflows/ci.yml`, `opencode.json`, `.codex/config.toml`, and `install -m644 /dev/null gradus.yaml` (`R1.AC1`, `R3.AC1`, `R3.AC2`, `NFR2`).
- CLI tests in `internal/cli/hardening_test.go` covering unauthorized vs authorized `exception approve --by`, user-level invariance of `view_hash`, `resolve --write --update` on ref change vs same-ref tamper, zero `"Phase 0"` notes on `obligations`/`bindings`/`toolchain`, sequential handbook numbering, and `gradusctl test` on `gradus-ai` (`R1.AC2`, `R2.AC1`–`R2.AC3`, `R4.AC1`–`R4.AC4`).
- `go mod tidy -diff` and `CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl` (`R4.AC2`, `NFR1`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/guard/antitamper.go`, `internal/cli/exception.go` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/resolve/resolve.go`, `internal/cli/resolve.go` (`R2.AC1`–`R2.AC3`) |
| `R3` | `internal/guard/antitamper.go` (`R3.AC1`, `R3.AC2`) |
| `R4` | `internal/schema/unknown.go`, `go.mod`, `.goreleaser.yaml`, `.github/actions/gradus/action.yml`, `internal/compile/handbook.go`, `README.md`, `CHANGELOG.md`, `PRJ-GO-001.md` (`R4.AC1`–`R4.AC4`) |
| `NFR1` | `CGO_ENABLED=0 go build` + `go mod tidy -diff` |
| `NFR2` | Fail-closed anti-tamper and approver checks tested in `internal/guard` and `internal/cli` |
| `C1` | Backward-compatible behavior when `approvers`/`owners` are omitted or `--update` is unused |
| `C2` | Reuses existing `internal/guard`, `internal/resolve`, `internal/schema`, `internal/cli` packages |
