---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T13:44:55Z
last_modified: 2026-11-20T13:48:15Z
approved_fingerprint: sha256:f98b441bc366d52aa11c43e2a0801210e1badce26b6ef4d6df00b8e783e6376e
source_design_approved_at: 2026-11-20T13:41:35Z
source_design_fingerprint: sha256:63a6db0b9a7a905e421efb7884f656f3ab0ae4303c694257e9bdc5e866df7051
---

# Implementation Plan

- [x] 1. Semantic conflict candidate discovery and `gradusctl conflicts`
  - [x] 1.1 Implement `FindingRecord`, `CandidatePair`, `FindCandidates`, `ListFindings`, and `SaveFinding` (canonical pair ordering, scope/keyword overlap, hash-based suppression of judged pairs) in `internal/state/findings.go`, and add `TestFindingsAndCandidates` in `internal/state/state_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `R1.AC4`, `NFR2`
    - Design: `internal/state/findings.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestFindingsAndCandidates$", "./internal/state"]
        expect_output: "--- PASS: TestFindingsAndCandidates"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "R1.AC4", "NFR2"]
  - [x] 1.2 Implement `newConflictsCmd` (`--candidates`, `--confirm`, `--dismiss`, default list, `--json`, exit code `2` on unknown rule IDs) in `internal/cli/conflicts.go`, wire it in `internal/cli/root.go`, and add `TestConflictsCommand` in `internal/cli/conflicts_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `R1.AC4`, `NFR3`
    - Design: `internal/cli/conflicts.go`, `internal/cli/root.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestConflictsCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestConflictsCommand"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "R1.AC4", "NFR3"]

- [x] 2. Redacted local lesson store and `gradusctl lessons`
  - [x] 2.1 Implement `RedactText` (API keys, GitHub/AWS/Bearer/JWT tokens, emails -> `[REDACTED]`) in `internal/state/redact.go` and `LessonRecord`, `ProjectKey`, `LessonsDir`, `AddLesson`, `ListLessons`, `UpdateLesson` in `internal/state/lessons.go`, and add `TestLessonsAndRedaction` in `internal/state/state_test.go`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR2`
    - Design: `internal/state/redact.go`, `internal/state/lessons.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestLessonsAndRedaction$", "./internal/state"]
        expect_output: "--- PASS: TestLessonsAndRedaction"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR1", "NFR2"]
  - [x] 2.2 Implement `newLessonsCmd` (`add`, `list`, `update`, `--lessons-dir`, `--json`) in `internal/cli/lessons.go`, wire it in `internal/cli/root.go`, and add `TestLessonsCommand` in `internal/cli/lessons_test.go`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR3`
    - Design: `internal/cli/lessons.go`, `internal/cli/root.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestLessonsCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestLessonsCommand"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR1", "NFR3"]

- [x] 3. Drift reconciliation and `gradusctl reconcile`
  - [x] 3.1 Implement `InspectDrift`, `RestoreDrift`, and `MoveRuleToLevel` (validating rule file and rejecting `locked: true` or unknown/remote levels) in `internal/state/reconcile.go`, and add `TestReconcileState` in `internal/state/state_test.go`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `R3.AC4`, `NFR2`
    - Design: `internal/state/reconcile.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestReconcileState$", "./internal/state"]
        expect_output: "--- PASS: TestReconcileState"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "R3.AC4", "NFR2"]
  - [x] 3.2 Implement `newReconcileCmd` (drift inspection `0`/`1`, `--restore`, `--move --rule-file --target-level` with exit `2` on locked levels, `--json`) in `internal/cli/reconcile.go`, wire it in `internal/cli/root.go`, and add `TestReconcileCommand` in `internal/cli/reconcile_test.go`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `R3.AC4`, `NFR3`
    - Design: `internal/cli/reconcile.go`, `internal/cli/root.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestReconcileCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestReconcileCommand"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "R3.AC4", "NFR3"]
