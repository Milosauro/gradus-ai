---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T11:05:47Z
last_modified: 2026-11-20T11:08:46Z
approved_fingerprint: sha256:f63dafe61da67554d0b6b2dc6772db2eb60217e6943fee638cea7458a8a0ee30
source_design_approved_at: 2026-11-20T11:04:10Z
source_design_fingerprint: sha256:efdac13fe2edbc12872284d62d723965fb24f9014f76fc541468247e53e347b0
---

# Implementation Plan

- [x] 1. Payload normalization and built-in anti-tamper protection
  - [x] 1.1 Implement `ParsePayload` and `NormalizeTargetPath` in `internal/guard/payload.go` (Claude Code `Bash`/`Edit`/`Write`/`MultiEdit` and Antigravity `run_command`/`write_to_file`/`replace_file_content`/`multi_replace_file_content` normalization, cross-platform slash/root path cleanup, fail-closed error on empty/malformed JSON) and add `TestParsePayload` in `internal/guard/guard_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR2`
    - Design: `internal/guard/payload.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestParsePayload$", "./internal/guard"]
        expect_output: "--- PASS: TestParsePayload"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "NFR2"]
  - [x] 1.2 Implement `IsProtectedGovernancePath` and built-in anti-tamper `ask` floor (with `deny` escalation) in `internal/guard/antitamper.go` for `gradus.yaml`, `gradus.lock`, `.gradus/**`, `.walden/**`, `.agents/hooks.json`, and `.claude/settings.json`, and add `TestAntiTamperProtection` in `internal/guard/guard_test.go`
    - Requirements: `R2.AC1`, `R2.AC2`, `NFR2`
    - Design: `internal/guard/antitamper.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAntiTamperProtection$", "./internal/guard"]
        expect_output: "--- PASS: TestAntiTamperProtection"
        covers: ["R2.AC1", "R2.AC2", "NFR2"]

- [x] 2. Local offline guard evaluation engine and user-level loading
  - [x] 2.1 Add `OfflineOnly` cache-only fast-path to `internal/source/git.go` and `internal/resolve/resolve.go`, implement `Evaluate` and `LoadLocalGuardRules` (including `~/.gradus/levels/user`, `deny > ask > allow` precedence, regex/glob matching, waived-rule skipping, `< 100ms` latency) in `internal/guard/evaluator.go`, and add `TestGuardEvaluatePrecedenceAndUserLevel` in `internal/guard/guard_test.go`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR1`
    - Design: `internal/guard/evaluator.go`, `internal/resolve/resolve.go`, `internal/source/git.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestGuardEvaluatePrecedenceAndUserLevel$", "./internal/guard"]
        expect_output: "--- PASS: TestGuardEvaluatePrecedenceAndUserLevel"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "NFR1"]

- [x] 3. `gradusctl guard` CLI command and exit-code contract
  - [x] 3.1 Implement `newGuardCmd` in `internal/cli/guard.go`, wire it in `internal/cli/root.go`, emitting the dual-compatible JSON decision envelope (`decision`, `reason`, `matched_rules`, `anti_tamper`, `hookSpecificOutput`) with exit codes `0` (`allow`/`ask`), `1` (`deny`), `2` (technical error), and add end-to-end `TestGuardCommand` in `internal/cli/guard_test.go`
    - Requirements: `R4.AC1`, `R4.AC2`, `NFR3`
    - Design: `internal/cli/guard.go`, `internal/cli/root.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestGuardCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestGuardCommand"
        covers: ["R4.AC1", "R4.AC2", "NFR3"]
