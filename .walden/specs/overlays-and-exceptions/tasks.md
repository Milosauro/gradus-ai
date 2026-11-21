---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T16:49:07Z
last_modified: 2026-11-21T16:53:41Z
approved_fingerprint: sha256:e569c51ab9d0563c979c49bb5ba0ef75d19a70a9f0037173b0a371ce5a27db74
source_design_approved_at: 2026-11-21T16:44:54Z
source_design_fingerprint: sha256:e055198d41f2b96c266157489cc2c5c6a5ddeed4ed1db304b5989df1f12fb562
---

# Implementation Plan

- [x] 1. Explicit Overlay Activation and Schema Alignment
  - [x] 1.1 Support `activation: explicit | always | disabled | profile` in schema validation, `spec/v1/schema/overlay.schema.json`, and `internal/resolve/resolve.go` with `allowed_overlays` checks
    - Requirements: `R1.AC1`, `R1.AC2`, `NFR2`, `C1`
    - Design: Architecture §1 (Explicit Overlay Activation)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveExplicitAndDisabledOverlays$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveExplicitAndDisabledOverlays"
        covers: ["R1.AC1", "R1.AC2"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestJSONSchemasCompileAndValidate$", "./spec/v1"]
        expect_output: "--- PASS: TestJSONSchemasCompileAndValidate"
        covers: ["C1"]

- [x] 2. Exception Policy and Break-Glass Resolution
  - [x] 2.1 Implement pending unapproved exception handling, `max_duration` enforcement, and `break-glass` policy checks (`reason`, `post_review_due`, `break_glass: false`, `fail-closed`) in `internal/resolve`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR2`
    - Design: Architecture §2 (Exception Policy & Break-Glass Resolution)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveExceptionAndBreakGlassPolicies$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveExceptionAndBreakGlassPolicies"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR1"]

- [x] 3. `gradusctl exception` State and CLI Workflow
  - [x] 3.1 Implement `internal/state/exceptions.go` and `internal/cli/exception.go` (`request`, `approve`, `break-glass`, `list`, `expire`) registered in `root.go` with end-to-end CLI tests
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `R3.AC4`, `NFR2`, `C1`
    - Design: Architecture §3 (CLI Exception State & Subcommands)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExceptionCommandWorkflow$", "./internal/cli"]
        expect_output: "--- PASS: TestExceptionCommandWorkflow"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "R3.AC4"]
