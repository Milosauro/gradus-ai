---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T13:41:35Z
last_modified: 2026-11-20T13:41:35Z
approved_fingerprint: sha256:63a6db0b9a7a905e421efb7884f656f3ab0ae4303c694257e9bdc5e866df7051
source_requirements_approved_at: 2026-11-20T13:36:37Z
source_requirements_fingerprint: sha256:183e39e2efa412975d1c33695f2cfde37629cde7f2e4542c447afa458d6b70ae
---

# Feature Design

## Architecture

Feature **F6 (`cli-lifecycle-state`)** introduces the `internal/state` package and three CLI subcommands (`internal/cli/conflicts.go`, `internal/cli/lessons.go`, and `internal/cli/reconcile.go`) registered on `NewRootCmd` (`internal/cli/root.go`).

### Component Overview

1. **`internal/state/findings.go` — Semantic Conflict Candidate Discovery & Findings Store**
   - `FindingRecord` (YAML under `.gradus/findings/<id>.yaml`):
     - `Schema` (`gradus/v1`), `ID`, `RuleA`, `RuleB`, `HashA`, `HashB`, `Status` (`suspected` | `confirmed` | `dismissed`), `Message`, `UpdatedAt`.
     - Canonical pair ordering: normalizes `(ruleA, ruleB)` lexicographically (`RuleA < RuleB`) so `F-<ruleA>-<ruleB>.yaml` is deterministic regardless of flag argument order.
     - Directly compatible with `internal/resolve.loadConfirmedFindings` (which already reads `schema`, `id`, `rule_a`, `rule_b`, `status`, `message`).
   - `CandidatePair`:
     - `RuleA`, `RuleB`, `LevelA`, `LevelB`, `HashA`, `HashB`, `Reason` (overlap basis: shared keywords/scope).
   - `FindCandidates(view resolve.ResolvedView, existing []FindingRecord) []CandidatePair`:
     - Compares every pair of active, non-waived rules from distinct provenance containers (`r1.Level != r2.Level || r1.Provenance.Source != r2.Provenance.Source`).
     - Checks scope overlap (`scopesOverlap(r1.Scope, r2.Scope)`: global `**/*` overlaps everything; otherwise checks exact match, common prefix, or `doublestar.Match`).
     - Extracts normalized significant lowercase keywords (`len >= 4`, excluding common English stopwords) from title + statement; pairs sharing at least one keyword qualify.
     - Skips any pair where an existing finding has `Status == "confirmed" || Status == "dismissed"` AND matching `(HashA, HashB)`. If either rule's content hash changes, the pair is surfaced again automatically.
   - `ListFindings(rootDir string) ([]FindingRecord, error)` and `SaveFinding(rootDir string, rec FindingRecord) (string, error)`.

2. **`internal/state/redact.go` — Secret & PII Redactor (Threat Model T4)**
   - `RedactText(input string) string`: deterministically replaces high-signal secret/PII patterns with `[REDACTED]` using compiled standard-library regexes:
     - GitHub tokens (`ghp_...`, `gho_...`, `ghu_...`, `ghs_...`, `github_pat_...`)
     - OpenAI / Anthropic / generic `sk-...` API keys
     - AWS access key IDs (`AKIA[0-9A-Z]{16}`)
     - Bearer / JWT authorization tokens (`Bearer <token>`, `eyJ...eyJ...`)
     - Email addresses (`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)

3. **`internal/state/lessons.go` — Local Redacted Lesson Store (`~/.gradus/lessons/<project-key>/`)**
   - `LessonRecord` matching `spec/v1/schema/lesson.schema.json`:
     - `Schema`, `ID`, `Created`, `Kind` (`correction` | `repeated-mistake` | `preference` | `frustration`), `Statement`, `Evidence` (`[]string`), `Occurrences`, `SuggestedLevel`, `SuggestedEnforcement`, `Status` (`open` | `proposed` | `promoted` | `dismissed`). Zero personal activity metrics (P16).
   - `ProjectKey(rootDir string) string`: computes a deterministic cross-platform directory key from the cleaned absolute path (`<basename>-<short-sha256>`).
   - `LessonsDir(rootDir, overrideDir string) (string, error)`: uses `overrideDir` if non-empty, otherwise `~/.gradus/lessons/<ProjectKey(rootDir)>`.
   - `AddLesson(dir string, rec LessonRecord, now time.Time) (LessonRecord, string, error)`: validates `kind`, applies `RedactText` to `Statement` and every `Evidence` item, assigns deterministic/unique ID (`les-YYYYMMDD-HHMMSS-<4hex>`), sets `Occurrences = max(1, rec.Occurrences)` and `Status = "open"`, writes indented JSON.
   - `ListLessons(dir string) ([]LessonRecord, error)` (sorted newest-first).
   - `UpdateLesson(dir, id, status string, incrementOccurrences bool) (LessonRecord, error)`: validates status enum, updates record on disk.

4. **`internal/state/reconcile.go` — Drift Inspection, Restore, and Safe Rule Move**
   - `DriftItem`: `Path`, `Kind` (`missing` | `modified`), `Options` (`[]string{"restore", "move"}`).
   - `InspectDrift(rootDir string, cfg schema.PyramidConfig, view resolve.ResolvedView, skillsFS fs.FS) ([]DriftItem, error)`: computes `compile.PlanCompile` and compares every expected target file byte-for-byte with disk.
   - `RestoreDrift(rootDir string, cfg schema.PyramidConfig, view resolve.ResolvedView, skillsFS fs.FS) (compile.CompileResult, error)`: executes `compile.ApplyCompile` to restore all generated targets cleanly.
   - `MoveRuleToLevel(rootDir string, cfg schema.PyramidConfig, ruleFile, targetLevel string) (string, error)`:
     - Finds `targetLevel` in `cfg.Levels`; returns an error if not found, if `lvl.Locked == true`, or if `lvl.Source.Path == ""` (remote git levels cannot be mutated directly on disk).
     - Validates the rule file with `schema.ParseRuleFile`, copies it into `<levelDir>/rules/<basename>`, and removes the source file only when source and destination differ (if source is inside workspace).

5. **CLI Subcommands (`internal/cli/{conflicts,lessons,reconcile}.go`, wired in `root.go`)**
   - `gradusctl conflicts [--candidates] [--confirm] [--dismiss] [--rule-a <id>] [--rule-b <id>] [--message <text>] [--json]`
   - `gradusctl lessons add|list|update [--lessons-dir <dir>] [--json]`
   - `gradusctl reconcile [--restore] [--move --rule-file <path> --target-level <id>] [--json]`

## Options Considered

1. **YAML vs. JSON for `.gradus/findings/`**
   - *Option A:* Store findings as JSON.
   - *Option B (Chosen):* Store findings as YAML (`*.yaml`) under `.gradus/findings/`.
   - *Rationale:* `internal/resolve/resolve.go` (`loadConfirmedFindings`) already reads `.gradus/findings/*.yaml`, so YAML ensures seamless round-trip integration with zero changes to `internal/resolve`.

2. **Pair Key Normalization in Findings**
   - *Option A:* Preserve caller's `--rule-a` / `--rule-b` order.
   - *Option B (Chosen):* Sort `(RuleA, RuleB)` lexicographically (swapping hashes accordingly) before generating the finding ID and filename.
   - *Rationale:* Prevents duplicate findings if a user confirms `(A, B)` and later dismisses `(B, A)`.

## Simplicity And Elegance Review

- Reuses `internal/resolve.Resolve`, `internal/compile.PlanCompile`/`ApplyCompile`, `internal/schema.ParseRuleFile`, and `internal/lock.HashBytes` directly without duplicating any core logic.
- All state persistence (`findings.go`, `lessons.go`, `redact.go`, `reconcile.go`) lives in one cohesive `internal/state` package with zero Cobra dependency, making unit testing fast and portable across macOS, Linux, and Windows.

## Failure Modes And Tradeoffs

- **Locked or Remote Target Level on `reconcile --move`:** Rejected with exit code `2` before any filesystem write occurs.
- **Regex-Based Redaction Tradeoff:** Pattern-based redaction catches common API keys, Bearer/JWT tokens, AWS keys, and email addresses deterministically with zero external dependencies, though arbitrary custom secret formats may still require user review.
- **Unknown Rule IDs on `conflicts --confirm/--dismiss`:** Returns exit code `2` rather than writing an orphan finding with empty hashes.

## Verification Plan

1. **`internal/state/state_test.go`:**
   - `TestFindingsAndCandidates`: tests cross-level candidate discovery, `--dismiss` hash-based suppression, re-surfacing after rule hash change, and `--confirm` persistence compatible with `resolve.Resolve`.
   - `TestLessonsAndRedaction`: tests `RedactText` on GitHub tokens, `sk-` keys, AWS keys, Bearer tokens, and emails, plus `AddLesson` / `ListLessons` / `UpdateLesson` lifecycle.
   - `TestReconcileState`: tests `InspectDrift`, `RestoreDrift`, `MoveRuleToLevel` on unlocked level (pass) and locked level (error).
2. **`internal/cli/` integration tests:**
   - `TestConflictsCommand` (`internal/cli/conflicts_test.go`): end-to-end `--candidates`, `--dismiss`, `--confirm` (causing `resolve` semantic conflict), and default list.
   - `TestLessonsCommand` (`internal/cli/lessons_test.go`): end-to-end `add` (verifying redacted JSON on disk), `list`, and `update`.
   - `TestReconcileCommand` (`internal/cli/reconcile_test.go`): end-to-end clean (`0`), drifted (`1`), `--restore` (`0` + `verify` pass), `--move` unlocked (`0`), and `--move` locked (`2`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC4`: `internal/state/findings.go` + `internal/cli/conflicts.go` (`TestFindingsAndCandidates`, `TestConflictsCommand`) |
| `R2` | `R2.AC1`–`R2.AC3`: `internal/state/redact.go`, `internal/state/lessons.go` + `internal/cli/lessons.go` (`TestLessonsAndRedaction`, `TestLessonsCommand`) |
| `R3` | `R3.AC1`–`R3.AC4`: `internal/state/reconcile.go` + `internal/cli/reconcile.go` (`TestReconcileState`, `TestReconcileCommand`) |
| `NFR1` | `internal/state/redact.go` (`RedactText`) + P16-clean `LessonRecord` schema |
| `NFR2` | `ProjectKey` and `filepath.ToSlash` normalization across `internal/state` |
| `NFR3` | `ExitErrorWithCode{Code: ExitError}` (`2`) on invalid inputs, unknown rule IDs, and locked targets |
