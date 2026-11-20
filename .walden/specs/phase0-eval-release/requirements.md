---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:50:04Z
last_modified: 2026-11-20T14:50:04Z
approved_fingerprint: sha256:f1f2c1f207e2b9ddbe89a960096756eaeb40d1ea239565f6eecf3ba52304fb4d
---

# Requirements Document

## Introduction

Feature **F8 (`phase0-eval-release`)** completes Phase 0 (`v0.1.0`) by delivering the reproducible comparative evaluation benchmark (`E9`, `evals/benchmark/`), end-to-end validation of Phase 0 exit criteria `E1`–`E10` (§24.3, including `README.md` personal setup `E10`), and the signed, SBOM-backed static release pipeline (`.goreleaser.yaml`, `.github/workflows/release.yml`, `CHANGELOG.md`, Threat Model T3).

## Requirements

### R1 Reproducible Comparative Evaluation Benchmark (`E9`)

**User Story:** As a Gradus maintainer and evaluator, I want a single-command comparative benchmark of at least 10 scenarios across Claude Code and Antigravity targets comparing Gradus against a plain instruction-file baseline, so that deterministic enforcement gains are objectively measured and committed.

#### Acceptance Criteria

1. `R1.AC1` WHEN `go test -v -count=1 ./evals/benchmark` executes `TestComparativeBenchmark`, the harness SHALL run at least 10 scenarios across both `claude-code` and `antigravity` targets comparing Gradus against a plain instruction-file baseline.
   - Acceptance check: `TestComparativeBenchmark` executes >= 10 scenarios for both targets and fails if fewer than 10 scenarios run.
2. `R1.AC2` WHEN comparing Gradus against the plain instruction-file baseline on each benchmark scenario, Gradus compliance SHALL be at least equal to the baseline on every scenario and strictly higher on deterministic enforcement scenarios (`hook`, `check`, `verify`, `lock`, `redaction`).
   - Acceptance check: Every scenario asserts `gradusScore >= baselineScore` and deterministic scenarios assert `gradusScore > baselineScore`, failing on regression.
3. `R1.AC3` The repository SHALL publish the benchmark scenarios, methodology, and committed results under `evals/benchmark/RESULTS.md` and `evals/benchmark/scenarios.yaml`.
   - Acceptance check: `evals/benchmark/RESULTS.md` and `evals/benchmark/scenarios.yaml` exist and match the benchmark scenario suite.

### R2 End-to-End Phase 0 Exit Criteria Validation (`E1`–`E10`)

**User Story:** As a release manager, I want an automated end-to-end test suite covering the §24.3 three-level setup (`team` git locked + `project` + `user`) and `README.md` personal onboarding flow, so that all Phase 0 exit criteria `E1`–`E10` are verified before tagging `v0.1.0`.

#### Acceptance Criteria

1. `R2.AC1` WHEN `TestPhase0ExitCriteria` runs in `evals/benchmark`, the test SHALL construct a three-level pyramid (simulated `team` git repo `locked: true`, `project`, and `user`) with >= 10 rules covering `hook`, `check`, `prompt`, scoped rules, valid overrides, and invalid overrides, verifying `E1`–`E8`.
   - Acceptance check: `TestPhase0ExitCriteria` verifies idempotent compilation, Antigravity `<12k` split, tamper/drift detection, `gradusctl guard` confirmation, `check` violations, `lessons` redaction, `conflicts` dismissal, and `hooks` installation.
2. `R2.AC2` WHEN a fresh repository is initialized following the documented commands in `README.md` (`E10`), `gradusctl resolve --write`, `gradusctl compile`, and `gradusctl verify` SHALL succeed with exit code `0`.
   - Acceptance check: `README.md` documents the complete personal setup commands and `TestReadmePersonalSetup` executes them end-to-end on a fresh temporary repository with exit `0`.

### R3 Signed Static Release Pipeline & Changelog (Threat Model T3)

**User Story:** As a project maintainer, I want a GoReleaser and GitHub Actions release pipeline producing `CGO_ENABLED=0` binaries, checksums, SBOMs, keyless `cosign` signatures, and provenance attestations, so that `v0.1.0` releases are verifiable and supply-chain hardened.

#### Acceptance Criteria

1. `R3.AC1` The repository SHALL define `.goreleaser.yaml` configured for `CGO_ENABLED=0` static builds across `linux`, `darwin`, and `windows` (`amd64`, `arm64`), `checksums.txt`, `syft` SBOM generation, and keyless `cosign` signing.
   - Acceptance check: `.goreleaser.yaml` contains `CGO_ENABLED=0`, multi-OS/arch targets, `checksum`, `sboms`, and `signs` configuration.
2. `R3.AC2` The repository SHALL define `.github/workflows/release.yml` triggered on `v*` tags with all `uses:` GitHub Actions pinned to 40-character commit SHAs, generating build provenance attestations alongside GoReleaser artifacts.
   - Acceptance check: `.github/workflows/release.yml` triggers on `v*` tags and every `uses:` reference is pinned to a 40-character hex commit SHA.
3. `R3.AC3` `CHANGELOG.md` SHALL document all Phase 0 MVP capabilities under `[0.1.0]` / `[Unreleased]` in canonical English ready for release tagging.
   - Acceptance check: `CHANGELOG.md` contains structured Phase 0 release notes covering F1–F8.

## Non-Functional Requirements

- `NFR1` **Supply-Chain Integrity (T3):** Every GitHub Action in `.github/workflows/release.yml` MUST be pinned to an immutable 40-character commit SHA.
- `NFR2` **Reproducibility:** `go test -v -count=1 ./evals/benchmark` MUST run deterministically offline without requiring external LLM API keys or network access.

## Constraints And Dependencies

- `C1` Must satisfy `gradus-design.md` §23, §24.3 (`E1`–`E10`), §26 (Decision Gate `O-2`), and Threat Model `T3`.
- `C2` Must reuse existing `internal/cli`, `internal/compile`, `internal/resolve`, `internal/guard`, `internal/state`, and `internal/hooks` packages.

## Out Of Scope

- Running `git tag v0.1.0` or pushing release tags before explicit human confirmation on Decision Gate `O-2` (`github.com/Milosauro/gradus-ai` vs `gradus-ai` org) and `release/v0.1` branch creation.
