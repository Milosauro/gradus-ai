---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:31:56Z
last_modified: 2026-11-21T17:31:56Z
approved_fingerprint: sha256:2b5779e8d90b906f2b08b979fab0d772875473f576d43b0eff792879b5149814
source_requirements_approved_at: 2026-11-21T17:30:58Z
source_requirements_fingerprint: sha256:fd5c17b9ab5cc823f467c3d525cbea6d73b80aad8081ca7bc78e3dcf2b48605e
---

# Feature Design

## Architecture

### 1. Reusable Composite GitHub Action (`.github/actions/gradus/action.yml`)
- Composite action using only 40-character SHA-pinned external actions (`PRJ-CI-001`, Threat Model T3):
  - `actions/setup-go@41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed # v5.1.0`
  - `actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # v4.6.2` (when `upload-evidence == 'true'`)
- Inputs:
  - `version` (default `latest`), `working-directory` (default `.`), `base-ref` (optional), `write-evidence` (default `true`), `upload-evidence` (default `true`).
- Steps:
  1. Set up Go and install `github.com/Milosauro/gradus-ai/cmd/gradusctl@${{ inputs.version }}` with `CGO_ENABLED=0`.
  2. Run `gradusctl verify --json` (with optional `--write-evidence`), capturing verdict and exit code.
  3. Run `gradusctl check --json` (with `--base` if `base-ref` is set and `--write-evidence` if enabled), capturing verdict and exit code.
  4. Append a Markdown summary to `$GITHUB_STEP_SUMMARY`.
  5. Upload `.gradus/evidence/` via `actions/upload-artifact` if `upload-evidence == 'true'`.
  6. Exit with the highest non-zero exit code (`2` > `1` > `0`).

### 2. Team Agent Profiles (`internal/resolve/`, `internal/guard/`, `internal/cli/`)
- **Resolver (`internal/resolve/resolve.go`):**
  - Add `AgentProfile string` to `resolve.Options` (falls back to `os.Getenv("GRADUS_AGENT_PROFILE")` when empty).
  - When `AgentProfile` is non-empty:
    - Validate directory exists (`os.Stat`); if missing or not a directory, return error (`ExitError` / `2`).
    - Skip any `user` level (`lvl.ID == "user"` or `~/...` path) so personal developer rules are replaced by the team-defined agent profile (§21).
    - Resolve `AgentProfile` as a local level with ID `"agent-profile"` appended after all configured levels, enforcing P2 restriction invariant and charters like any other level.
- **Guard (`internal/guard/evaluator.go`, `internal/cli/guard.go`):**
  - Update `LoadLocalGuardRules(rootDir, userLevelDir string)` / add `LoadLocalGuardRulesWithProfile(rootDir, userLevelDir, agentProfile string)`: when `agentProfile != ""` (or `GRADUS_AGENT_PROFILE` is set), load `agent-profile` rules from `agentProfile` (failing closed if missing) and do NOT load `~/.gradus/levels/user`.
- **CLI flags (`internal/cli/resolve.go`, `compile.go`, `guard.go`):**
  - Add `--agent-profile` flag to `resolve`, `compile`, and `guard` (each falling back to `GRADUS_AGENT_PROFILE`).

## Options Considered

- **Chosen: Resolve `--agent-profile <dir>` as a standard level (`id: agent-profile`) at the bottom of the pyramid replacing `user`.**
  - Reuses `level.yaml` + `rules/*.md`, P2 restriction invariant, charter checks, compilers, and `gradusctl guard` with zero new file formats (`C1`).
- **Alternative rejected: Introduce a separate `agent-profile.yaml` format.**
  - Rejected because teams already version and review `level.yaml` + `rules/*.md` directories.

## Simplicity And Elegance Review

- Composite GitHub Action avoids JavaScript/Node bundling and maintenance overhead while working across Linux/macOS/Windows runners.
- Reusing `level.yaml` + `rules/*.md` for agent profiles keeps the pyramid model uniform.

## Failure Modes And Tradeoffs

- **Missing agent profile directory:** Fails closed with exit `2` in `resolve`, `compile`, and `guard` so CI agents never silently run without their guardrails.
- **Floating action tags:** Enforced by an automated test verifying every `uses:` line in `.github/actions/gradus/action.yml` matches a 40-char hex SHA (`PRJ-CI-001`).

## Verification Plan

- Automated test `TestGitHubActionSpec` in `evals/benchmark/release_test.go` (or `internal/cli/agent_profile_test.go`) validating `.github/actions/gradus/action.yml` YAML structure, required inputs, `$GITHUB_STEP_SUMMARY`, and 40-char SHA pinning on every `uses:`.
- Integration test `TestAgentProfileWorkflow` in `internal/cli/agent_profile_test.go` verifying `--agent-profile` / `GRADUS_AGENT_PROFILE` on `resolve`, `compile`, and `guard`, verifying `user` exclusion and missing-dir exit `2`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `.github/actions/gradus/action.yml` composite action with SHA-pinned `uses:` verified by `TestGitHubActionSpec` |
| `R2` | `R2.AC1`–`R2.AC3`: `Options.AgentProfile` in `internal/resolve/resolve.go`, `LoadLocalGuardRulesWithProfile` in `internal/guard/evaluator.go`, and `--agent-profile` CLI flags verified by `TestAgentProfileWorkflow` |
| `NFR1` | 40-char SHA pinning check (`PRJ-CI-001`), canonical English, `CGO_ENABLED=0` |
| `NFR2` | Omitting `--agent-profile` / `GRADUS_AGENT_PROFILE` leaves existing behavior unchanged |
| `C1` | Agent profiles use standard `level.yaml` + `rules/*.md` layout |
