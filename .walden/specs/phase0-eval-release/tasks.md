---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:54:31Z
last_modified: 2026-11-20T15:00:33Z
approved_fingerprint: sha256:3a17f8bc79e78638338841a92bdaf2d2577220ed6a94e9d2c98368d88464fc97
source_design_approved_at: 2026-11-20T14:53:19Z
source_design_fingerprint: sha256:c34a1bb79e93bb32348435868a8b3d507346b261ad2bdccc763fd070d753a5ed
---

# Implementation Plan

- [x] 1. Comparative evaluation benchmark (`E9`)
  - [x] 1.1 Create `evals/benchmark/scenarios.yaml` (12 comparative scenarios across `claude-code` and `antigravity`), `evals/benchmark/RESULTS.md`, and `TestComparativeBenchmark` in `evals/benchmark/benchmark_test.go` asserting `gradusScore >= baselineScore` on all scenarios and `>` on deterministic scenarios
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR2`
    - Design: `evals/benchmark/`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestComparativeBenchmark$", "./evals/benchmark"]
        expect_output: "--- PASS: TestComparativeBenchmark"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "NFR2"]

- [x] 2. End-to-end Phase 0 exit criteria validation (`E1`–`E10`)
  - [x] 2.1 Add `TestPhase0ExitCriteria` in `evals/benchmark/exit_criteria_test.go` executing the §24.3 three-level setup (`team` git `locked: true`, `project`, `user`, `>= 10` real rules) and validating `E1`–`E8`
    - Requirements: `R2.AC1`, `NFR2`
    - Design: `evals/benchmark/exit_criteria_test.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestPhase0ExitCriteria$", "./evals/benchmark"]
        expect_output: "--- PASS: TestPhase0ExitCriteria"
        covers: ["R2.AC1", "NFR2"]
  - [x] 2.2 Update `README.md` with tested personal Quick Start setup commands (`github.com/Milosauro/gradus-ai`) and add `TestReadmePersonalSetup` (`E10`) in `evals/benchmark/exit_criteria_test.go`
    - Requirements: `R2.AC2`
    - Design: `README.md`, `evals/benchmark/exit_criteria_test.go`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestReadmePersonalSetup$", "./evals/benchmark"]
        expect_output: "--- PASS: TestReadmePersonalSetup"
        covers: ["R2.AC2"]

- [x] 3. Signed static release pipeline and `v0.1.0` changelog (`T3`)
  - [x] 3.1 Create `.goreleaser.yaml` (`CGO_ENABLED=0`, multi-OS/arch, `checksums.txt`, `syft` SBOM, keyless `cosign`), `.github/workflows/release.yml` (40-char SHA-pinned actions + provenance attestation), update `CHANGELOG.md` for `v0.1.0`, and add `TestReleasePipelineConfig` in `evals/benchmark/release_test.go`
    - Requirements: `R3.AC1`, `R3.AC2`, `R3.AC3`, `NFR1`
    - Design: `.goreleaser.yaml`, `.github/workflows/release.yml`, `CHANGELOG.md`
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestReleasePipelineConfig$", "./evals/benchmark"]
        expect_output: "--- PASS: TestReleasePipelineConfig"
        covers: ["R3.AC1", "R3.AC2", "R3.AC3", "NFR1"]
