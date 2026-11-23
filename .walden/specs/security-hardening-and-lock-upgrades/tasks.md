---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T17:09:39Z
last_modified: 2026-11-23T17:21:56Z
approved_fingerprint: sha256:6ab92b9350a8137033d2f2dc1bd0cc0c075cdbc204e7f0d6355745e33d05ef2f
source_design_approved_at: 2026-11-23T17:03:14Z
source_design_fingerprint: sha256:40c7912ba8ba8bafe66a5abf4def02b9003d2999d2a62f77e8d422bcd8e97f41
---

# Implementation Plan

- [x] 1. Harden anti-tamper guard for human-only commands, `--no-verify`, and protected paths
  - [x] 1.1 Extend `internal/guard/antitamper.go` and add tests in `internal/guard/guard_test.go`
    - Requirements: `R1.AC1`, `R3.AC1`, `R3.AC2`, `NFR2`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAntiTamperHardeningV031$", "./internal/guard"]
        expect_output: "--- PASS: TestAntiTamperHardeningV031"
        covers: ["R1.AC1", "R3.AC1", "R3.AC2", "NFR2"]

- [x] 2. Enforce authorized approvers, user-free `view_hash`, and controlled `--update` lock upgrades
  - [x] 2.1 Update `internal/cli/exception.go`, `internal/resolve/resolve.go`, `internal/cli/resolve.go`, and `internal/schema/unknown.go` with CLI tests
    - Requirements: `R1.AC2`, `R2.AC1`, `R2.AC2`, `R2.AC3`, `R4.AC1`, `C1`, `C2`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestHardeningApproversViewHashAndUpdate$", "./internal/cli"]
        expect_output: "--- PASS: TestHardeningApproversViewHashAndUpdate"
        covers: ["R1.AC2", "R2.AC1", "R2.AC2", "R2.AC3", "R4.AC1", "C1"]

- [x] 3. Fix supply-chain reproducibility, handbook numbering, dogfooding rule tests, and docs
  - [x] 3.1 Tidy `go.mod`, harden `.goreleaser.yaml` and `.github/actions/gradus/action.yml`, fix `internal/compile/handbook.go`, add `PRJ-GO-001.md` examples, and update `README.md` and `CHANGELOG.md`
    - Requirements: `R4.AC2`, `R4.AC3`, `R4.AC4`, `NFR1`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "mod", "tidy", "-diff"]
        expect_exit: 0
        covers: ["R4.AC2"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestHandbookNumberingAndSupplyChain$", "./internal/cli"]
        expect_output: "--- PASS: TestHandbookNumberingAndSupplyChain"
        covers: ["R4.AC2", "R4.AC3", "R4.AC4"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
