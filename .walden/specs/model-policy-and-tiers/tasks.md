---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T08:35:09Z
last_modified: 2026-11-25T08:40:22Z
approved_fingerprint: sha256:14b1ff6bfac5f21345061e79f663838ce6b72318c16ccb2e4c90aed7dfd3fe06
source_design_approved_at: 2026-11-25T08:35:09Z
source_design_fingerprint: sha256:9d9c350cc5b07c82636c96c2dfb62e4550ffce2513511683a27a8005da33a210
---

# Implementation Plan

- [x] 1. Add `models:` schema and monotonic model policy resolution
  - [x] 1.1 Update `internal/schema/{types,unknown,parse_config,validate_config,types_test}.go`, create `internal/resolve/models.go` + `models_test.go`, wire into `resolve.go` and `types.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `C1`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestModelPolicySchemaAndResolution$", "./internal/resolve"]
        expect_output: "--- PASS: TestModelPolicySchemaAndResolution"
        covers: ["R1.AC1", "R1.AC2"]

- [x] 2. Emit `model:` frontmatter in Claude Code compiled skills
  - [x] 2.1 Update `internal/compile/claudecode.go` and add `TestClaudeCodeSkillModelInjection` in `internal/compile/claudecode_test.go`
    - Requirements: `R2.AC1`, `C1`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestClaudeCodeSkillModelInjection$", "./internal/compile"]
        expect_output: "--- PASS: TestClaudeCodeSkillModelInjection"
        covers: ["R2.AC1"]

- [x] 3. Add evidence tier recording, `verify --check-model-tier`, and JSON schema updates
  - [x] 3.1 Update `internal/cli/evidence.go`, `internal/cli/verify.go`, `spec/v1/schema/{gradus,level,resolved-view}.schema.json`, and add `TestModelPolicyEvidenceAndVerifyCLI`
    - Requirements: `R3.AC1`, `R3.AC2`, `NFR1`, `NFR2`
    - Design: Architecture, Failure Modes And Tradeoffs, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "./spec/v1"]
        expect_output: "PASS"
        covers: ["R3.AC1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestModelPolicyEvidenceAndVerifyCLI$", "./internal/cli"]
        expect_output: "--- PASS: TestModelPolicyEvidenceAndVerifyCLI"
        covers: ["R3.AC1", "R3.AC2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
