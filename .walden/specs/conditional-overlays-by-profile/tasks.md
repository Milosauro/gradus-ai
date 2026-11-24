---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T16:42:52Z
last_modified: 2026-11-24T16:49:07Z
approved_fingerprint: sha256:e0cd48797837aeda4a12f6f77f4597d22ed66ed4be76b7f3c8c97a36227b36ae
source_design_approved_at: 2026-11-24T16:42:52Z
source_design_fingerprint: sha256:7bdb937e7f2d5f424b26f6039fab38b989c4e205054bb6e8f7f53f611513cf1d
---

# Implementation Plan

- [x] 1. Extend `Profile`, `ActivationRule`, and `OverlayConfig` schemas and parsers
  - [x] 1.1 Update `internal/schema/types.go`, `internal/schema/parse_config.go`, `internal/schema/validate_config.go`, and add `TestProfileAndActivationRuleParsing`
    - Requirements: `R1.AC1`, `R1.AC2`, `C1`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestProfileAndActivationRuleParsing$", "./internal/schema"]
        expect_output: "--- PASS: TestProfileAndActivationRuleParsing"
        covers: ["R1.AC1", "R1.AC2"]

- [x] 2. Implement profile-driven overlay activation and mandatory conflict enforcement
  - [x] 2.1 Add `internal/resolve/overlays.go`, wire into `internal/resolve/resolve.go` and `internal/resolve/types.go`, and add `TestConditionalOverlayActivation`
    - Requirements: `R2.AC1`, `R2.AC2`, `C1`, `C2`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestConditionalOverlayActivation$", "./internal/resolve"]
        expect_output: "--- PASS: TestConditionalOverlayActivation"
        covers: ["R2.AC1", "R2.AC2", "C1"]

- [x] 3. Update JSON schemas and add end-to-end CLI verification
  - [x] 3.1 Update `spec/v1/schema/*.json`, add `TestConditionalOverlaysCLI`, and verify static build
    - Requirements: `R3.AC1`, `NFR1`, `NFR2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "./spec/v1"]
        expect_output: "PASS"
        covers: ["R3.AC1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestConditionalOverlaysCLI$", "./internal/cli"]
        expect_output: "--- PASS: TestConditionalOverlaysCLI"
        covers: ["R3.AC1"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
