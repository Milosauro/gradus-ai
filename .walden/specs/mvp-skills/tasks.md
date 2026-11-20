---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:11:21Z
last_modified: 2026-11-20T14:13:49Z
approved_fingerprint: sha256:c71f931fbe245d0f2317bfe7e08d11748471a82e53acf6bad2734605f4abe2af
source_design_approved_at: 2026-11-20T14:08:25Z
source_design_fingerprint: sha256:b68c51c85b9c390764fa432595d0ef58f9c3463de11109d1518a360e611597d9
---

# Implementation Plan

- [x] 1. Author and embed the six canonical MVP Agent Skills
  - [x] 1.1 Create `skills/gradus-{onboard,author,review,learn,reconcile,conflicts}/SKILL.md` with valid YAML frontmatter, `gradusctl` CLI workflows, and §16 safety guardrails/prohibitions, remove `skills/gradus-placeholder/`, and add `TestEmbeddedSkills` in `skills/skills_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR2`
    - Design: `skills/gradus-*/SKILL.md`, `skills/embed.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestEmbeddedSkills$", "./skills"]
        expect_output: "--- PASS: TestEmbeddedSkills"
        covers: ["R1.AC1", "R1.AC2", "R2.AC1", "R2.AC2", "R2.AC3", "NFR1", "NFR2"]

- [x] 2. Skill-TDD regression suite (`evals/skills/`)
  - [x] 2.1 Create `evals/skills/README.md` and the seven YAML regression scenarios (`gradus-{onboard,author,review,learn,reconcile,conflicts}.yaml` + `prompt-injection.yaml`) documenting baseline failures, verbatim rationalizations, required behaviors, and forbidden behaviors, and add `TestSkillEvalScenarios` in `skills/skills_test.go`
    - Requirements: `R3.AC1`, `R3.AC2`, `NFR2`
    - Design: `evals/skills/`, `skills/skills_test.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestSkillEvalScenarios$", "./skills"]
        expect_output: "--- PASS: TestSkillEvalScenarios"
        covers: ["R3.AC1", "R3.AC2", "NFR2"]

- [x] 3. End-to-end compiler extraction verification
  - [x] 3.1 Add `TestEmbeddedSkillsAndEvals` in `skills/skills_test.go` verifying `compile.ApplyCompile` extracts all six canonical skills cleanly into `.claude/skills/` and `.agents/skills/` with zero `gradus-placeholder` artifacts
    - Requirements: `R1.AC1`, `R3.AC2`, `NFR1`
    - Design: `skills/skills_test.go`, `internal/compile/compile.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestEmbeddedSkillsAndEvals$", "./skills"]
        expect_output: "--- PASS: TestEmbeddedSkillsAndEvals"
        covers: ["R1.AC1", "R3.AC2", "NFR1"]
