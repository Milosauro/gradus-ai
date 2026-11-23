---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T11:10:44Z
last_modified: 2026-11-23T11:15:00Z
approved_fingerprint: sha256:3145d374ae915b20cf9fb199ffc34235cdaa0b4cd9a8e56b61903c02759f42fa
source_design_approved_at: 2026-11-23T11:02:51Z
source_design_fingerprint: sha256:043898d3b8b9d19515e3709b17fb410a78b57ca34e27a9bae6d3d42715670c90
---

# Implementation Plan

- [x] 1. Add schema fields (`signature`, `min_version`) and stdlib SemVer comparator
  - [x] 1.1 Extend `LevelRef`, `LevelConfig`, JSON Schemas, config validation, and `internal/schema/semver.go` with unit tests
    - Requirements: `R1.AC4`, `R2.AC3`, `C1`, `C2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestSemVerAndRemoteLevelValidation$", "./internal/schema"]
        expect_output: "--- PASS: TestSemVerAndRemoteLevelValidation"
        covers: ["R1.AC4", "R2.AC3"]
      - command: ["go", "test", "-v", "-count=1", "./spec/v1"]
        expect_output: "--- PASS: TestJSONSchemasCompileAndValidate"
        covers: ["C1"]

- [x] 2. Implement Git signature verification and `min_version` enforcement in `source` and `resolve`
  - [x] 2.1 Implement `VerifyGitSignature` in `internal/source/signature.go` using `os/exec` `git` with SSH-signed tag/commit tests
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR1`, `NFR2`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestVerifyGitSignature$", "./internal/source"]
        expect_output: "--- PASS: TestVerifyGitSignature"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "NFR1", "NFR2"]
  - [x] 2.2 Integrate signature policy (`gradus.lock` `signature`) and `min_version` enforcement into `internal/resolve` and `gradusctl resolve`/`verify`
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `R2.AC1`, `R2.AC2`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestResolveRemoteSignaturesAndMinVersion$", "./internal/resolve"]
        expect_output: "--- PASS: TestResolveRemoteSignaturesAndMinVersion"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "R2.AC1", "R2.AC2"]

- [x] 3. Implement `gradusctl changelog` for readable level diffs
  - [x] 3.1 Implement non-destructive `internal/changelog` (`git archive` + `archive/tar` diff engine and Markdown formatter) with unit tests
    - Requirements: `R3.AC1`, `NFR1`, `NFR2`, `NFR3`, `C2`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestDiffLevelChangelog$", "./internal/changelog"]
        expect_output: "--- PASS: TestDiffLevelChangelog"
        covers: ["R3.AC1", "NFR2", "NFR3"]
  - [x] 3.2 Wire `gradusctl changelog` command in `internal/cli/changelog.go` with default lock/config refs, JSON/Markdown output, and error handling
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR1`, `NFR3`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestChangelogCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestChangelogCommand"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
