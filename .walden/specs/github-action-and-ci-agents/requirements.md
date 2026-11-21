---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:30:58Z
last_modified: 2026-11-21T17:30:58Z
approved_fingerprint: sha256:fd5c17b9ab5cc823f467c3d525cbea6d73b80aad8081ca7bc78e3dcf2b48605e
---

# Requirements Document

## Introduction

`github-action-and-ci-agents` (`F13`, §12, §21, §24.1, X-09) delivers a reusable composite GitHub Action (`.github/actions/gradus/action.yml`) for running `gradusctl verify` and `gradusctl check` in CI with evidence artifact output, plus team-defined **agent profiles** (`--agent-profile <path>`) that replace the personal `user` level for background and CI agents.

## Requirements

### R1 Reusable Composite GitHub Action

**User Story:** As a repository maintainer, I want a reusable composite GitHub Action in `.github/actions/gradus/action.yml`, so that any GitHub repository can enforce `gradusctl verify` and `gradusctl check` on pull requests with SHA-pinned dependencies and structured job summaries.

#### Acceptance Criteria

1. `R1.AC1` WHEN `.github/actions/gradus/action.yml` is inspected, the composite action SHALL declare inputs (`version`, `working-directory`, `base-ref`, `write-evidence`, `upload-evidence`), run `gradusctl verify --json` and `gradusctl check --json`, write a Markdown summary to `$GITHUB_STEP_SUMMARY`, and preserve exit codes `0`/`1`/`2`.
   - Acceptance check: Parsing `.github/actions/gradus/action.yml` confirms valid composite action YAML with all required inputs, `gradusctl verify`, `gradusctl check`, and `$GITHUB_STEP_SUMMARY` reporting.
2. `R1.AC2` WHEN `.github/actions/gradus/action.yml` references any external GitHub Action via `uses:`, every `uses:` reference SHALL be pinned to an immutable 40-character commit SHA with an accompanying version comment (`PRJ-CI-001`, Threat Model T3).
   - Acceptance check: Automated YAML/regex verification confirms zero floating tags (`@v4`, `@main`) in `.github/actions/gradus/action.yml`.

### R2 Background and CI Agent Profiles

**User Story:** As a team lead running background or CI coding agents, I want `--agent-profile <path>` (or `GRADUS_AGENT_PROFILE`) on `gradusctl resolve`, `compile`, and `guard`, so that headless/CI agents inherit a team-defined agent profile level in place of a developer's personal `user` level.

#### Acceptance Criteria

1. `R2.AC1` WHEN `--agent-profile <dir>` (or `GRADUS_AGENT_PROFILE=<dir>`) is supplied to `gradusctl resolve` or `gradusctl compile`, the resolver SHALL load rules from `<dir>` as an `agent-profile` level at the bottom of the pyramid (skipping any personal `~/...` `user` level) and include those rules in `ResolvedView.Rules`.
   - Acceptance check: Resolving with `--agent-profile <dir>` includes the agent-profile rule (e.g., `AGENT-PR-001`) while excluding personal `user` rules even when `CI` is unset.
2. `R2.AC2` WHEN `--agent-profile <dir>` (or `GRADUS_AGENT_PROFILE=<dir>`) is supplied to `gradusctl guard`, the guard evaluator SHALL load hook rules from `<dir>` instead of `~/.gradus/levels/user` and enforce them deterministically.
   - Acceptance check: A `hook` rule (`decision: deny`) inside `<dir>` blocks matching commands in `gradusctl guard --agent-profile <dir>`.
3. `R2.AC3` WHEN `--agent-profile <dir>` points to a non-existent directory, `gradusctl resolve` and `gradusctl guard` SHALL fail closed with exit code `2`.
   - Acceptance check: Passing `--agent-profile /nonexistent/profile` exits with code `2`.

## Non-Functional Requirements

- `NFR1` **Supply-Chain & Static Build:** Every `uses:` in `.github/actions/gradus/action.yml` MUST use a 40-character commit SHA (`PRJ-CI-001`), and all Go code MUST compile with `CGO_ENABLED=0` (`PRJ-GO-001`) in canonical English (`PRJ-LANG-001`).
- `NFR2` **Backward Compatibility:** Omitting `--agent-profile` / `GRADUS_AGENT_PROFILE` MUST preserve exact Phase 0–F12 resolution and guard behavior.

## Constraints And Dependencies

- `C1` Agent profiles MUST follow the existing `level.yaml` + `rules/*.md` directory layout so teams can version and review them like any other level.

## Out Of Scope

- Publishing the action to the public GitHub Marketplace (users reference `Milosauro/gradus-ai/.github/actions/gradus@<sha>`).
