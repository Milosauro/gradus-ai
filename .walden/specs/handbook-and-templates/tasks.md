---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:44:32Z
last_modified: 2026-11-21T17:47:54Z
approved_fingerprint: sha256:c3b7a53a57da6b2caee2eee02b255078418463199563f9b4290fc80a226726af
source_design_approved_at: 2026-11-21T17:43:32Z
source_design_fingerprint: sha256:ad2ebcd789f60fa649f2d671e1419a79659c02f9b4822708eb90a0dda484f597
---

# Implementation Plan

- [x] 1. Deterministic Onboarding Handbook (`gradusctl handbook`)
  - [x] 1.1 Implement `compile.RenderHandbook` in `internal/compile/handbook.go` and `newHandbookCmd` in `internal/cli/handbook.go` (registered in `root.go`) supporting stdout, `--output`, and `--json`
    - Requirements: `R1.AC1`, `R1.AC2`, `NFR1`, `NFR2`
    - Design: Architecture §1 (Deterministic Handbook Renderer)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestHandbookCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestHandbookCommand"
        covers: ["R1.AC1", "R1.AC2", "NFR2"]

- [x] 2. Non-Prescriptive Starter Templates (`examples/`)
  - [x] 2.1 Add `examples/personal/`, `examples/team/`, and `examples/regulated-fintech/` starter pyramids with charters, rules, and examples, verified by `resolve` and `gradusctl test`
    - Requirements: `R2.AC1`, `NFR1`, `C1`
    - Design: Architecture §2 (Non-Prescriptive Starter Templates)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestExampleTemplates$", "./internal/cli"]
        expect_output: "--- PASS: TestExampleTemplates"
        covers: ["R2.AC1", "C1"]
