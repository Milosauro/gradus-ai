---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T08:35:09Z
last_modified: 2026-11-25T08:35:09Z
approved_fingerprint: sha256:1bd160438b9aac6f1198d240ee833d359d33825066d7c0193785cc8d608ee089
---

# Requirements Document

## Introduction

Feature `F25` (`model-policy-and-tiers`) implements Design §21: activating the `models:` configuration block in `gradus.yaml` and `level.yaml`, enforcing monotonic capability tiers (`tier-high > tier-standard > tier-light`) across pyramid levels, compiling agent-specific `model:` frontmatter for Claude Code skills, recording `skill` and `model_tier` in evidence records, and allowing CI/`verify` to reject evidence produced below the required tier.

## Requirements

### R1 Model Policy Schema and Monotonic Resolution

**User Story:** As a security lead, I want to declare capability tiers (`tier-high`, `tier-standard`, `tier-light`), per-agent concrete model mappings, and per-operation minimum tiers in `level.yaml` and `gradus.yaml`, with lower levels prevented from relaxing upper-level tier requirements.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradus.yaml` or `level.yaml` declares a `models:` block (`tiers` mapping tier names to agent→model maps, and `operations` mapping operation names to `tier-high`, `tier-standard`, or `tier-light`), the schema parsers SHALL validate tier names without emitting unknown-key or reserved-key notes.
   - Acceptance check: `TestModelPolicySchemaAndResolution` verifies valid `models:` parsing with zero notes and invalid tier names rejected.
2. `R1.AC2` WHEN `resolve.Resolve` merges `models:` across levels in pyramid order, lower levels MAY raise an operation's required tier (e.g. `tier-standard` → `tier-high`), whereas attempting to lower an upper level's required tier (e.g. `tier-high` → `tier-light`) SHALL emit a blocking conflict (`kind: "model_policy"`).
   - Acceptance check: `TestModelPolicySchemaAndResolution` verifies monotonic tightening succeeds and relaxation emits a blocking `model_policy` conflict.

### R2 Compiler Frontmatter Emission for Claude Code Skills

**User Story:** As a platform engineer, I want `gradusctl compile` to emit the Claude Code `model:` frontmatter field on compiled `.claude/skills/gradus-*/SKILL.md` files when `models:` maps a skill's operation to a concrete `claude-code` model, while keeping source skills open-standard.

#### Acceptance Criteria

1. `R2.AC1` WHEN `view.Models` defines a required tier for an operation corresponding to a Gradus skill (e.g. `review` → `gradus-review`, `author` → `gradus-author`, `conflicts` → `gradus-conflicts`, `learn` → `gradus-learn`) and maps that tier for `claude-code`, `PlanClaudeCode` SHALL inject `model: <concrete-model>` into the compiled `.claude/skills/gradus-<op>/SKILL.md` frontmatter.
   - Acceptance check: `TestClaudeCodeSkillModelInjection` verifies compiled `.claude/skills/gradus-review/SKILL.md` contains `model: opus` while source `skills/gradus-review/SKILL.md` remains unmodified.

### R3 Evidence Tier Recording and Verification

**User Story:** As a CI maintainer, I want `gradusctl evidence write` to record `--skill` and `--model-tier`, and `gradusctl verify --check-model-tier` to fail (exit `1`) when recorded evidence for a governed operation was produced below the required capability tier.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl evidence write` is invoked with `--skill <name>` and `--model-tier <tier>`, the persisted evidence record SHALL populate `producer.skill` and `producer.model_tier` (rejecting invalid tier names with exit code `2`).
   - Acceptance check: `TestModelPolicyEvidenceAndVerifyCLI` verifies `producer.skill` and `producer.model_tier` in written JSON evidence.
2. `R3.AC2` WHEN `gradusctl verify --check-model-tier` runs on a repository with declared `models.operations`, it SHALL exit `1` with a finding/conflict if the latest local evidence record for a governed operation has a `model_tier` ranked below the required tier (or missing), and exit `0` when all governed evidence meets or exceeds the required tier.
   - Acceptance check: `TestModelPolicyEvidenceAndVerifyCLI` verifies `tier-light` evidence fails against a `tier-high` requirement (exit `1`) and `tier-high` evidence passes (exit `0`).

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary invoking `git` via `os/exec` (`PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` Repositories without a `models:` block SHALL behave identically to `v0.3.2` (zero changes to compiled skills or `verify` behavior).
2. `C2` All existing unit and integration tests SHALL pass without regressions.
