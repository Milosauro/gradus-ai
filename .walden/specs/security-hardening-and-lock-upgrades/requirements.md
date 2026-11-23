---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T17:01:17Z
last_modified: 2026-11-23T17:01:17Z
approved_fingerprint: sha256:30d721f3e8dfde9c2040ceb476beade3a525d4b555047b5d5bb019bee48c7016
---

# Requirements Document

## Introduction

This consolidation release (`v0.3.1`) addresses security, correctness, and supply-chain findings identified during external auditing and field testing of `v0.3.0` on Claude Code and Antigravity: preventing AI agents from self-approving governance actions, unifying project `view_hash` without user-level leakage, enabling controlled remote lock upgrades (`--update`), extending anti-tamper coverage, removing stale Phase 0 reserved-key warnings, and hardening GoReleaser / GitHub Action reproducibility.

## Requirements

### R1 Human-Only Governance Guard and Approver Validation

**User Story:** As a security lead, I want Gradus to prevent AI agents from self-approving exceptions, break-glass waivers, conflict dismissals, lesson promotions, or hook reinstalls, and to verify that exception approvers are authorized by the level, so that "agents propose, humans approve" is enforced deterministically.

#### Acceptance Criteria

1. `R1.AC1` WHEN an agent invokes a shell command matching a human-only governance subcommand (`gradusctl exception approve`, `gradusctl exception break-glass`, `gradusctl conflicts dismiss`, `gradusctl lessons update`, or `gradusctl hooks install`), `gradusctl guard` SHALL trigger anti-tamper enforcement using `anti_tamper.mode` (`ask` by default, `deny` when configured).
   - Acceptance check: Sending `gradusctl exception approve EX-1 --by me` or `gradusctl conflicts dismiss F-1` to `gradusctl guard` returns `anti_tamper: true` with `ask` (or `deny` under `anti_tamper.mode: deny`), never `allow`.
2. `R1.AC2` IF `gradusctl exception approve <id> --by <approver>` is invoked for a rule whose level declares `exceptions.approvers`, `level.yaml` `approvers`/`owners`, or `gradus.yaml` `levels[].approvers` and `<approver>` is not in the authorized set, THEN `gradusctl exception approve` SHALL reject the approval with exit code `2` and leave the exception file unchanged.
   - Acceptance check: Approving an exception on a level owned by `security-team` with `--by random-dev` fails with exit code `2` while `--by security-team` succeeds.

### R2 User-Free `view_hash` and Controlled Remote Lock Upgrades

**User Story:** As a team member with personal user-level rules, I want `gradus.lock` and compiled project markers to use a project-only `view_hash` independent of `~/.gradus/levels/user`, and I want an explicit `--update` flag to upgrade locked remote levels when their `ref` changes in `gradus.yaml`.

#### Acceptance Criteria

1. `R2.AC1` WHEN `resolve.Resolve` computes `ViewHash` for `ResolvedView` and `gradus.lock`, the hash SHALL exclude `user`-level rules and `user`-level summaries so that `resolve --write`, `compile`, and `verify` produce identical `view_hash` values regardless of personal `~/.gradus/levels/user` rules.
   - Acceptance check: Adding a rule to `~/.gradus/levels/user` does not change `gradus.lock` `view_hash` or trigger false drift in `gradusctl verify`.
2. `R2.AC2` WHEN `gradusctl resolve --write --update` runs on a pyramid where a `locked: true` git level's `source.ref` in `gradus.yaml` differs from `gradus.lock`, the CLI SHALL accept the newly resolved commit and hash and update `gradus.lock`.
   - Acceptance check: Changing `ref: v1.0.0` to `ref: v1.1.0` on a locked git level succeeds with `gradusctl resolve --write --update` and writes the new commit and hash to `gradus.lock`.
3. `R2.AC3` IF a `locked: true` level's `source.ref` has NOT changed relative to `gradus.lock` (or `--update` is not passed) and its content hash differs from `gradus.lock`, THEN `gradusctl resolve` SHALL fail with exit code `2` reporting a lock integrity mismatch.
   - Acceptance check: Tampering with a locked level at the same ref fails with exit code `2` even when `--update` is passed.

### R3 Extended Anti-Tamper Protection

**User Story:** As a repository maintainer, I want anti-tamper protection to cover git hook manager configs, CI workflows, new agent configs, `install` shell writes, and `git commit --no-verify`, so that agents cannot silently disable local or CI checks.

#### Acceptance Criteria

1. `R3.AC1` WHEN a write/edit tool or mutating shell command (including `install`) targets `.pre-commit-config.yaml`, `lefthook.yml`, `.lefthook.yml`, `.husky/**`, `.github/workflows/**`, `opencode.json`, or `.codex/config.toml`, `gradusctl guard` SHALL trigger anti-tamper enforcement (`ask` or `deny` per `anti_tamper.mode`).
   - Acceptance check: Editing `.pre-commit-config.yaml`, `.github/workflows/ci.yml`, `opencode.json`, or running `install -m644 /dev/null gradus.yaml` returns `anti_tamper: true`.
2. `R3.AC2` WHEN a shell command invokes `git commit` with `--no-verify` or `-n`, `gradusctl guard` SHALL trigger anti-tamper enforcement (`ask` or `deny` per `anti_tamper.mode`) even if no explicit user hook rule is configured.
   - Acceptance check: Evaluating `git commit --no-verify -m "skip"` returns `anti_tamper: true` with a reason citing `--no-verify`.

### R4 Schema Cleanup, Reproducible Supply Chain, and Documentation Alignment

**User Story:** As a user installing signed `gradusctl` releases or using the GitHub Action, I want clean non-dirty signed binaries, pinned Action defaults, accurate schema notes, and up-to-date `v0.3.1` documentation.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradus.yaml` or `level.yaml` uses active Phase 1/2 keys (`obligations`, `bindings`, `toolchain`, `signature`, `min_version`, `charter`, `context_budget`, `exceptions`, `anti_tamper`), configuration parsing SHALL NOT emit `"is not active in Phase 0"` notes.
   - Acceptance check: Resolving a pyramid with `obligations`, `bindings`, and `toolchain` produces zero `"Phase 0"` notes in `gradusctl resolve --json`.
2. `R4.AC2` The repository SHALL maintain a clean `go.mod` verified by `go mod tidy -diff` in `.goreleaser.yaml` (preventing `+dirty` release binaries) and pin `.github/actions/gradus/action.yml` to a specific version default with `go-version-file: go.mod` and `env:`-passed inputs.
   - Acceptance check: `go mod tidy -diff` exits `0`, `.goreleaser.yaml` uses `go mod tidy -diff`, and `.github/actions/gradus/action.yml` does not default to `@latest` or inline `${{ inputs.* }}` in shell scripts.
3. `R4.AC3` `gradusctl handbook` SHALL number sections sequentially without skipping section `2.`.
   - Acceptance check: `gradusctl handbook` contains sequential sections `## 1.`, `## 2.`, `## 3.`.
4. `R4.AC4` The repository SHALL update `README.md`, `CHANGELOG.md`, and `.gradus/levels/project/rules/PRJ-GO-001.md` to document `v0.3.1` capabilities and provide runnable `gradusctl test` examples.
   - Acceptance check: `gradusctl test` on the `gradus-ai` repository executes at least 1 passing rule test case and `README.md` reflects Phase 2 (`v0.3.1`).

## Non-Functional Requirements

- `NFR1` **Static Build & Zero New Dependencies (`PRJ-GO-001`):** All changes MUST compile with `CGO_ENABLED=0` and zero new third-party dependencies.
- `NFR2` **Fail-Closed Safety:** All new guard and approver checks MUST fail closed (never silently downgrade to `allow`).

## Constraints And Dependencies

- `C1` Must preserve `gradus/v1` backward compatibility for existing repositories and lockfiles.
- `C2` Must reuse existing `internal/guard`, `internal/resolve`, `internal/schema`, and `internal/cli` packages.

## Out Of Scope

- OpenCode `tool.execute.before` plugin compiler generation (tracked for Phase 3).
- Migrating from `gopkg.in/yaml.v3` to another YAML library.
