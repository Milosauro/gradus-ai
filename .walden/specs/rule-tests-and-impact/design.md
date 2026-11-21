---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:18:46Z
last_modified: 2026-11-21T17:18:46Z
approved_fingerprint: sha256:afcb24cdf884233421f52990e36008cbce5c151821d9acb65419c6357b76a75d
source_requirements_approved_at: 2026-11-21T17:17:42Z
source_requirements_fingerprint: sha256:f1088bc7dddb875b988151b31fa76947d57a135c8efd437390770cecc15aa4f1
---

# Feature Design

## Architecture

### 1. Rule Testing Engine (`internal/check/ruletest.go`, `internal/cli/test.go`)
- **Inputs:** Load local pyramid rules via `schema.LoadLocalPyramid(dir)` (which preserves `Rule.Examples` with `Compliant` and `NonCompliant` code blocks).
- **Filtering:** If `--rule <id>` is passed, locate that rule (exit `2` if missing); otherwise test all active/deprecated `check` and `hook` rules that contain at least one example.
- **Evaluating `check` rules (`RunRuleTests`):**
  - Materialize each example snippet into a temporary directory (`os.MkdirTemp`) using a filename derived from the rule's first scope extension (or `sample.txt` fallback) so scope filtering and checkers run cleanly without touching the repo.
  - Execute `RunChecksOnFiles(tmpDir, singleRuleView, []string{relPath})` (a small helper in `internal/check/engine.go` extracted from `RunChecks` that runs checkers on an explicit file list without invoking `git ls-files`).
  - Assert `len(findings) == 0` for `Compliant` examples and `len(findings) > 0` for `Non-compliant` examples.
- **Evaluating `hook` rules:**
  - Strip fenced code block markers if present; construct a `guard.NormalizedAction` matching `rule.Guard.Action` (`command` -> `ActionCommand` with `Target = snippet`, `write` -> `ActionWrite` with `Target = snippet`).
  - Evaluate against `guard.Evaluate(action, []resolve.ResolvedRule{singleRule})`.
  - Assert `res.Decision == guard.DecisionAllow` for `Compliant` examples and `res.Decision == guard.DecisionAsk || res.Decision == guard.DecisionDeny` for `Non-compliant` examples.
- **CLI (`internal/cli/test.go`, registered in `root.go`):**
  - `gradusctl test [--rule <id>] [--dir .] [--json]` exits `0` when all cases pass, `1` if any case fails, `2` on invalid rule ID or checker error.

### 2. Historical Impact Simulation (`internal/check/simulate.go`, `internal/cli/simulate.go`)
- **Git History Inspection (`os/exec` `git`, `PRJ-GO-001`):**
  - Validate `--since` ref does not start with `-` (exit `2` if invalid).
  - Resolve commit list via `git rev-list -n <N> HEAD` (default `N=10`) or `git rev-list <since>..HEAD`.
  - For each commit SHA, discover changed files via `git diff-tree --no-commit-id --name-only -r --diff-filter=ACMR <sha>`.
  - For each scoped file matching an active `check` rule, read file contents at `<sha>` via `git show <sha>:<path>`, write it into a per-commit temporary directory at `<tmp>/<path>`, and execute `RunChecksOnFiles(tmpDir, view, matchedFiles)`.
- **Report & Exit Code:**
  - `SimulateReport{CommitsScanned, AffectedCommits, FindingsByRule, Commits: []SimulatedCommit}`.
  - Exits `0` on successful simulation (informational), `2` on invalid git ref or technical error.

## Options Considered

- **Chosen: Extract `RunChecksOnFiles(rootDir, view, files)` in `internal/check/engine.go` and materialize temporary files for `test` and `simulate`.**
  - Reuses both built-in `pattern` and external checker plugins identically across `check`, `test`, and `simulate` with zero working-tree mutations.
- **Alternative rejected: `git checkout <sha>` in the user's working tree during `simulate`.**
  - Rejected because mutating the user's working tree or index is destructive and unsafe (`NFR2`).

## Simplicity And Elegance Review

- Reuses `schema.Rule.Examples` already parsed by `internal/schema/parse_rule.go`, `internal/check`, and `internal/guard`.
- Zero new third-party dependencies; all git queries use `os/exec` `git`.

## Failure Modes And Tradeoffs

- **Option injection on `--since`:** Rejecting refs starting with `-` prevents git flag injection (`NFR1`).
- **Scope extension inference in `gradusctl test`:** When a `check` rule has `scope: ["**/*.py"]`, `gradusctl test` derives `sample.py` (or uses the literal path if no wildcard) so scoped rules match their own examples automatically.
- **Missing external checker:** Returns exit `2` (fail closed), never silently passing.

## Verification Plan

- Unit & CLI tests in `internal/cli/test_simulate_test.go` covering:
  - `gradusctl test --json` passing and failing on `check` (`pattern`) and `hook` rules, plus `--rule <id>` filtering and unknown rule exit `2`.
  - `gradusctl simulate --commits 2 --json` and `--since HEAD~1 --json` on a temporary git repository with 2 commits, verifying non-destructive execution and flag-injection rejection (`--since --exec=...` -> exit `2`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC3`: `RunRuleTests` in `internal/check/ruletest.go` and `internal/cli/test.go` testing `check` and `hook` rules against `Compliant`/`Non-compliant` Markdown examples with `--rule` filtering |
| `R2` | `R2.AC1`–`R2.AC3`: `RunSimulation` in `internal/check/simulate.go` and `internal/cli/simulate.go` scanning git history via `os/exec` `git` and reporting `commits_scanned`, `affected_commits`, and findings with exit `0` |
| `NFR1` | `os/exec` `git` with `-`-prefix ref rejection and `CGO_ENABLED=0` static build |
| `NFR2` | Temporary directories (`os.MkdirTemp` + `defer os.RemoveAll`) ensuring zero working-tree mutations |
| `C1` | Reuses `schema.Rule.Examples`, `internal/check`, and `internal/guard` with zero new dependencies |
