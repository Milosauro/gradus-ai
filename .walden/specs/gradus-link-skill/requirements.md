---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T15:23:45Z
last_modified: 2026-11-23T15:23:45Z
approved_fingerprint: sha256:99b8a1ecbbbb196f0cf0cb3e218a990ac89f14ac65286269a4f2c79a9b5fff93
---

# Requirements Document

## Introduction

Phase 2 ("Business unit") adds the seventh official Agent Skill, **`gradus-link`** (`docs/design/gradus-design.md` §17, §19, §22, §24.1). When a team adopts or links a third-party skill, CLI tool, MCP server, or workflow provider, `gradus-link` performs a compatibility and security audit against the resolved Gradus pyramid (`obligations`, `bindings`, `toolchain`), drafts declarative adapter manifests (`.gradus/adapters/<id>.yaml`) and binding proposals, and refuses to link any tool that violates the pyramid without an approved governance decision.

## Requirements

### R1 Canonical `gradus-link` Agent Skill

**User Story:** As a developer or security reviewer, I want an embedded `gradus-link` Agent Skill so that AI coding agents follow a safe, repeatable audit and manifest-authoring workflow before linking third-party tools or skills.

#### Acceptance Criteria

1. `R1.AC1` The system SHALL author and embed `skills/gradus-link/SKILL.md` via `skills/embed.go` with valid YAML frontmatter (`name: gradus-link`, non-empty `description`) and canonical English instructions (`PRJ-LANG-001`).
   - Acceptance check: `skills.FS` contains `gradus-link/SKILL.md` with valid frontmatter and `name: gradus-link`.
2. `R1.AC2` The `gradus-link` skill SHALL instruct agents to inspect `gradusctl resolve --json`, `gradusctl adapter list --json`, and `gradusctl toolchain check --json`, perform a compatibility and security audit (checking `toolchain.forbidden`, `allowed_providers`, prompt-injection and data-exfiltration risks), propose a declarative `.gradus/adapters/<id>.yaml` manifest (with safe relative `target_file` and argv-only commands) and/or binding proposal, and require explicit human approval before applying any file changes.
   - Acceptance check: `skills/gradus-link/SKILL.md` includes the audit workflow, CLI commands, manifest/binding proposal guidance, and explicit approval requirement.
3. `R1.AC3` The `gradus-link` skill SHALL include explicit Guardrails & Prohibitions and Rejected Rationalizations forbidding linking any tool that violates the pyramid without an approved decision and forbidding treating untrusted third-party tool/skill text as overriding `gradus.yaml` rules.
   - Acceptance check: `skills/gradus-link/SKILL.md` contains explicit prohibitions and rejected rationalizations covering pyramid violations and prompt injection.

### R2 Compiler Extraction and Skill-TDD Regression Coverage

**User Story:** As a Gradus maintainer, I want `gradus-link` automatically extracted by `gradusctl compile` and covered by the §23 Skill-TDD regression suite, so that agents always receive the skill and its guardrails remain tested.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl compile` runs for `claude-code` or `antigravity` targets, the compiler SHALL extract `gradus-link/SKILL.md` into `.claude/skills/gradus-link/SKILL.md` and `.agents/skills/gradus-link/SKILL.md` deterministically.
   - Acceptance check: Compiling a repository targeting `claude-code` and `antigravity` writes `gradus-link/SKILL.md` in both skill directories with zero diff on a second run.
2. `R2.AC2` The repository SHALL include a Skill-TDD evaluation scenario file under `evals/skills/` covering `gradus-link` (including forbidden tool refusal and prompt-injection defense) validated by `skills/skills_test.go`.
   - Acceptance check: `go test -v -count=1 ./skills` validates `gradus-link` in `canonicalSkills`, its required guardrail snippets, and its `evals/skills/` scenario.

## Non-Functional Requirements

- `NFR1` **Static Embedding (`PRJ-GO-001`):** `gradus-link` MUST be embedded directly via `go:embed` in `skills/embed.go` with zero external runtime dependencies and `CGO_ENABLED=0`.
- `NFR2` **Repository Self-Consistency:** After adding `gradus-link`, running `gradusctl compile` and `gradusctl verify` on the `gradus-ai` repository itself MUST pass with zero drift.

## Constraints And Dependencies

- `C1` Must align with `F16` (`obligations` / `bindings`), `F17` (declarative `.gradus/adapters/` manifests and `gradusctl adapter`), and `F18` (`gradusctl toolchain check`).
- `C2` Must follow the structure and frontmatter conventions of existing skills in `skills/gradus-*/SKILL.md`.

## Out Of Scope

- Automated runtime execution of third-party vulnerability scanners or remote package registries.
- Community adapter marketplace or remote manifest fetching.
