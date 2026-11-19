---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-19T19:47:17Z
last_modified: 2026-11-19T19:53:56Z
approved_fingerprint: sha256:f9d2209fe65566c12d05064e54cb9c10450865f6e4617ca583c5ab5f2f54e05e
source_design_approved_at: 2026-11-19T17:43:32Z
source_design_fingerprint: sha256:d10e8cd72d2da81b700126d042932fb121d09a194a2b8204c406c320f19b281e
---

# Implementation Plan

- [x] 1. Author `gradus/v1` open specification and JSON Schemas
  - [x] 1.1 Create `spec/v1/SPEC.md` and the 4 config JSON Schemas (`gradus.schema.json`, `level.schema.json`, `overlay.schema.json`, `rule-frontmatter.schema.json`) with `spec/v1/schema_test.go` (`TestConfigSchemasValid`)
    - Requirements: `R1.AC1`, `R1.AC2`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestConfigSchemasValid$", "./spec/v1"]
        expect_output: "--- PASS: TestConfigSchemasValid"
        covers: ["R1.AC1", "R1.AC2"]
  - [x] 1.2 Create the 5 runtime/state JSON Schemas (`lock.schema.json`, `resolved-view.schema.json`, `evidence.schema.json`, `lesson.schema.json`, `exception.schema.json`) and `TestStateSchemasValid` in `spec/v1/schema_test.go`
    - Requirements: `R1.AC2`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestStateSchemasValid$", "./spec/v1"]
        expect_output: "--- PASS: TestStateSchemasValid"
        covers: ["R1.AC2"]

- [x] 2. Implement `internal/schema` parser, validator, and local loader
  - [x] 2.1 Create `internal/schema/types.go` (`PyramidConfig`, `LevelConfig`, `OverlayConfig`, `Rule`, `OverridableSpec.UnmarshalYAML`, `ParseResult[T]`) and `internal/schema/unknown.go` (`CollectUnknownKeys` for unknown and reserved keys `obligations|bindings|toolchain|models|adapters`)
    - Requirements: `R2.AC3`, `R3.AC4`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestOverridableAndUnknownKeys$", "./internal/schema"]
        expect_output: "--- PASS: TestOverridableAndUnknownKeys"
        covers: ["R2.AC3", "R3.AC4"]
  - [x] 2.2 Create `internal/schema/parse_config.go` and `internal/schema/validate_config.go` (`ParsePyramidConfig`, `ParseLevelConfig`, `ParseOverlayConfig` validating `schema: gradus/v1` and ID regex `^[a-z][a-z0-9-]*$`)
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC4`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestConfigParseAndValidate$", "./internal/schema"]
        expect_output: "--- PASS: TestConfigParseAndValidate"
        covers: ["R2.AC1", "R2.AC2", "R2.AC4"]
  - [x] 2.3 Create `internal/schema/parse_rule.go` and `internal/schema/validate_rule.go` (`ParseRuleFile(path, derivedLevel, source)` splitting frontmatter `---`, parsing `Why`/`Rationalizations`/`Compliant`/`Non-compliant`, validating `checker`/`guard`, and enforcing ADR 0004 level derivation)
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `R3.AC4`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestRuleParseAndLevelDerivation$", "./internal/schema"]
        expect_output: "--- PASS: TestRuleParseAndLevelDerivation"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "R3.AC4"]
  - [x] 2.4 Create `internal/schema/loader.go` (`LoadLocalPyramid(rootDir string)` skipping `~/` when `CI` is set and handling cross-platform paths)
    - Requirements: `R2.AC1`, `R3.AC3`, `NFR1`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestLoadLocalPyramid$", "./internal/schema"]
        expect_output: "--- PASS: TestLoadLocalPyramid"
        covers: ["NFR1"]

- [x] 3. Implement `gradusctl init`, `gradusctl explain`, and exit code `2` error mapping
  - [x] 3.1 Update `internal/cli/root.go` with `ExitErrorWithCode` and `ExecuteCode() int` mapping errors to exit code `2`
    - Requirements: `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExitErrorWithCode$", "./internal/cli"]
        expect_output: "--- PASS: TestExitErrorWithCode"
        covers: ["NFR3"]
  - [x] 3.2 Create `internal/cli/init.go` implementing `gradusctl init [--force] [--json]` (writes `gradus.yaml`, `.gradus/levels/project/level.yaml`, `rules/PRJ-GEN-001.md`, appends `.gitignore`, emits JSON `created_files`/`updated_files`)
    - Requirements: `R4.AC1`, `R4.AC2`, `R4.AC3`, `NFR1`, `NFR2`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestInitCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestInitCommand"
        covers: ["R4.AC1", "R4.AC2", "R4.AC3", "NFR3"]
  - [x] 3.3 Create `internal/cli/explain.go` implementing `gradusctl explain <id> [--json]` (outputs `rule`, `overridden_by`, `notes`) and verify `CGO_ENABLED=0` static build
    - Requirements: `R5.AC1`, `R5.AC2`, `R5.AC3`, `NFR1`, `NFR2`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExplainCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestExplainCommand"
        covers: ["R5.AC1", "R5.AC2", "R5.AC3", "NFR3"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -trimpath -o /dev/null ./cmd/gradusctl && echo BUILD_OK"]
        expect_output: "BUILD_OK"
        covers: ["NFR1"]
