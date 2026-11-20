---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:53:19Z
last_modified: 2026-11-20T14:53:19Z
approved_fingerprint: sha256:c34a1bb79e93bb32348435868a8b3d507346b261ad2bdccc763fd070d753a5ed
source_requirements_approved_at: 2026-11-20T14:50:04Z
source_requirements_fingerprint: sha256:f1f2c1f207e2b9ddbe89a960096756eaeb40d1ea239565f6eecf3ba52304fb4d
---

# Feature Design

## Architecture

Feature **F8 (`phase0-eval-release`)** delivers three cohesive deliverables completing Phase 0 (`v0.1.0`) under Decision Gate **O-2** (`github.com/Milosauro/gradus-ai` retained as repository owner for `v0.1.0`):

### 1. Comparative Evaluation Benchmark (`evals/benchmark/`, Exit Criterion E9)
- **`evals/benchmark/scenarios.yaml`**: defines 10+ structured comparative scenarios across `claude-code` and `antigravity` targets covering:
  1. `BENCH-01`: PreToolUse `git commit` confirmation (`hook: ask`)
  2. `BENCH-02`: PreToolUse anti-tamper protection on `.agents/hooks.json` / `.claude/settings.json` (`ask`)
  3. `BENCH-03`: PreToolUse `rm -rf` command denial (`hook: deny`)
  4. `BENCH-04`: Deterministic `check` pattern violation detection (`exit 1`)
  5. `BENCH-05`: Missing external checker fail-closed safety (`exit 2`, never `0`)
  6. `BENCH-06`: Lockfile tamper detection on `locked: true` level (`verify` `exit 2`)
  7. `BENCH-07`: Compiled-target manual drift detection (`verify` `exit 1`)
  8. `BENCH-08`: P2 restriction invariant blocking lower-level relaxation (`resolve` `exit 1`)
  9. `BENCH-09`: Antigravity `< 12,000`-character chunked rule compilation & idempotency (`compile`)
  10. `BENCH-10`: Lesson secret/PII redaction (`lessons add` -> `[REDACTED]`)
  11. `BENCH-11`: Locked-level refusal on `reconcile --move` (`exit 2`)
  12. `BENCH-12`: Semantic conflict candidate hash suppression after `--dismiss`
- **`evals/benchmark/benchmark_test.go` (`TestComparativeBenchmark`)**:
  - Executes every scenario in `scenarios.yaml` for both `claude-code` and `antigravity` targets against both:
    - **Plain instruction-file baseline** (markdown instructions only; zero deterministic runtime/CLI enforcement on tool calls, drift, lock tamper, or redaction -> score `0.0` on deterministic enforcement gates, `1.0` on advisory-only baseline checks).
    - **Gradus** (full CLI + compiled targets + `guard` + `verify` + `check` + `state` -> score `1.0` on every scenario).
  - Asserts `len(scenarios) >= 10`, `gradusScore >= baselineScore` on every scenario, and `gradusScore > baselineScore` on every deterministic scenario (`E9`).
- **`evals/benchmark/RESULTS.md`**: committed human-readable summary table of methodology, reproduction command (`go test -v -count=1 ./evals/benchmark`), and per-scenario comparative results for Claude Code and Antigravity.

### 2. End-to-End Phase 0 Exit Criteria Validation (`E1`–`E10`)
- **`evals/benchmark/exit_criteria_test.go`**:
  - `TestPhase0ExitCriteria`: constructs the exact §24.3 three-level setup:
    - Level 1 (`team`, `locked: true`) hosted in a separate local bare/non-bare Git repository resolved via `source.git` + `gradus.lock`.
    - Level 2 (`project`) local path `.gradus/levels/project`.
    - Level 3 (`user`) local user directory.
    - `>= 10` real rules including 2 `hook` rules, 2 `check` rules, 3 `prompt`/`review` rules, 1 path-scoped rule, 1 valid override (`overridable: true`), and 1 invalid override test case.
    - Exercises `E1` (idempotent `resolve` + `compile`, Antigravity `<12k`), `E2` (`verify` tamper/drift/conflict exit codes), `E3` (`guard` on both Claude Code and Antigravity payloads), `E4`–`E7` (skills + `lessons`, `reconcile`, `conflicts`), and `E8` (`hooks install` + `evidence` ci-artifact sink).
  - `TestReadmePersonalSetup`: executes the exact personal setup flow documented in `README.md` (`E10`) on a fresh temp repository (`init` -> `resolve --write` -> `compile` -> `verify` exit `0`).
- **`README.md`**: updated with a concise, tested **Quick Start (Personal Setup)** section matching `E10` and `github.com/Milosauro/gradus-ai`.

### 3. Signed Static Release Pipeline & Changelog (Threat Model T3)
- **`.goreleaser.yaml`**:
  - `version: 2`, `CGO_ENABLED=0`, builds `./cmd/gradusctl` with `-trimpath` and `version`/`commit`/`date` ldflags for `linux`, `darwin`, `windows` (`amd64`, `arm64`).
  - Archives (`tar.gz` / `zip` on Windows), `checksums.txt` (`sha256`), `sboms` (`syft` SPDX JSON), and `signs` (`cosign sign-blob` keyless bundle).
- **`.github/workflows/release.yml`**:
  - Triggers on `push: tags: ['v*']`.
  - Permissions: `contents: write`, `id-token: write`, `attestations: write`.
  - Every `uses:` pinned to an immutable 40-character commit SHA (`actions/checkout`, `actions/setup-go`, `sigstore/cosign-installer`, `goreleaser/goreleaser-action`, `actions/attest-build-provenance`), plus `syft` installation and provenance attestation over `dist/checksums.txt`.
- **`CHANGELOG.md`**: updated with complete `[0.1.0]` / `[Unreleased]` release notes covering Wave 0–Wave 3.
- **`evals/benchmark/release_test.go` (`TestReleasePipelineConfig`)**: verifies `.goreleaser.yaml` contains `CGO_ENABLED=0`, `sboms`, `signs`, `checksum`, and verifies every `uses:` in `.github/workflows/release.yml` is pinned to a 40-character hex commit SHA.

## Options Considered

1. **Deterministic Go Benchmark Harness vs. External Shell/Python Script**
   - *Option A:* Write a Bash/Python benchmark runner invoking external LLM APIs.
   - *Option B (Chosen):* Write a self-contained Go test suite (`evals/benchmark/`) executing real `gradusctl` CLI and package flows offline across macOS, Linux, and Windows.
   - *Rationale:* Runs reproducibly in CI with one command (`go test -v -count=1 ./evals/benchmark`), requires zero API keys or non-Go runtimes, and tests the actual compiled Go binary logic end-to-end.

## Simplicity And Elegance Review

- Reuses `internal/cli.ExecuteCode`, `internal/compile`, `internal/guard`, `internal/resolve`, `internal/state`, and `skills.FS` directly.
- Keeps all Phase 0 verification (`E1`–`E10`), benchmark (`E9`), and release-config hardening (`T3`) verifiable via `go test ./...`.

## Failure Modes And Tradeoffs

- **Offline vs. Live LLM Eval Tradeoff:** The deterministic harness in `evals/benchmark/` verifies all deterministic enforcement gates (`hook`, `check`, `verify`, `lock`, `redaction`, `reconcile`, `conflicts`) and structural skill contracts offline; live LLM grading on open-ended prose remains optional and out of CI's offline path.
- **Cosign/Syft Availability in Release Workflow:** Installed explicitly in `.github/workflows/release.yml` via SHA-pinned actions / verified release steps so GoReleaser `sboms` and `signs` succeed on `ubuntu-latest`.

## Verification Plan

- `go test -v -count=1 -run ^TestComparativeBenchmark$ ./evals/benchmark` (`R1.AC1`–`R1.AC3`, `NFR2`)
- `go test -v -count=1 -run ^TestPhase0ExitCriteria$ ./evals/benchmark` (`R2.AC1`)
- `go test -v -count=1 -run ^TestReadmePersonalSetup$ ./evals/benchmark` (`R2.AC2`)
- `go test -v -count=1 -run ^TestReleasePipelineConfig$ ./evals/benchmark` (`R3.AC1`–`R3.AC3`, `NFR1`)

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC3`: `evals/benchmark/{scenarios.yaml,benchmark_test.go,RESULTS.md}` (`TestComparativeBenchmark`) |
| `R2` | `R2.AC1`–`R2.AC2`: `evals/benchmark/exit_criteria_test.go` + `README.md` (`TestPhase0ExitCriteria`, `TestReadmePersonalSetup`) |
| `R3` | `R3.AC1`–`R3.AC3`: `.goreleaser.yaml`, `.github/workflows/release.yml`, `CHANGELOG.md`, `evals/benchmark/release_test.go` (`TestReleasePipelineConfig`) |
| `NFR1` | 40-char SHA pinning verified by `TestReleasePipelineConfig` |
| `NFR2` | Offline deterministic Go test execution in `evals/benchmark` |
