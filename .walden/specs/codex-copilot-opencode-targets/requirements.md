---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T15:52:48Z
last_modified: 2026-11-23T15:52:48Z
approved_fingerprint: sha256:f59362cce6b34db26330285cbe9d6b98d960cf6022ee2da3115618e1faf6be1e
---

# Requirements Document

## Introduction

Phase 2 ("Business unit") expands Gradus compilation beyond the MVP targets (`claude-code`, `antigravity`, `agents-md`) to support three additional coding agents in the order defined by `docs/design/gradus-design.md` §20 and §24.1: **`codex`**, **`copilot`** (GitHub Copilot), and **`opencode`**, together with a maintained agent × feature compatibility matrix.

## Requirements

### R1 Codex Target Compiler (`codex`)

**User Story:** As a developer using OpenAI Codex, I want `targets: [codex]` in `gradus.yaml` to compile our resolved pyramid into Codex's native instruction and skill locations, so that Codex enforces the same governed rules as Claude Code and Antigravity.

#### Acceptance Criteria

1. `R1.AC1` WHEN `codex` is configured in `targets`, `compile.PlanCompile` SHALL generate an `AGENTS.md` managed section (excluding `user`-level rules and preserving surrounding human-written content) and extract embedded Gradus skills into `.agents/skills/gradus-*/SKILL.md`.
   - Acceptance check: Compiling a pyramid with `targets: [codex]` writes `AGENTS.md` with managed markers and `.agents/skills/gradus-*/SKILL.md`, excluding `user`-level rules from `AGENTS.md`.

### R2 GitHub Copilot Target Compiler (`copilot`)

**User Story:** As a team using GitHub Copilot, I want `targets: [copilot]` in `gradus.yaml` to compile global and path-scoped rules into Copilot's native `.github/copilot-instructions.md` and `.github/instructions/gradus-*.instructions.md` files, so that Copilot receives scoped instructions automatically.

#### Acceptance Criteria

1. `R2.AC1` WHEN `copilot` is configured in `targets`, `compile.PlanCompile` SHALL generate a managed section inside `.github/copilot-instructions.md` containing global non-user rules (preserving surrounding human-written prose), emit scoped project rules under `.github/instructions/gradus-<id>.instructions.md` with `applyTo` frontmatter, and extract embedded Gradus skills into `.github/skills/gradus-*/SKILL.md`.
   - Acceptance check: Compiling a pyramid with global and scoped project rules targeting `copilot` writes `.github/copilot-instructions.md`, `.github/instructions/gradus-<id>.instructions.md` with `applyTo:`, and `.github/skills/gradus-*/SKILL.md`.
2. `R2.AC2` WHEN a previously compiled scoped rule is removed or waived and `compile.ApplyCompile` runs for `copilot`, the compiler SHALL prune stale `.github/instructions/gradus-*.instructions.md` files while leaving human-authored instruction files untouched.
   - Acceptance check: Removing a scoped rule and re-running `ApplyCompile` removes its stale `gradus-*.instructions.md` file and preserves custom `team.instructions.md`.

### R3 OpenCode Target Compiler (`opencode`)

**User Story:** As a developer using OpenCode, I want `targets: [opencode]` in `gradus.yaml` to compile our resolved pyramid into `AGENTS.md` and `.opencode/skills/gradus-*/SKILL.md`, so that OpenCode receives both instructions and Gradus skills.

#### Acceptance Criteria

1. `R3.AC1` WHEN `opencode` is configured in `targets`, `compile.PlanCompile` SHALL generate an `AGENTS.md` managed section (excluding `user`-level rules and preserving surrounding human-written prose) and extract embedded Gradus skills into `.opencode/skills/gradus-*/SKILL.md`.
   - Acceptance check: Compiling a pyramid with `targets: [opencode]` writes `AGENTS.md` and `.opencode/skills/gradus-*/SKILL.md`.

### R4 Determinism, Idempotency, Drift Detection, and Compatibility Matrix

**User Story:** As a maintainer, I want all new targets to be zero-diff idempotent, integrated with `gradusctl verify` drift detection, and documented in an agent × feature compatibility matrix, so that teams know the exact capabilities of each target.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradusctl compile` runs twice consecutively for `codex`, `copilot`, and `opencode`, the second run SHALL report zero changed files (`E1`).
   - Acceptance check: Consecutive `gradusctl compile` runs produce `Changed: 0` on the second run.
2. `R4.AC2` IF a compiled target file for `codex`, `copilot`, or `opencode` is modified manually, THEN `gradusctl verify` SHALL report drift and exit with code `1`.
   - Acceptance check: Modifying `.github/copilot-instructions.md` causes `gradusctl verify` to exit `1` with a drift finding.
3. `R4.AC3` IF an unsupported target name is configured in `gradus.yaml`, THEN `compile.PlanCompile` SHALL return an error causing `gradusctl compile` to exit with code `2`.
   - Acceptance check: Configuring `targets: [unknown-agent]` exits with code `2`.
4. `R4.AC4` The repository SHALL maintain `docs/compatibility-matrix.md` documenting feature support across all six official targets (`claude-code`, `antigravity`, `agents-md`, `codex`, `copilot`, `opencode`).
   - Acceptance check: `docs/compatibility-matrix.md` exists and lists all six targets with their compiled artifacts and supported enforcement levels.

## Non-Functional Requirements

- `NFR1` **Idempotency & Determinism (`E1`):** Output ordering and bytes MUST be identical across repeated runs and platforms (`CGO_ENABLED=0`).
- `NFR2` **Backward Compatibility:** Existing `claude-code`, `antigravity`, and `agents-md` targets MUST continue to produce byte-identical outputs.

## Constraints And Dependencies

- `C1` Must reuse `internal/compile` managed-section helpers (`PlanAgentsMD`, `RenderManagedBlock`, `ApplyManagedSection`, `PlanEmbeddedSkills`, `WriteIfChanged`) without new third-party dependencies.
- `C2` Must integrate seamlessly with `gradusctl verify`, `gradusctl status`, `gradusctl doctor`, and `gradusctl reconcile` via `compile.PlanCompile`.

## Out Of Scope

- Native runtime `PreToolUse` hook dispatchers for Codex, Copilot, or OpenCode (only instruction + skill compilation is in scope for `v0.3.0`; hooks remain documented as not supported / manual in `docs/compatibility-matrix.md`).
- Chat ZIP packaging or MCP server (`gradusctl serve --mcp`) — Phase 3.
