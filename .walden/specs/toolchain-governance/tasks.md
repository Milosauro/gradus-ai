---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T15:01:30Z
last_modified: 2026-11-23T15:04:32Z
approved_fingerprint: sha256:635a3509b636ff369699ca84216ca1dee3f8e53728e5956f5d02b0b7ae63ce99
source_design_approved_at: 2026-11-23T14:58:56Z
source_design_fingerprint: sha256:7a5b1d6b09c85ce251e643cec23b31c8d8bde607770500c7fa669a9ccee4b5cf
---

# Implementation Plan

- [x] 1. Add `toolchain` schema types, validation, and JSON Schema definitions
  - [x] 1.1 Extend `internal/schema/types.go`, `internal/schema/validate_config.go`, and `spec/v1/schema/*.json` with unit tests
    - Requirements: `R1.AC1`, `R1.AC2`, `C1`, `C2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestToolchainSchema$", "./internal/schema"]
        expect_output: "--- PASS: TestToolchainSchema"
        covers: ["R1.AC1", "R1.AC2"]
      - command: ["go", "test", "-v", "-count=1", "./spec/v1"]
        expect_output: "--- PASS: TestJSONSchemasCompileAndValidate"
        covers: ["C1"]

- [x] 2. Resolve toolchain declarations monotonically in `internal/resolve`
  - [x] 2.1 Implement `internal/resolve/toolchain.go`, extend `ResolvedView`, and wire into `resolve.Resolve` with unit tests
    - Requirements: `R2.AC1`, `R2.AC2`, `NFR1`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveToolchain$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveToolchain"
        covers: ["R2.AC1", "R2.AC2", "NFR1"]

- [x] 3. Implement local workspace toolchain checker and `gradusctl toolchain check` CLI
  - [x] 3.1 Implement `internal/toolchain/check.go` and `internal/cli/toolchain.go` (registered in `root.go`) with unit and CLI tests
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR2`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestToolchainCheck$", "./internal/toolchain"]
        expect_output: "--- PASS: TestToolchainCheck"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestToolchainCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestToolchainCommand"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR2"]
