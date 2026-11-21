---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T15:56:33Z
last_modified: 2026-11-21T16:08:41Z
approved_fingerprint: sha256:6fb9642212e57858cae9c58eaeb99f2fc8a04eb4d9283a38f8ebb5668e48d795
source_design_approved_at: 2026-11-21T15:50:39Z
source_design_fingerprint: sha256:723605b7f015b07b41f1639062f4e5f0d23c24fcb931a8b6b4ba5b8382bc5f85
---

# Implementation Plan

- [x] 1. Schema and JSON Schema Extensions
  - [x] 1.1 Add `ContextBudget` (`max_rules`, `max_chars`) to `LevelConfig`, `Expires` to `Rule`, validate `YYYY-MM-DD` dates and charter severities, and sync `spec/v1/schema/` JSON Schemas
    - Requirements: `R1.AC2`, `R3.AC3`, `R4.AC3`, `NFR1`, `NFR3`, `C1`
    - Design: Architecture §1 (Schema Layer)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCharterAndLifecycleSchemaValidation$", "./internal/schema"]
        expect_output: "--- PASS: TestCharterAndLifecycleSchemaValidation"
        covers: ["R1.AC2", "R3.AC3", "R4.AC3", "NFR1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestJSONSchemasCompileAndValidate$", "./spec/v1"]
        expect_output: "--- PASS: TestJSONSchemasCompileAndValidate"
        covers: ["C1"]

- [x] 2. Resolver Charter, Lifecycle, and Context Budget Enforcement
  - [x] 2.1 Implement charter `allowed_severities` checks, `LevelSummary.Charter` exposure, and `context_budget` (`max_rules`, `max_chars`) enforcement in `internal/resolve`
    - Requirements: `R1.AC1`, `R1.AC2`, `R2.AC1`, `R4.AC1`, `R4.AC2`, `R4.AC3`, `NFR1`, `C2`
    - Design: Architecture §2 (Resolution & Policy Enforcement)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveCharterAndContextBudget$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveCharterAndContextBudget"
        covers: ["R1.AC1", "R1.AC2", "R2.AC1", "R4.AC1", "R4.AC2", "R4.AC3"]
  - [x] 2.2 Implement `draft`/`retired` partitioning into `InactiveRules`, `deprecated` & overdue `review_by` warnings, and expired `expires` conflicts with injectable clock `Options.Now`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR2`
    - Design: Architecture §2 (Resolution & Policy Enforcement)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveRuleLifecycle$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveRuleLifecycle"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "NFR2"]

- [x] 3. CLI Integration and Embedded Skill Updates
  - [x] 3.1 Expose `Warnings` in `gradusctl verify`, expose level `Charter` in `gradusctl explain`, and verify end-to-end exit codes across `resolve`, `verify`, `compile`, and `guard`
    - Requirements: `R1.AC1`, `R2.AC1`, `R3.AC1`, `R3.AC2`, `R3.AC3`, `R4.AC1`, `R4.AC2`, `NFR3`, `C2`
    - Design: Architecture §3 (CLI & Skills)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCharterLifecycleAndBudgetCLI$", "./internal/cli"]
        expect_output: "--- PASS: TestCharterLifecycleAndBudgetCLI"
        covers: ["R1.AC1", "R2.AC1", "R3.AC1", "R3.AC2", "R3.AC3", "R4.AC1", "R4.AC2"]
  - [x] 3.2 Update `skills/gradus-author/SKILL.md` and `skills/gradus-review/SKILL.md` with charter-based placement and review guidance and verify embedded skills
    - Requirements: `R2.AC2`, `NFR3`
    - Design: Architecture §3 (CLI & Skills)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCharterSkillGuidance$", "./skills"]
        expect_output: "--- PASS: TestCharterSkillGuidance"
        covers: ["R2.AC2"]
