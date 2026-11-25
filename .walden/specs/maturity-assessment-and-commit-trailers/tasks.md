---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:56:10Z
last_modified: 2026-11-25T09:59:55Z
approved_fingerprint: sha256:9ab1425bfe52bf4b812a87f43164dc486ad5b71600a5dde0e20698332f37e04e
source_design_approved_at: 2026-11-25T09:56:10Z
source_design_fingerprint: sha256:8beec994841efc7f4417777b9d80bc003a9b62f74928c834038b1c2b048ac707
---

# Implementation Plan

- [x] 1. Implement process maturity assessment engine and `gradus-assess` skill
  - [x] 1.1 Create `internal/assess/assess.go`, `internal/assess/assess_test.go`, `skills/gradus-assess/SKILL.md`, `evals/scenarios/skill-08-assess.yaml`, and update `skills/skills_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `NFR2`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestProcessMaturityAssessment$", "./internal/assess"]
        expect_output: "--- PASS: TestProcessMaturityAssessment"
        covers: ["R1.AC1"]
      - command: ["go", "test", "-v", "-count=1", "./skills"]
        expect_output: "PASS"
        covers: ["R1.AC2"]

- [x] 2. Implement commit attribution trailers and CLI commands (`assess`, `trailers`)
  - [x] 2.1 Create `internal/trailers/trailers.go`, `internal/cli/assess.go`, `internal/cli/trailers.go`, register in `internal/cli/root.go`, and add `TestAssessAndTrailersCLI`
    - Requirements: `R2.AC1`, `R2.AC2`, `NFR1`, `C1`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAssessAndTrailersCLI$", "./internal/cli"]
        expect_output: "--- PASS: TestAssessAndTrailersCLI"
        covers: ["R2.AC1", "R2.AC2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]

- [x] 3. Update Phase 3 closure docs (`README.md`, `CHANGELOG.md`, `phase-3-plan.md`) and recompile
  - [x] 3.1 Update `docs/plan/phase-3-plan.md`, `README.md`, `CHANGELOG.md`, rebuild `gradusctl`, recompile targets, and verify
    - Requirements: `R3.AC1`, `NFR2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "mod", "tidy", "-diff"]
        expect_exit: 0
        covers: ["R3.AC1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAssessAndTrailersCLI$", "./internal/cli"]
        expect_output: "--- PASS: TestAssessAndTrailersCLI"
        covers: ["R3.AC1"]
