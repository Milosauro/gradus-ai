---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:32:53Z
last_modified: 2026-11-21T17:36:17Z
approved_fingerprint: sha256:b4256ed971224f2aa4f1ab825e5fc400278acd8d2703709b4422f5f26429122f
source_design_approved_at: 2026-11-21T17:31:56Z
source_design_fingerprint: sha256:2b5779e8d90b906f2b08b979fab0d772875473f576d43b0eff792879b5149814
---

# Implementation Plan

- [x] 1. Reusable Composite GitHub Action
  - [x] 1.1 Create `.github/actions/gradus/action.yml` with SHA-pinned `uses:` references, `verify` + `check` execution, `$GITHUB_STEP_SUMMARY` reporting, and evidence upload, plus automated YAML/SHA verification test
    - Requirements: `R1.AC1`, `R1.AC2`, `NFR1`
    - Design: Architecture §1 (Reusable Composite GitHub Action)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestGitHubActionSpec$", "./internal/cli"]
        expect_output: "--- PASS: TestGitHubActionSpec"
        covers: ["R1.AC1", "R1.AC2", "NFR1"]

- [x] 2. Team Agent Profiles (`--agent-profile` / `GRADUS_AGENT_PROFILE`)
  - [x] 2.1 Implement `Options.AgentProfile` in `internal/resolve`, `LoadLocalGuardRulesWithProfile` in `internal/guard`, and `--agent-profile` CLI flags on `resolve`, `compile`, and `guard`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR2`, `C1`
    - Design: Architecture §2 (Team Agent Profiles)
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAgentProfileWorkflow$", "./internal/cli"]
        expect_output: "--- PASS: TestAgentProfileWorkflow"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR2"]
