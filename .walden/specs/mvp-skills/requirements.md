---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:05:03Z
last_modified: 2026-11-20T14:05:03Z
approved_fingerprint: sha256:aeab029fe780e698ea5f3fe3535ed21c4903b8a93e728dbd1225cb30816fb90d
---

# Requirements Document

## Introduction

Feature **F7 (`mvp-skills`)** authors and embeds all six canonical MVP Agent Skills (`gradus-onboard`, `gradus-author`, `gradus-review`, `gradus-learn`, `gradus-reconcile`, `gradus-conflicts`) under `skills/`, replaces the Wave 0 placeholder skill (`skills/gradus-placeholder/`), and establishes the **TDD for skills** regression suite under `evals/skills/` (`gradus-design.md` §16, §22, §23, and Phase 0 exit criteria **E4**, **E5**, **E6**, **E7**).

## Requirements

### R1 Six Canonical MVP Agent Skills

**User Story:** As a developer and repository maintainer, I want all six canonical Gradus Agent Skills embedded in `gradusctl`, so that Claude Code and Antigravity can execute guided onboarding, rule authoring, code review, lesson learning, drift reconciliation, and semantic conflict triage consistently.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl compile` targets `claude-code` or `antigravity`, the embedded `skills.FS` SHALL provide `gradus-onboard/SKILL.md`, `gradus-author/SKILL.md`, `gradus-review/SKILL.md`, `gradus-learn/SKILL.md`, `gradus-reconcile/SKILL.md`, and `gradus-conflicts/SKILL.md` without `gradus-placeholder`.
   - Acceptance check: Compiling a project extracts all six `gradus-*/SKILL.md` directories into `.claude/skills/` and `.agents/skills/` with zero `gradus-placeholder` artifacts.
2. `R1.AC2` WHEN any canonical skill `SKILL.md` is parsed, the skill file SHALL include valid YAML frontmatter (`name` matching its directory name and non-empty `description`), explicit inputs, outputs, step-by-step `gradusctl` CLI workflow, and explicit prohibitions matching `gradus-design.md` §16.
   - Acceptance check: Each of the six `SKILL.md` files passes frontmatter validation and explicitly documents its required workflow and safety prohibitions.

### R2 Safety, Privacy, and Governance Guardrails in Skills (E4–E7)

**User Story:** As a security-conscious team lead, I want each skill to enforce Gradus's non-negotiable safety boundaries in its instructions, so that agents never bypass locked levels, leak secrets, fabricate violations, or mutate governance without approval.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradus-onboard` or `gradus-learn` processes repository history or session signals, the skill instructions SHALL forbid applying configuration changes without explicit user approval, forbid sending data outside the active model, and require secret/PII redaction (`gradusctl lessons add`).
   - Acceptance check: `gradus-onboard` and `gradus-learn` explicitly instruct approval-gating and secret/PII redaction.
2. `R2.AC2` WHEN `gradus-author` or `gradus-reconcile` proposes rule placements or drift resolution, the skill instructions SHALL forbid direct edits to `locked: true` levels and delegate mutations to `gradusctl reconcile` / `gradusctl resolve`.
   - Acceptance check: `gradus-author` and `gradus-reconcile` explicitly prohibit modifying locked levels directly.
3. `R2.AC3` WHEN `gradus-review` or `gradus-conflicts` evaluates code or candidate rule pairs, the skill instructions SHALL require citing rule ID, level, severity, provenance, and confidence, forbidding low-confidence violations, and recording evidence/findings strictly through `gradusctl` (`evidence` / `conflicts`).
   - Acceptance check: `gradus-review` and `gradus-conflicts` explicitly require provenance citations, confidence thresholds, and CLI-backed recording without autonomous blocking.

### R3 Skill-TDD Regression Scenarios (`evals/skills/`)

**User Story:** As a Gradus maintainer, I want a structured TDD-for-skills scenario suite under `evals/skills/`, so that every skill is backed by documented baseline failure modes, rationalizations, and testable assertions (§23).

#### Acceptance Criteria

1. `R3.AC1` The repository SHALL include at least one scenario specification under `evals/skills/` for each of the six MVP skills (plus a prompt-injection resistance scenario per §22), recording baseline failures/rationalizations and expected skill-guided behavior.
   - Acceptance check: `evals/skills/` contains validated scenario files covering all six skills and prompt-injection resistance.
2. `R3.AC2` WHEN `go test ./skills/...` runs, the test suite SHALL deterministically verify `skills.FS` embedding, frontmatter validity, absence of `gradus-placeholder`, and full mapping between the six skills and `evals/skills/` scenarios.
   - Acceptance check: Running `go test -v -count=1 ./skills` executes `TestEmbeddedSkillsAndEvals` and passes with zero skipped assertions.

## Non-Functional Requirements

- `NFR1` **Zero External Runtime Dependencies:** All six skills remain embedded in the static `gradusctl` binary via `skills/embed.go` (`//go:embed gradus-*/*.md`).
- `NFR2` **Canonical English & Conciseness:** All skill prompts and eval scenarios are written in canonical English and avoid bloated boilerplate so agent context budgets stay lean.

## Constraints And Dependencies

- `C1` Must align strictly with the existing CLI surface implemented in F1–F6 (`gradusctl init`, `resolve`, `compile`, `verify`, `check`, `evidence`, `conflicts`, `lessons`, `reconcile`).
- `C2` Must satisfy `gradus-design.md` §16, §22, §23, and exit criteria `E4`, `E5`, `E6`, `E7`.

## Out Of Scope

- Phase 2/3 skills (`gradus-link`, `gradus-assess`).
- Full comparative Gradus-vs-baseline agent benchmark runner (`evals/benchmark/`, covered by F8).
