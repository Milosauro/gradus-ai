---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T10:09:07Z
last_modified: 2026-11-20T10:15:20Z
approved_fingerprint: sha256:a13f4f96dcfe8fcabcf6174053aa8f2692d3a011c5c59a517dea389faede19a7
source_design_approved_at: 2026-11-20T10:00:35Z
source_design_fingerprint: sha256:29a9656f15f4a3621867f4e44e0132706237edf16fe61ad33946bbfe40ffa478
---

# Implementation Plan

- [x] 1. Implement `internal/evidence` sinks and `gradusctl evidence` CLI
  - [x] 1.1 Extend `internal/schema/{types,parse_config}.go` with `evidence.ci.sinks` and `retention`, and create `internal/evidence/sink.go` implementing `Record` (§9.9), `WriteRecord` for all 5 sinks (`none`, `local` with retention pruning, `ci-artifact`, `repo`, `command` via stdin), and `ListLocalRecords`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestEvidenceSinks$", "./internal/evidence"]
        expect_output: "--- PASS: TestEvidenceSinks"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "NFR3"]
  - [x] 1.2 Create `internal/cli/evidence.go` implementing `gradusctl evidence list` and `gradusctl evidence write` wired into `internal/cli/root.go`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestEvidenceCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestEvidenceCommand"
        covers: ["R3.AC3"]

- [x] 2. Implement `internal/check` engine and `gradusctl check` CLI
  - [x] 2.1 Create `internal/check/{builtin,external,engine}.go` implementing `DiscoverFiles` (`--staged` / `--base <ref>` via `os/exec` `git`), built-in `pattern`/`regex` line scanner, external checker runner (`exec.LookPath` missing -> error; exit `1` -> violations; exit `>=2` -> error), and `RunChecks` with `doublestar` scope matching
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `R2.AC4`, `NFR1`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCheckEngine$", "./internal/check"]
        expect_output: "--- PASS: TestCheckEngine"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "R2.AC4", "NFR1", "NFR2"]
  - [x] 2.2 Create `internal/cli/check.go` implementing `gradusctl check [--dir] [--staged] [--base <ref>] [--write-evidence] [--json]` wired into `internal/cli/root.go`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `R2.AC4`, `NFR1`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCheckCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestCheckCommand"
        covers: ["R2.AC2", "R2.AC4", "NFR1"]

- [x] 3. Implement `gradusctl verify` CLI command
  - [x] 3.1 Create `internal/cli/verify.go` implementing `gradusctl verify [--dir] [--write-evidence] [--json]` checking `resolve.Resolve` integrity/conflicts/expired exceptions and `compile.PlanCompile` byte-for-byte drift & stale files (`0` pass, `1` drift/conflicts, `2` lock tamper/error)
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR1`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestVerifyCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestVerifyCommand"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "NFR1"]

- [x] 4. Implement `internal/hooks` manager integration and `gradusctl hooks` CLI
  - [x] 4.1 Create `internal/hooks/manager.go` implementing `DetectManager` (`pre-commit` -> `lefthook` -> `husky` -> `native`), `RenderSnippet`, and `InstallHooks` (requiring `allowNative == true` before touching `core.hooksPath`)
    - Requirements: `R4.AC1`, `R4.AC2`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestHooksManager$", "./internal/hooks"]
        expect_output: "--- PASS: TestHooksManager"
        covers: ["R4.AC1", "R4.AC2", "NFR2"]
  - [x] 4.2 Create `internal/cli/hooks.go` implementing `gradusctl hooks [print|install] [--dir] [--manager] [--native] [--json]` wired into `internal/cli/root.go` and verify `CGO_ENABLED=0` static build
    - Requirements: `R4.AC1`, `R4.AC2`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestHooksCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestHooksCommand"
        covers: ["R4.AC1", "R4.AC2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -trimpath -o /dev/null ./cmd/gradusctl && echo BUILD_OK"]
        expect_output: "BUILD_OK"
        covers: ["NFR2"]
