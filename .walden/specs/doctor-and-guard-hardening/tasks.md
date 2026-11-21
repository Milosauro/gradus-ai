---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:05:05Z
last_modified: 2026-11-21T17:10:37Z
approved_fingerprint: sha256:6e80da06892da2767aa787cd1e72d20fe567bb13a960bac13521def3c0f558c9
source_design_approved_at: 2026-11-21T17:03:40Z
source_design_fingerprint: sha256:15c5efbf4e16864c3b28736a44e2cf3dfec468234b0f0ab555b755c375ce0ebb
---

# Implementation Plan

- [x] 1. Configurable Anti-Tamper Policy (`ask` vs `deny`)
  - [x] 1.1 Add `AntiTamperConfig` (`mode: ask | deny`) to `PyramidConfig` and `spec/v1/schema/gradus.schema.json`, implement `EvaluateWithOptions` and `LoadAntiTamperMode` in `internal/guard`, and wire into `internal/cli/guard.go`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR2`, `C1`
    - Design: Architecture §1 (Anti-Tamper Policy)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestGuardAntiTamperDenyMode$", "./internal/cli"]
        expect_output: "--- PASS: TestGuardAntiTamperDenyMode"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestJSONSchemasCompileAndValidate$", "./spec/v1"]
        expect_output: "--- PASS: TestJSONSchemasCompileAndValidate"
        covers: ["C1"]

- [x] 2. `gradusctl doctor` Diagnostics Command
  - [x] 2.1 Implement `internal/cli/doctor.go` (registered in `root.go`) diagnosing `gradus.yaml`, `gradus.lock`, `PATH`, `.claude/settings.json` hook types and guard entries, and `.agents/hooks.json`
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR2`
    - Design: Architecture §2 (`gradusctl doctor`)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestDoctorCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestDoctorCommand"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3"]

- [x] 3. Evidence/Lessons Prune and Status User-Rule Visibility
  - [x] 3.1 Implement `gradusctl evidence prune`, `gradusctl lessons prune`, and `statusReport.UserRules` in `internal/state/` and `internal/cli/`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR2`
    - Design: Architecture §3 (Hygiene Commands & User-Rule Visibility)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestPruneAndStatusUserRules$", "./internal/cli"]
        expect_output: "--- PASS: TestPruneAndStatusUserRules"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3"]
