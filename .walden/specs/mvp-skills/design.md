---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:08:25Z
last_modified: 2026-11-20T14:08:25Z
approved_fingerprint: sha256:b68c51c85b9c390764fa432595d0ef58f9c3463de11109d1518a360e611597d9
source_requirements_approved_at: 2026-11-20T14:05:03Z
source_requirements_fingerprint: sha256:aeab029fe780e698ea5f3fe3535ed21c4903b8a93e728dbd1225cb30816fb90d
---

# Feature Design

## Architecture

Feature **F7 (`mvp-skills`)** replaces the Wave 0 placeholder skill with the six canonical MVP Agent Skills embedded via `skills/embed.go` (`//go:embed gradus-*/*.md`), adds the skill-TDD regression suite under `evals/skills/`, and verifies embedding, frontmatter, safety guardrails, and compiler extraction via `skills/skills_test.go`.

### Component Overview

1. **Canonical Skills (`skills/gradus-*/SKILL.md`)**
   Each skill uses standard Agent Skills YAML frontmatter (`name`, `description`) and five concise, structured sections (`Inputs`, `Workflow`, `Outputs`, `Guardrails & Prohibitions`, `Rejected Rationalizations`):
   - `skills/gradus-onboard/SKILL.md`: inspects repo rules/hooks/CI/git history, produces context dossier + divergence report + draft pyramid with citations and confidence, never applies without explicit human approval, redacts secrets/PII, never sends data outside the active model.
   - `skills/gradus-author/SKILL.md`: drafts rules with valid `gradus/v1` frontmatter, rationale, rationalizations, and compliant/non-compliant examples; checks level charters and redirects out-of-charter rules; proposes PRs for `locked: true` levels and never writes directly to locked levels.
   - `skills/gradus-review/SKILL.md`: reviews staged changes, branch diff, or code against `gradusctl resolve --json`; cites rule ID, level, severity, provenance, and confidence; refuses to report low-confidence guesses as violations; records evidence via `gradusctl evidence write`; never modifies code unless asked.
   - `skills/gradus-learn/SKILL.md`: captures session corrections/repeated mistakes/preferences, records redacted lessons via `gradusctl lessons add`, proposes promotions with suggested level and enforcement, never promotes without explicit approval, never stores unredacted secrets/PII.
   - `skills/gradus-reconcile/SKILL.md`: inspects drift via `gradusctl reconcile --json`, explains each drift item and allowed options (`restore`, `move`, upstream proposal, exception), delegates execution strictly to `gradusctl reconcile --restore` or `--move`, never edits `locked: true` levels directly.
   - `skills/gradus-conflicts/SKILL.md`: reads candidate pairs from `gradusctl conflicts --candidates --json`, classifies each pair (`contradiction`, `duplicate`, `redundant`, `compatible`) with justification and confidence, presents results to human for `gradusctl conflicts --confirm` or `--dismiss`, never blocks or resolves autonomously.
   - Removes `skills/gradus-placeholder/SKILL.md` so `skills.FS` contains strictly the six canonical skills.

2. **Skill-TDD Regression Suite (`evals/skills/`)**
   - `evals/skills/README.md`: documents the §23 Skill-TDD methodology (baseline without skill → verbatim rationalizations → minimal skill fix → regression rubric).
   - Seven structured YAML scenario files under `evals/skills/`:
     - `gradus-onboard.yaml`
     - `gradus-author.yaml`
     - `gradus-review.yaml`
     - `gradus-learn.yaml`
     - `gradus-reconcile.yaml`
     - `gradus-conflicts.yaml`
     - `prompt-injection.yaml` (§22 adversarial injection resistance scenario)
   - Each scenario records `id`, `skill`, `title`, `prompt`, `baseline_failures`, `baseline_rationalizations`, `required_behaviors`, and `forbidden_behaviors`.

3. **Deterministic Go Validation (`skills/skills_test.go`)**
   - `TestEmbeddedSkillsAndEvals`:
     - Verifies `skills.FS` contains all 6 canonical `gradus-*/SKILL.md` files and zero `gradus-placeholder` entries.
     - Parses YAML frontmatter of each embedded `SKILL.md` (`name` matches directory, non-empty `description`) and verifies required safety guardrails and `gradusctl` CLI references.
     - Verifies `compile.ApplyCompile` extracts all 6 skills cleanly into both `.claude/skills/` and `.agents/skills/`.
     - Parses and validates all 7 YAML scenario files under `../evals/skills/`, verifying every canonical skill plus prompt-injection resistance is covered with non-empty `baseline_rationalizations`, `required_behaviors`, and `forbidden_behaviors`.

## Options Considered

1. **YAML Scenario Specs vs. Prose-Only Markdown Evals**
   - *Option A:* Write eval scenarios as free-form Markdown prose.
   - *Option B (Chosen):* Store eval scenarios as structured YAML (`evals/skills/*.yaml`) alongside `evals/skills/README.md`.
   - *Rationale:* Structured YAML is directly machine-parseable by `skills/skills_test.go` in CI and reusable by the upcoming F8 benchmark harness (`evals/benchmark/`).

2. **Monolithic Single Skill vs. Six Focused Lifecycle Skills**
   - *Option A:* Combine all workflows into one large `gradus/SKILL.md`.
   - *Option B (Chosen):* Six dedicated skills (`onboard`, `author`, `review`, `learn`, `reconcile`, `conflicts`) as specified in `gradus-design.md` §16.
   - *Rationale:* Keeps each skill concise, minimizes context token overhead, and allows targeted invocation per workflow.

## Simplicity And Elegance Review

- Zero new runtime dependencies: `skills/embed.go` already uses `//go:embed gradus-*/*.md`, so replacing `gradus-placeholder` with the six canonical directories works automatically with zero changes to `internal/compile`.
- Every skill delegates all state mutations to the deterministic `gradusctl` CLI built in F1–F6, preserving the strict boundary between LLM reasoning and deterministic enforcement.

## Failure Modes And Tradeoffs

- **Skill Drift vs. CLI Flags:** If a skill references a non-existent `gradusctl` subcommand, agents fail at runtime. Mitigated by verifying exact CLI command references in `TestEmbeddedSkillsAndEvals`.
- **Static vs. Live-LLM Verification Tradeoff:** `go test ./skills` deterministically validates embedding, frontmatter, guardrails, and scenario integrity offline in CI, while live LLM behavioral scoring against a plain-file baseline is handled by F8 (`evals/benchmark/`).

## Verification Plan

- `skills/skills_test.go` (`TestEmbeddedSkillsAndEvals`): validates all 6 embedded skills, absence of `gradus-placeholder`, YAML frontmatter, key safety guardrails, `compile.ApplyCompile` extraction into `.claude/skills/` and `.agents/skills/`, and all 7 `evals/skills/*.yaml` scenario specs.
- Full repository `go test -v -race -count=1 ./...` confirms existing `internal/compile` and `internal/cli` tests pass cleanly with the six real embedded skills.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `skills/gradus-{onboard,author,review,learn,reconcile,conflicts}/SKILL.md` + placeholder removal + `TestEmbeddedSkillsAndEvals` |
| `R2` | `R2.AC1`–`R2.AC3`: explicit guardrails & prohibitions in all 6 `SKILL.md` files verified by `TestEmbeddedSkillsAndEvals` |
| `R3` | `R3.AC1`–`R3.AC2`: `evals/skills/{README.md,*.yaml}` + `skills/skills_test.go` (`TestEmbeddedSkillsAndEvals`) |
| `NFR1` | `skills/embed.go` (`//go:embed gradus-*/*.md`) with zero external dependencies |
| `NFR2` | Concise canonical English prompts and structured YAML scenarios |
