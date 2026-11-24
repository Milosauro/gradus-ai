---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T20:04:37Z
last_modified: 2026-11-24T20:07:57Z
approved_fingerprint: sha256:ded2dd89ec8debc3747abb479dbe6564c87c808a92d5dfb6d3c538cc16d8377f
source_design_approved_at: 2026-11-24T20:04:37Z
source_design_fingerprint: sha256:20992e038c42d504432362d25a75411c8fc24565d672027a1fed92ec67e0e48c
---

# Implementation Plan

- [x] 1. Propagate `controls` into `ResolvedRule` and implement `internal/audit`
  - [x] 1.1 Update `internal/resolve/types.go`, `internal/resolve/resolve.go`, `internal/resolve/invariant.go`, and create `internal/audit/report.go` + `internal/audit/report_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `R2.AC1`, `C1`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAuditCoverageAndExport$", "./internal/audit"]
        expect_output: "--- PASS: TestAuditCoverageAndExport"
        covers: ["R1.AC1", "R1.AC2", "R2.AC1"]

- [x] 2. Add `gradusctl audit coverage` and `gradusctl audit export` CLI commands
  - [x] 2.1 Create `internal/cli/audit.go`, register in `internal/cli/root.go`, and add `TestAuditCLICommands` in `internal/cli/audit_test.go`
    - Requirements: `R2.AC2`, `NFR1`, `NFR2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAuditCLICommands$", "./internal/cli"]
        expect_output: "--- PASS: TestAuditCLICommands"
        covers: ["R2.AC2", "NFR2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
