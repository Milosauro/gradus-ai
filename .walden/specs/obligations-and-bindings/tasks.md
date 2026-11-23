---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T14:00:28Z
last_modified: 2026-11-23T14:04:30Z
approved_fingerprint: sha256:ec45e440223b3692ffa9b14ae80fc0cc121107493ba2d08eec20192ec39cea11
source_design_approved_at: 2026-11-23T13:57:03Z
source_design_fingerprint: sha256:bd8cd863e1e028e5dc0a36da4eb671441506bd6e608e430e3cbdb03227623976
---

# Implementation Plan

- [x] 1. Add `obligations` and `bindings` schema types, validation, and JSON Schema definitions
  - [x] 1.1 Extend `internal/schema/types.go`, `internal/schema/validate_config.go`, and `spec/v1/schema/*.json` with unit tests
    - Requirements: `R1.AC1`, `R1.AC2`, `C1`, `C2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestObligationsAndBindingsSchema$", "./internal/schema"]
        expect_output: "--- PASS: TestObligationsAndBindingsSchema"
        covers: ["R1.AC1", "R1.AC2"]
      - command: ["go", "test", "-v", "-count=1", "./spec/v1"]
        expect_output: "--- PASS: TestJSONSchemasCompileAndValidate"
        covers: ["C1"]

- [x] 2. Resolve obligations and bindings and enforce restriction invariants
  - [x] 2.1 Implement `internal/resolve/obligations.go`, extend `ResolvedView`, and wire into `resolve.Resolve` with unit tests
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `R3.AC1`, `NFR1`, `NFR2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveObligationsAndBindings$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveObligationsAndBindings"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "R3.AC1", "NFR1", "NFR2"]

- [x] 3. Render obligations and bindings in `gradusctl handbook` and verify CLI integration
  - [x] 3.1 Update `internal/compile/handbook.go` and add CLI end-to-end tests in `internal/cli/obligations_test.go`
    - Requirements: `R2.AC2`, `R3.AC1`, `R3.AC2`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestObligationsCLIAndHandbook$", "./internal/cli"]
        expect_output: "--- PASS: TestObligationsCLIAndHandbook"
        covers: ["R2.AC2", "R3.AC1", "R3.AC2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["C2"]
