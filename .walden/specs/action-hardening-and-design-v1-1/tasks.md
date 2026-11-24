---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T15:16:57Z
last_modified: 2026-11-24T15:37:33Z
approved_fingerprint: sha256:30aa8fd79822c004d7ae2942ff7e7e99d478373414c36396c6d606172cb86b81
source_design_approved_at: 2026-11-24T15:16:56Z
source_design_fingerprint: sha256:dcc9791030e28303faaaae9b5eec24abf2dc9cb712387bf18c736f5a3c833662
---

# Implementation Plan

- [x] 1. Harden anti-tamper, redaction, and early exception duration validation
  - [x] 1.1 Update `internal/guard/antitamper.go`, `internal/state/redact.go`, and `internal/cli/exception.go` with targeted tests
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `R1.AC4`, `C1`, `C2`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAntiTamperHardeningV032$", "./internal/guard"]
        expect_output: "--- PASS: TestAntiTamperHardeningV032"
        covers: ["R1.AC1", "R1.AC2"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestRedactionGapsV032$", "./internal/state"]
        expect_output: "--- PASS: TestRedactionGapsV032"
        covers: ["R1.AC3"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExceptionRequestMaxDurationV032$", "./internal/cli"]
        expect_output: "--- PASS: TestExceptionRequestMaxDurationV032"
        covers: ["R1.AC4", "C1"]

- [x] 2. Expand deterministic check engine edge-case coverage
  - [x] 2.1 Add edge-case tests in `internal/check/engine_test.go` for binary files, large files, complex globs, and failing/missing external checkers
    - Requirements: `R2.AC1`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCheckEngineEdgeCasesV032$", "./internal/check"]
        expect_output: "--- PASS: TestCheckEngineEdgeCasesV032"
        covers: ["R2.AC1"]

- [x] 3. Fix non-Go GitHub Action, upgrade YAML/Action pins, and promote `PRJ-CI-001` to check
  - [x] 3.1 Update `.github/actions/gradus/action.yml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, migrate `gopkg.in/yaml.v3` to `go.yaml.in/yaml/v3`, and promote `PRJ-CI-001.md` to `check`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR1`, `C1`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "mod", "tidy", "-diff"]
        expect_exit: 0
        covers: ["R3.AC3"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestPRJCI001CheckAndActionV032$", "./internal/cli"]
        expect_output: "--- PASS: TestPRJCI001CheckAndActionV032"
        covers: ["R3.AC1", "R3.AC2"]

- [x] 4. Publish Design 1.1, align `SPEC.md`, add Phase 2 plan, and update `README.md` & `CHANGELOG.md`
  - [x] 4.1 Update `docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/plan/phase-2-plan.md`, `README.md`, and `CHANGELOG.md`, then recompile and verify
    - Requirements: `R4.AC1`, `R4.AC2`, `NFR2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestDesign11AndSpecAlignmentV032$", "./internal/cli"]
        expect_output: "--- PASS: TestDesign11AndSpecAlignmentV032"
        covers: ["R4.AC1", "R4.AC2", "NFR2"]
