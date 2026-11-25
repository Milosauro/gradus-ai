---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:56:10Z
last_modified: 2026-11-25T09:56:10Z
approved_fingerprint: sha256:8beec994841efc7f4417777b9d80bc003a9b62f74928c834038b1c2b048ac707
source_requirements_approved_at: 2026-11-25T09:56:09Z
source_requirements_fingerprint: sha256:2db72fdc92eb8a7c672c2fbd91bd0b84341226cb153d8b27348d835b8cbd08c9
---

# Design: Maturity Assessment & Commit Trailers (`F28`)

## Architecture

1. **Process Maturity Assessment (`internal/assess/assess.go` + `internal/cli/assess.go`):**
   - `Evaluate(rootDir string) (Report, error)` inspects `gradus.yaml`, `gradus.lock`, resolved view, and local evidence records (`.gradus/evidence/*.json`) across 5 dimensions (each `0..20` points, total `0..100`):
     1. `pyramid_integrity` (`0..20`): valid `gradus.yaml`, `gradus.lock` present, zero blocking conflicts, locked upper level or remote signature policy.
     2. `enforcement_depth` (`0..20`): deterministic `check` rules, runtime `hook`/`permission` rules, and runnable rule tests (`Compliant`/`Non-compliant` examples).
     3. `change_governance` (`0..20`): level charters, exception policy (`max_duration`/`approvers`), `anti_tamper` configured, git hook manager configured.
     4. `obligations_and_toolchain` (`0..20`): obligations/bindings, declared toolchain, model policy, or conditional profile overlays.
     5. `evidence_and_audit` (`0..20`): configured evidence sinks, local evidence records present, framework `controls:` mapped on rules.
   - Maturity levels: `L1 Initial` (0–29), `L2 Managed` (30–49), `L3 Defined` (50–69), `L4 Governed` (70–84), `L5 Verified` (85–100).
   - CLI: `gradusctl assess [--dir .] [--json]`.
2. **Embedded Skill (`skills/gradus-assess/SKILL.md` + `evals/scenarios/skill-08-assess.yaml`):**
   - Guides agents to run `gradusctl assess --json`, explain gaps in process controls, propose prioritized process improvements, and strictly refuse any request to score or rank individual developers (P16).
   - Registered in `skills/skills_test.go` (`canonicalSkills` + `requiredSnippets`).
3. **Commit Attribution Trailers (`internal/trailers/trailers.go` + `internal/cli/trailers.go`):**
   - `Generate(rootDir string, opts GenerateOptions) (TrailerSet, error)`: resolves `view_hash` and `lock_hash` (from `gradus.lock` if present), validates optional `ModelTier` (`resolve.TierRank`), and formats canonical git trailer lines (`Gradus-Lock-Hash`, `Gradus-View-Hash`, `Gradus-Agent`, `Gradus-Model-Tier`, `Gradus-Review-ID`).
   - `ParseCommitMessage(msg string) map[string]string` & `VerifyCommit(rootDir, ref string, opts VerifyOptions) (VerifyReport, error)`: runs `git log -1 --format=%B <ref>` via `os/exec` (`PRJ-GO-001`), parses `Gradus-*` trailers, validates `Gradus-Model-Tier` if present, and checks `Gradus-Lock-Hash` against current `gradus.lock` when `RequireLockHash` is true.
   - CLI: `gradusctl trailers generate` and `gradusctl trailers verify`.
4. **Phase 3 Closure Docs & Recompile:**
   - Update `docs/plan/phase-3-plan.md`, `README.md`, `CHANGELOG.md` (`v0.4.0`), and recompile `gradus-ai` targets so `gradus-assess` is installed in `.claude/skills/` and `.agents/skills/`.

## Options Considered

- **Option A (Chosen):** Deterministic 5-dimension rubric in Go + `gradus-assess` skill + standalone `gradusctl trailers generate|verify` commands. Offline, transparent, auditable, and strictly process-only.
- **Option B (Rejected):** Git-history author analysis for maturity scoring. Explicitly forbidden by Design Principle P16 (never measure people).

## Simplicity And Elegance Review

- Reuses `resolve.Resolve`, `resolve.TierRank`, `evidence.ListLocalRecords`, and `os/exec` `git` with zero new dependencies.

## Failure Modes And Tradeoffs

- **Fresh repo without `gradus.lock`:** `trailers generate` still emits `Gradus-View-Hash` and omits `Gradus-Lock-Hash` when no lockfile exists; `trailers verify --require-lock-hash` fails with exit `1` if the commit lacks a matching `Gradus-Lock-Hash`.

## Verification Plan

- Unit tests in `internal/assess/assess_test.go` (`TestProcessMaturityAssessment`) and `skills/skills_test.go` (`TestEmbeddedSkills`).
- End-to-end CLI tests in `internal/cli/assess_trailers_test.go` (`TestAssessAndTrailersCLI`).
- Static `CGO_ENABLED=0` build and full `gradusctl verify`/`check`/`test`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/assess/assess.go`, `skills/gradus-assess/SKILL.md`, `evals/scenarios/skill-08-assess.yaml` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/trailers/trailers.go`, `internal/cli/trailers.go` (`R2.AC1`, `R2.AC2`) |
| `R3` | `docs/plan/phase-3-plan.md`, `README.md`, `CHANGELOG.md`, compiled skills (`R3.AC1`) |
| `NFR1` | Static `CGO_ENABLED=0` build and `git` via `os/exec` |
| `NFR2` | Canonical English across all repository files |
| `C1` | Opt-in `trailers` commands leaving existing git workflows unchanged |
| `C2` | Full Go test suite regression pass |
