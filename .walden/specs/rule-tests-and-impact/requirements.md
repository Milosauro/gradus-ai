---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:17:42Z
last_modified: 2026-11-21T17:17:42Z
approved_fingerprint: sha256:f1088bc7dddb875b988151b31fa76947d57a135c8efd437390770cecc15aa4f1
---

# Requirements Document

## Introduction

`rule-tests-and-impact` (`F12`, §16 `test` & `simulate`, §23, X-03) implements `gradusctl test` to verify deterministic `check` and `hook` rules against the `### Compliant` and `### Non-compliant` examples embedded in rule Markdown files, and `gradusctl simulate` to evaluate the resolved rule set across recent git commits and report historical impact before merging rule changes.

## Requirements

### R1 Declarative Rule Testing (`gradusctl test`)

**User Story:** As a rule author, I want `gradusctl test [--rule <id>] [--json]` to run my `check` and `hook` rules against their own `### Compliant` and `### Non-compliant` Markdown examples, so that regex/checker/guard mistakes are caught before publishing a rule.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl test [--json]` runs on a `check` rule (`checker.name: pattern` or external checker) containing `### Compliant` and `### Non-compliant` code blocks, the CLI SHALL assert that every `Compliant` example produces zero findings and every `Non-compliant` example produces at least one finding, exiting with `0` on full pass or `1` on any mismatch.
   - Acceptance check: A pattern rule whose `Compliant` block does not match and `Non-compliant` block matches passes with exit `0`; swapping them fails with exit `1` and identifies the failing example.
2. `R1.AC2` WHEN `gradusctl test [--json]` runs on a `hook` rule (`guard` configured) containing `### Compliant` and `### Non-compliant` examples, the CLI SHALL evaluate each example against `guard.Evaluate` and assert that `Compliant` examples return `allow` while `Non-compliant` examples return `ask` or `deny`.
   - Acceptance check: A `hook` rule guarding `git commit` passes when `git status` is under `### Compliant` and `git commit -m x` is under `### Non-compliant`, and exits `1` if a `Non-compliant` example is allowed.
3. `R1.AC3` WHEN `--rule <id>` is specified on `gradusctl test`, the CLI SHALL execute tests only for that rule ID (or exit `2` if the rule ID does not exist).
   - Acceptance check: `gradusctl test --rule PRJ-01 --json` runs only `PRJ-01` and returns exit `2` for an unknown ID.

### R2 Historical Impact Simulation (`gradusctl simulate`)

**User Story:** As a team lead proposing a new or stricter `check` rule, I want `gradusctl simulate [--since <ref> | --commits <N>] [--rule <id>] [--json]` to scan recent git commits and report which commits and files would have violated the rule, so that I can assess rollout noise before enforcing it.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl simulate --commits <N> [--json]` runs in a git repository, the CLI SHALL inspect the last `N` commits via `os/exec` `git` and report the total commits scanned, affected commits, and per-rule finding counts without modifying the working tree.
   - Acceptance check: Simulating a `pattern` rule over 2 commits where 1 commit introduced a matching file reports `commits_scanned: 2`, `affected_commits: 1`, and the matching commit SHA, file path, and rule ID, leaving `git status` unchanged.
2. `R2.AC2` WHEN `--since <ref>` is specified on `gradusctl simulate`, the CLI SHALL inspect all commits in `<ref>..HEAD` (or fail with exit `2` if `<ref>` is invalid or starts with `-`).
   - Acceptance check: `gradusctl simulate --since HEAD~1 --json` scans only the commit after `HEAD~1`, and `--since --exec=...` exits `2`.
3. `R2.AC3` WHEN `gradusctl simulate` completes successfully, the CLI SHALL exit with code `0` (impact reporting is informational, not a gate failure) while reporting all simulated findings in JSON and text output.
   - Acceptance check: Even when simulated findings are non-empty, `gradusctl simulate --json` exits `0` and lists all findings.

## Non-Functional Requirements

- `NFR1` **Git via `os/exec` & Static Build:** All git history queries in `simulate` MUST shell out to `git` via `os/exec` (`PRJ-GO-001`) with `CGO_ENABLED=0` and reject refs starting with `-`.
- `NFR2` **Non-Destructive Execution:** `gradusctl test` and `gradusctl simulate` MUST NOT modify the repository working tree, index, or HEAD (using temporary directories for example/blob materialization).

## Constraints And Dependencies

- `C1` Must reuse `schema.Rule.Examples` (`ExamplePair`), `internal/check`, and `internal/guard` without introducing new third-party dependencies.

## Out Of Scope

- LLM-based testing of natural-language `prompt` / `review` rules (handled offline by skills/evals).
