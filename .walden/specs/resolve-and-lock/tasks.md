---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-19T23:23:27Z
last_modified: 2026-11-19T23:30:17Z
approved_fingerprint: sha256:fec3744f9ace47ff905e9cc48cd35088d4656cca4447611ba91017f64137c381
source_design_approved_at: 2026-11-19T22:43:33Z
source_design_fingerprint: sha256:08463f52e6ecf2a2a8789d62f70c8f6b186349d43d101d3fb6161d3fa3fe1921
---

# Implementation Plan

- [x] 1. Implement `internal/lock` normalized cross-platform hashing and `gradus.lock` management
  - [x] 1.1 Create `internal/lock/hash.go` implementing `HashBytes(data []byte) string` and `HashDirectory(dir string) (string, error)` (skips `.git/` and `.fetched_at`, sorts `filepath.ToSlash` relative paths, normalizes `\r\n` to `\n`, outputs `"sha256:" + hex`)
    - Requirements: `R2.AC1`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestHashDirectoryAndBytes$", "./internal/lock"]
        expect_output: "--- PASS: TestHashDirectoryAndBytes"
        covers: ["R2.AC1"]
  - [x] 1.2 Create `internal/lock/lockfile.go` implementing `Lockfile`, `Entry`, `ReadLockfile(path string)`, `WriteLockfile(path string, lf Lockfile)` (excluding `id == "user"`), and `VerifyLockedEntry(entry Entry, actualHash string) error`
    - Requirements: `R2.AC2`, `R2.AC3`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestLockfileReadWriteAndVerify$", "./internal/lock"]
        expect_output: "--- PASS: TestLockfileReadWriteAndVerify"
        covers: ["R2.AC2", "R2.AC3"]

- [x] 2. Implement `internal/source` local path and `os/exec` Git source fetchers with offline policy
  - [x] 2.1 Create `internal/source/local.go` implementing `ResolvedSource` and `ResolveLocalSource(rootDir, kind, id, rawPath string, locked, inCI bool) (ResolvedSource, error)` (skips `~/` with a note when `inCI == true`)
    - Requirements: `R1.AC1`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveLocalSource$", "./internal/source"]
        expect_output: "--- PASS: TestResolveLocalSource"
        covers: ["R1.AC1"]
  - [x] 2.2 Create `internal/source/git.go` implementing `ResolveGitSource` using `os/exec` `git` into `.gradus/cache/<id>/repo` with `.fetched_at` timestamp and `offline.mode` (`cache` within `max_age` vs `fail-closed`)
    - Requirements: `R1.AC2`, `R1.AC3`, `R1.AC4`, `NFR2`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveGitSourceAndOfflinePolicy$", "./internal/source"]
        expect_output: "--- PASS: TestResolveGitSourceAndOfflinePolicy"
        covers: ["R1.AC2", "R1.AC3", "R1.AC4", "NFR2"]

- [x] 3. Implement `internal/resolve` §10 resolution algorithm, inherited constraints, and P2 restriction invariant
  - [x] 3.1 Create `internal/resolve/types.go` (§9.7 `ResolvedView` structs) and `internal/resolve/constraints.go` (`CheckInheritedConstraints` for `require_levels`/`max_extra_levels`/`allowed_overlays` and `OrderPyramidSteps` inserting active overlays after `strength`)
    - Requirements: `R3.AC1`, `R3.AC2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestConstraintsAndOverlayOrdering$", "./internal/resolve"]
        expect_output: "--- PASS: TestConstraintsAndOverlayOrdering"
        covers: ["R3.AC2"]
  - [x] 3.2 Create `internal/resolve/invariant.go` implementing `CheckRestrictionInvariant` (ADR 0001 committed-to-`user` override error, severity/enforcement/`guard.decision` rank checks, `doublestar` scope containment check, `overridable.Allowed`, and `locked` + `editable_rules`)
    - Requirements: `R3.AC3`, `R3.AC4`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestRestrictionInvariantAndADR0001$", "./internal/resolve"]
        expect_output: "--- PASS: TestRestrictionInvariantAndADR0001"
        covers: ["R3.AC3", "R3.AC4"]
  - [x] 3.3 Create `internal/resolve/resolve.go` implementing `Resolve(rootDir string, opts Options) (ResolvedView, error)` (combines source resolution, lock verification, `retired`/`draft` filtering, `.gradus/exceptions/*.yaml` waivers, rule merge, `.gradus/findings/*.yaml` confirmed conflicts, `view_hash`, and `budget`)
    - Requirements: `R3.AC1`, `R3.AC5`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestFullResolvePipeline$", "./internal/resolve"]
        expect_output: "--- PASS: TestFullResolvePipeline"
        covers: ["R3.AC1", "R3.AC5"]

- [x] 4. Implement `gradusctl resolve` and `gradusctl status` CLI commands
  - [x] 4.1 Create `internal/cli/resolve.go` implementing `gradusctl resolve [--dir] [--write] [--json]` (writes `gradus.lock` on `--write`, emits `ResolvedView`, exits `0` when clean, `1` on conflicts, `2` on errors)
    - Requirements: `R2.AC2`, `R2.AC3`, `R4.AC1`, `R4.AC2`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestResolveCommand"
        covers: ["R4.AC1", "R4.AC2", "NFR3"]
  - [x] 4.2 Create `internal/cli/status.go` implementing `gradusctl status [--dir] [--brief] [--json]` (zero-network local summary `< 100ms`) and verify `CGO_ENABLED=0` static build
    - Requirements: `R4.AC3`, `NFR1`, `NFR2`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestStatusCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestStatusCommand"
        covers: ["R4.AC3", "NFR1"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -trimpath -o /dev/null ./cmd/gradusctl && echo BUILD_OK"]
        expect_output: "BUILD_OK"
        covers: ["NFR2"]
