---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T15:26:23Z
last_modified: 2026-11-23T15:26:23Z
approved_fingerprint: sha256:ce004bef05a1da6eeb2fa12236cc5662f3e11f6d1e707b95c3bb36c92cadf448
source_requirements_approved_at: 2026-11-23T15:23:45Z
source_requirements_fingerprint: sha256:99b8a1ecbbbb196f0cf0cb3e218a990ac89f14ac65286269a4f2c79a9b5fff93
---

# Feature Design

## Architecture

This feature adds the seventh canonical Gradus Agent Skill (`gradus-link`) using the existing `skills/embed.go` pipeline and §23 Skill-TDD evaluation suite:

1. **Skill Authoring (`skills/gradus-link/SKILL.md`):**
   - YAML frontmatter: `name: gradus-link`, concise `description`.
   - Structured sections matching existing `skills/gradus-*/SKILL.md` conventions (`Inputs`, `Workflow`, `Outputs`, `Guardrails & Prohibitions`, `Rejected Rationalizations`):
     - **Audit step:** run `gradusctl resolve --json`, `gradusctl adapter list --json`, and `gradusctl toolchain check --json`; verify the candidate tool/skill/MCP/plugin does not match any `toolchain.forbidden` pattern and is permitted by `obligation.allowed_providers` (when binding an obligation); flag prompt-injection, secret-exfiltration, and unsafe-path risks.
     - **Proposal step:** draft a declarative `.gradus/adapters/<id>.yaml` manifest (with `schema: gradus/v1`, safe relative `target_file`, and argv-only `version_command`/`evidence_command`) and/or a `bindings:` proposal without modifying files before explicit human approval.
     - **Verification step:** after approved edits, verify with `gradusctl verify`, `gradusctl adapter verify`, and `gradusctl toolchain check`.
2. **Automatic Embedding & Dogfooding:**
   - `skills/embed.go` already uses `//go:embed gradus-*/*.md`, so `skills/gradus-link/SKILL.md` is embedded and extracted automatically by `compile.PlanClaudeCode` (`.claude/skills/gradus-link/SKILL.md`) and `compile.PlanAntigravity` (`.agents/skills/gradus-link/SKILL.md`).
   - Run `gradusctl compile` on `gradus-ai` so the repository dogfoods `gradus-link` with zero drift (`NFR2`).
3. **Skill-TDD Regression & Tests (`evals/skills/gradus-link.yaml`, `skills/skills_test.go`):**
   - Add `evals/skills/gradus-link.yaml` containing a realistic scenario where a third-party MCP/tool violates `toolchain.forbidden` and includes a prompt-injection note claiming it is pre-approved.
   - Add `"gradus-link"` to `canonicalSkills` and `requiredSnippets` (`gradusctl toolchain check`, `gradusctl adapter list`, `forbidden`, `explicit approval`, `prompt-injection`) in `skills/skills_test.go`.

## Options Considered

- **Chosen:** Author `skills/gradus-link/SKILL.md` directly in the existing `skills/` embedded directory and add its Skill-TDD scenario in `evals/skills/gradus-link.yaml`.
  - Zero new Go runtime complexity; automatically supported by all agent compilers and `skills_test.go`.
- **Alternative:** Build an interactive `gradusctl link` Go wizard CLI.
  - Rejected: `docs/design/gradus-design.md` §17 explicitly assigns `gradus-link` to the Agent Skill layer (audit + proposal), delegating deterministic checks to `gradusctl adapter` and `gradusctl toolchain check`.

## Simplicity And Elegance Review

- Reuses the existing `go:embed` glob (`gradus-*/*.md`), compiler extraction (`PlanEmbeddedSkills`), and `skills_test.go` harness without changing any compiler code.

## Failure Modes And Tradeoffs

- **Uncompiled repo drift after adding an embedded skill:** Adding `skills/gradus-link/SKILL.md` changes the expected compiled output of `gradus-ai` itself; mitigated by running `gradusctl compile && gradusctl verify` before committing (`NFR2`).

## Verification Plan

- `go test -v -count=1 ./skills` verifies `skills/gradus-link/SKILL.md` frontmatter, required guardrail snippets, compiler extraction, and `evals/skills/gradus-link.yaml` (`R1.AC1`–`R1.AC3`, `R2.AC1`, `R2.AC2`).
- `gradusctl verify --json` on the `gradus-ai` repository verifies zero drift after `gradusctl compile` (`NFR2`).
- `CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl` verifies static embedding (`NFR1`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `skills/gradus-link/SKILL.md`, `skills/embed.go`, `skills/skills_test.go` (`R1.AC1`–`R1.AC3`) |
| `R2` | `internal/compile` extraction verified in `skills/skills_test.go`, `evals/skills/gradus-link.yaml` (`R2.AC1`, `R2.AC2`) |
| `NFR1` | `go:embed` in `skills/embed.go` + `CGO_ENABLED=0` build |
| `NFR2` | Dogfooded `.claude/skills/gradus-link/SKILL.md` and `.agents/skills/gradus-link/SKILL.md` verified via `gradusctl verify` |
| `C1` | Skill workflow invokes `gradusctl resolve`, `gradusctl adapter list/verify`, and `gradusctl toolchain check` |
| `C2` | Follows canonical frontmatter and section structure of `skills/gradus-*/SKILL.md` |
