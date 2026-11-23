---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T16:04:01Z
last_modified: 2026-11-23T16:05:37Z
approved_fingerprint: sha256:407380c0a3b99f0be22ef53a1f06140fd0e94f570948ee62d28156cb498cb9e5
source_design_approved_at: 2026-11-23T16:02:49Z
source_design_fingerprint: sha256:d3a8639242e781345644f35eec98ba639927d800353ce002ed60720aa01f60ba
---

# Implementation Plan

- [x] 1. Implement `codex`, `copilot`, and `opencode` target compilers and dispatcher wiring
  - [x] 1.1 Create `internal/compile/codex.go`, `copilot.go`, `opencode.go`, wire `compile.go` with Copilot pruning, and add unit tests
    - Requirements: `R1.AC1`, `R2.AC1`, `R2.AC2`, `R3.AC1`, `R4.AC3`, `NFR1`, `NFR2`, `C1`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExtraCompileTargets$", "./internal/compile"]
        expect_output: "--- PASS: TestExtraCompileTargets"
        covers: ["R1.AC1", "R2.AC1", "R2.AC2", "R3.AC1", "R4.AC3", "NFR1", "NFR2", "C1"]

- [x] 2. Add compatibility matrix documentation and CLI idempotency / drift verification tests
  - [x] 2.1 Create `docs/compatibility-matrix.md` and add CLI end-to-end tests in `internal/cli/extra_targets_test.go`
    - Requirements: `R4.AC1`, `R4.AC2`, `R4.AC4`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExtraTargetsCLIAndMatrix$", "./internal/cli"]
        expect_output: "--- PASS: TestExtraTargetsCLIAndMatrix"
        covers: ["R4.AC1", "R4.AC2", "R4.AC4", "C2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
