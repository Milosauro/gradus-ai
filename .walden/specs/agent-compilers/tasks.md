---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T09:31:27Z
last_modified: 2026-11-20T09:35:47Z
approved_fingerprint: sha256:b70e11a0e8151d00c440ac00f1b3b5c8aab12f623882a16ca22681ae16e7a9c2
source_design_approved_at: 2026-11-20T09:29:51Z
source_design_fingerprint: sha256:9b1b61ebc3c7284d75f9c7af6320a192f493ce6c7f72c2d945fee9b25380ba14
---

# Implementation Plan

- [x] 1. Implement shared managed-section, idempotency, and `AGENTS.md` compiler helpers
  - [x] 1.1 Create `internal/compile/managed.go` implementing `RenderManagedBlock`, `ApplyManagedSection`, `WriteIfChanged`, `PruneStaleGradusFiles`, `RenderPrecedenceHeader`, `RenderRuleMarkdown`, and `ExtractEmbeddedSkills` (skipping `gradus-placeholder`)
    - Requirements: `R1.AC1`, `NFR1`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestManagedSectionsAndIdempotency$", "./internal/compile"]
        expect_output: "--- PASS: TestManagedSectionsAndIdempotency"
        covers: ["R1.AC1", "NFR1", "NFR2"]
  - [x] 1.2 Create `internal/compile/agentsmd.go` implementing `PlanAgentsMD(existing string, view resolve.ResolvedView) []byte` rendering all active non-`user` rules in pyramid order (ADR 0002)
    - Requirements: `R1.AC2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAgentsMDCompiler$", "./internal/compile"]
        expect_output: "--- PASS: TestAgentsMDCompiler"
        covers: ["R1.AC2"]

- [x] 2. Implement Claude Code and Antigravity target compilers
  - [x] 2.1 Create `internal/compile/claudecode.go` implementing `PlanClaudeCode` (`CLAUDE.md` managed section, `.claude/rules/gradus-*.md` with `paths` frontmatter, `CLAUDE.local.md` for `user` rules, non-destructive `.claude/settings.json` hook/permission merge, and `.claude/skills/` extraction)
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestClaudeCodeCompiler$", "./internal/compile"]
        expect_output: "--- PASS: TestClaudeCodeCompiler"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3"]
  - [x] 2.2 Create `internal/compile/antigravity.go` implementing `PlanAntigravity` (`AGENTS.md`, `.agents/rules/gradus-*.md` and `gradus-user-*.md` with `trigger: always_on`/`glob` frontmatter and automatic `< 11,500`-char splitting strictly below `12,000` chars, static `.agents/hooks.json` merge, and `.agents/skills/` extraction)
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAntigravityCompilerAndSplit$", "./internal/compile"]
        expect_output: "--- PASS: TestAntigravityCompilerAndSplit"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3"]

- [x] 3. Implement compilation orchestrator and `gradusctl compile` CLI command
  - [x] 3.1 Create `internal/compile/compile.go` implementing `PlanCompile` and `ApplyCompile` (writes only changed files via `WriteIfChanged`, prunes stale `gradus-*.md` files in `.claude/rules/` and `.agents/rules/`, returns deterministic `CompileResult`)
    - Requirements: `R4.AC1`, `NFR1`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCompilePlanAndPrune$", "./internal/compile"]
        expect_output: "--- PASS: TestCompilePlanAndPrune"
        covers: ["R4.AC1", "NFR1", "NFR2"]
  - [x] 3.2 Create `internal/cli/compile.go` implementing `gradusctl compile [--dir] [--json]` wired into `internal/cli/root.go` (exit `0` idempotent, `1` on blocking conflicts without modifying targets, `2` on config/lock/JSON errors) and verify `CGO_ENABLED=0` static build
    - Requirements: `R4.AC1`, `R4.AC2`, `R4.AC3`, `NFR3`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestCompileCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestCompileCommand"
        covers: ["R4.AC1", "R4.AC2", "R4.AC3", "NFR3"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -trimpath -o /dev/null ./cmd/gradusctl && echo BUILD_OK"]
        expect_output: "BUILD_OK"
        covers: ["NFR2"]
