---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T16:18:16Z
last_modified: 2026-11-25T16:18:16Z
approved_fingerprint: sha256:10e234dc8690006f800818a4f0cab1ff1e9d1ceaa653560aa2a9e99e0806bbf3
source_requirements_approved_at: 2026-11-25T16:18:16Z
source_requirements_fingerprint: sha256:d25899dbdcee201beafdab963cf3dc40cd18fcfdfb692969b5534799b8888d61
---

# Design: Enterprise Hardening & Design 1.2 (`v0.4.1`, `F29`)

## Architecture

1. **Root Discovery & Non-Gradus Pass-Through (`internal/cli/guard.go`, `internal/cli/status.go`):**
   - Add `findGradusRoot(startDir string) (string, bool)` walking upward from `filepath.Abs(startDir)` until `gradus.yaml` is found (stopping if `.git` is reached without `gradus.yaml`, or at filesystem root).
   - In `newGuardCmd`: resolve `rootDir` via `findGradusRoot(rootDir)`. If not found, emit native `allow` (`{}` for Claude Code, `{"decision":"allow"}` for Antigravity, standard allow JSON otherwise) and return `nil` (exit `0`). If found in a parent dir, re-normalize `action.Target` relative to the discovered `rootDir`.
   - In `newStatusCmd`: when `hookMode || antigravityHook`, use `findGradusRoot(dir)`; if not found, return `nil` (exit `0`) silently (D-82).
2. **MCP Server Hardening (`internal/mcp/server.go`):**
   - Replace `bufio.Scanner` with `bufio.NewReader` + bounded line reader (`readBoundedLine(r, maxLineBytes)`): if a line exceeds `maxLineBytes` (4 MiB), drain until `\n`, emit JSON-RPC `-32600` (`request exceeds maximum line size`), and continue serving.
   - In `initialize`: parse optional `params.protocolVersion`; if non-empty, echo it back (otherwise default `"2024-11-05"`), and use `version.ResolvedVersion()` for `serverInfo.version`.
   - In `toolDefinitions()`: attach `annotations: { readOnlyHint: true, destructiveHint: false, idempotentHint: true }` to all 5 tools.
   - In `toolProposeRule`: validate `level` exists in `res.View.Levels` (and reject any `..`, `/`, or `\` characters) before formatting `proposed_path`.
3. **Historical Trailer Verification (`internal/trailers/trailers.go`) & `resolve --write` Guard (`internal/cli/resolve.go`):**
   - In `VerifyCommit`: load expected `gradus.lock` bytes from `git show <ref>:gradus.lock` (falling back to working tree `gradus.lock` only if untracked in `<ref>`). Whenever `Gradus-Lock-Hash` is present in the commit trailers, verify it matches `lock.HashBytes(expectedBytes)` (even when `--require-lock-hash` is false!).
   - In `internal/cli/resolve.go`: guard `lock.WriteLockfile` behind `len(res.View.Conflicts) == 0` so blocking conflicts never overwrite `gradus.lock`.
4. **Anti-Tamper Precision (`internal/guard/antitamper.go`):**
   - Add `.config/lefthook.yml` to `protectedGovernanceGlobs` and `protectedShellTargetsRe`.
   - Restrict `hookBypassEnvRe` to command/env-assignment prefix position (`(?:^|[;&|]\s*)(?:env\s+)?(?:[A-Za-z_][A-Za-z0-9_]*=\S+\s+)*(?:HUSKY=0|LEFTHOOK=0|SKIP=[^\s;&|]+|LEFTHOOK_EXCLUDE=[^\s;&|]+|GIT_CONFIG_COUNT=[^\s;&|]+)\b`) so `echo SKIP=foo` and `git commit -m "fix HUSKY=0"` are not false-positived.
5. **Design 1.2 & Docs (`docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/compatibility-matrix.md`, `README.md`, `CHANGELOG.md`):**
   - Bump `gradus-design.md` to Version `1.2`, update §20 and §21 labels, add decisions `D-84`–`D-89`.
   - Sync `SPEC.md` (`models` active), `compatibility-matrix.md` (Phase 3 capabilities), `README.md` (`v0.4.1`, `--model-tier` self-declaration note), and `CHANGELOG.md` (`v0.4.1`).

## Options Considered

- **Option A (Chosen):** Parent-directory `findGradusRoot` + silent exit `0` pass-through in non-Gradus directories. Directly solves machine-wide `managed-settings.json` deployment without weakening Gradus-governed repos.
- **Option B (Rejected):** Require `GRADUS_ROOT` env var. Breaks zero-config enterprise MDM rollout.

## Simplicity And Elegance Review

- Minimal, targeted fixes across existing packages with zero new dependencies.

## Failure Modes And Tradeoffs

- **Parent traversal stopping at `.git`:** Stopping when a `.git` boundary is crossed without `gradus.yaml` prevents a nested unrelated git repository from accidentally inheriting an outer directory's `gradus.yaml`.

## Verification Plan

- CLI tests in `internal/cli/v041_hardening_test.go` (`TestGuardAndStatusOutsideGradusV041`, `TestTrailersHistoricalAndResolveGuardV041`, `TestDesign12SyncV041`).
- MCP tests in `internal/mcp/server_test.go` (`TestMCPHardeningV041`).
- Guard tests in `internal/guard/guard_test.go` (`TestAntiTamperPrecisionV041`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/cli/guard.go`, `internal/cli/status.go` (`R1.AC1`–`R1.AC3`) |
| `R2` | `internal/mcp/server.go` (`R2.AC1`–`R2.AC3`) |
| `R3` | `internal/trailers/trailers.go`, `internal/cli/resolve.go`, `internal/guard/antitamper.go` (`R3.AC1`–`R3.AC3`) |
| `R4` | `docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/compatibility-matrix.md`, `README.md`, `CHANGELOG.md` (`R4.AC1`) |
| `NFR1` | Static `CGO_ENABLED=0` build and `git` via `os/exec` |
| `NFR2` | Canonical English across all repository files |
| `C1` | Backward-compatible schemas and exit codes |
| `C2` | Full Go test suite regression pass |
