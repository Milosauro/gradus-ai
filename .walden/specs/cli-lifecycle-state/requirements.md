---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T13:36:37Z
last_modified: 2026-11-20T13:36:37Z
approved_fingerprint: sha256:183e39e2efa412975d1c33695f2cfde37629cde7f2e4542c447afa458d6b70ae
---

# Requirements Document

## Introduction

Feature **F6 (`cli-lifecycle-state`)** implements the lifecycle state management package (`internal/state`) and three CLI subcommands (`gradusctl conflicts`, `gradusctl lessons`, and `gradusctl reconcile`) required by Phase 0 Wave 2 (`docs/plan/phase-0-plan.md`, `gradus-design.md` §9.8, §10.1, §11.5, §16, Threat Model T4, and P16). These commands provide deterministic backing state for the upcoming Agent Skills (`F7`).

## Requirements

### R1 Semantic Conflict Candidate Discovery and Findings (`gradusctl conflicts`)

**User Story:** As a rule maintainer, I want `gradusctl conflicts` to discover candidate rule pairs across pyramid levels/overlays and persist confirmed or dismissed judgments by rule content hash, so that semantic conflict review is incremental and never repeats already-judged unchanged pairs.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl conflicts --candidates` runs, the system SHALL compare active rules across distinct levels or overlays whose `scope` patterns overlap and that share tags or significant statement keywords, skipping pairs already recorded in `.gradus/findings/` with identical `hash_a` and `hash_b`.
   - Acceptance check: Two rules in different levels with overlapping scopes and shared keywords appear in `--candidates`, and disappear from `--candidates` after being dismissed or confirmed with their current hashes.
2. `R1.AC2` WHEN `gradusctl conflicts --confirm --rule-a <id> --rule-b <id>` runs, the system SHALL write or update a finding record in `.gradus/findings/` with `status: confirmed` and current rule content hashes so subsequent `resolve` and `verify` runs treat it as a blocking semantic conflict.
   - Acceptance check: Confirming a pair creates a YAML finding file with `status: confirmed` and causes `gradusctl resolve` to report a `semantic` conflict.
3. `R1.AC3` WHEN `gradusctl conflicts --dismiss --rule-a <id> --rule-b <id>` runs, the system SHALL write or update a finding record in `.gradus/findings/` with `status: dismissed` and current `hash_a` and `hash_b` so the unchanged pair is not surfaced again by `--candidates`, while surfacing again if either rule's content hash changes.
   - Acceptance check: Dismissing a pair removes it from `--candidates` while rule hashes are unchanged, and modifying one of the rules restores the pair to `--candidates`.
4. `R1.AC4` WHEN `gradusctl conflicts` runs without mutation flags, the system SHALL list existing `.gradus/findings/*.yaml` records and exit with code `0`.
   - Acceptance check: Listing findings outputs all stored finding records in text or `--json` format with exit code `0`.

### R2 Redacted Local Lesson Store (`gradusctl lessons`)

**User Story:** As a developer, I want `gradusctl lessons` to store and manage local correction and preference lessons with automatic secret and PII redaction, so that agents can learn from sessions without leaking credentials or personal data.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl lessons add` runs with `--kind`, `--statement`, and optional `--level`, `--enforcement`, or `--evidence`, the system SHALL persist a lesson JSON file conforming to `spec/v1/schema/lesson.schema.json` under `~/.gradus/lessons/<project-key>/` (or `--lessons-dir` override) with zero personal activity metrics (P16).
   - Acceptance check: Adding a lesson writes a valid `gradus/v1` lesson JSON file with `occurrences: 1` and `status: open`.
2. `R2.AC2` WHEN persisting or updating a lesson, the system SHALL automatically redact common secrets and PII (such as API tokens `ghp_...`, `sk-...`, `AKIA...`, Bearer tokens, and email addresses) from `statement` and `evidence` with `[REDACTED]` before writing to disk (Threat Model T4).
   - Acceptance check: Supplying an email address and an API token in `--statement` or `--evidence` results in `[REDACTED]` on disk with zero raw secret or email strings stored.
3. `R2.AC3` WHEN `gradusctl lessons list` or `gradusctl lessons update --id <id>` runs, the system SHALL list stored lessons or update a lesson's `status` (`open`, `proposed`, `promoted`, `dismissed`) and/or increment `occurrences`.
   - Acceptance check: Listing shows stored lessons, and updating status to `promoted` persists the updated status across subsequent reads.

### R3 Drift Reconciliation (`gradusctl reconcile`)

**User Story:** As a developer or CI operator, I want `gradusctl reconcile` to inspect compiled-target drift and safely restore generated files or move custom rule edits into an editable level, so that drift can be resolved without bypassing locked governance levels.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl reconcile` runs without `--restore` or `--move`, the system SHALL inspect compiled target outputs against `PlanCompile` and list every drifted file, exiting with code `0` when clean and code `1` when drift exists.
   - Acceptance check: Inspecting a clean workspace returns exit `0` with zero drift items; editing a compiled target triggers exit `1` listing the drifted path.
2. `R3.AC2` WHEN `gradusctl reconcile --restore` runs on a workspace with compiled-target drift, the system SHALL re-apply `compile.ApplyCompile` deterministically to restore generated sections and files, returning exit code `0`.
   - Acceptance check: Running `gradusctl reconcile --restore` on a drifted `AGENTS.md` restores the expected managed section so a subsequent `gradusctl verify` passes with exit `0`.
3. `R3.AC3` WHEN `gradusctl reconcile --move --rule-file <path> --target-level <id>` runs targeting an unlocked level (`locked: false`), the system SHALL move or copy the rule file into that level's `rules/` directory and exit with code `0`.
   - Acceptance check: Moving a rule file into an unlocked `project` level places the file in `.gradus/levels/project/rules/` and exits with code `0`.
4. `R3.AC4` IF `gradusctl reconcile --move` targets a `locked: true` or unknown level, THEN the system SHALL refuse the modification without altering the target level and exit with code `2`.
   - Acceptance check: Targeting a `locked: true` level returns exit code `2` and leaves the locked directory unchanged.

## Non-Functional Requirements

- `NFR1` **Secret & PII Redaction (T4, P16):** Lesson persistence MUST redact secrets and email addresses before disk write and MUST omit personal user-activity metrics.
- `NFR2` **Deterministic Project Key & Cross-Platform Paths:** Project keys for `~/.gradus/lessons/<project-key>/` and finding paths MUST normalize slashes and paths identically across macOS, Linux, and Windows.
- `NFR3` **Exit Code Contract (D-73):** Technical errors (invalid YAML/JSON, missing rule IDs, locked target level on `--move`, missing `gradus.yaml`) MUST exit with code `2`.

## Constraints And Dependencies

- `C1` Must reuse `internal/resolve`, `internal/compile`, `internal/schema`, `internal/lock`, and `github.com/bmatcuk/doublestar/v4`.
- `C2` Must build statically with `CGO_ENABLED=0` and invoke `git` only via `os/exec` if needed.

## Out Of Scope

- LLM-based semantic classification itself (performed by the `gradus-conflicts` Agent Skill in F7 using `--candidates` output).
- Automatic rule authoring or session-history mining (performed by `gradus-learn` / `gradus-author` skills in F7).
