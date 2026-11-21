---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:20:26Z
last_modified: 2026-11-21T17:23:05Z
approved_fingerprint: sha256:bd8064ce9a514c9130c1869966fd6e5c30a2a27b4ce428a60e858cff6bed80cf
source_design_approved_at: 2026-11-21T17:18:46Z
source_design_fingerprint: sha256:afcb24cdf884233421f52990e36008cbce5c151821d9acb65419c6357b76a75d
---

# Implementation Plan

- [x] 1. Declarative Rule Testing (`gradusctl test`)
  - [x] 1.1 Extract `RunChecksOnFiles` in `internal/check/engine.go`, implement `RunRuleTests` in `internal/check/ruletest.go`, and add `internal/cli/test.go` (registered in `root.go`) supporting `check` and `hook` rules and `--rule` filtering
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR2`, `C1`
    - Design: Architecture §1 (Rule Testing Engine)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestRuleTestCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestRuleTestCommand"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3"]

- [x] 2. Historical Impact Simulation (`gradusctl simulate`)
  - [x] 2.1 Implement `RunSimulation` in `internal/check/simulate.go` (using `os/exec` `git` and non-destructive temp dirs) and add `internal/cli/simulate.go` (registered in `root.go`) supporting `--commits`, `--since`, and `--rule`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR2`, `C1`
    - Design: Architecture §2 (Historical Impact Simulation)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestSimulateCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestSimulateCommand"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR1", "NFR2"]
