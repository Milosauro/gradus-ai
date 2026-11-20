---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-19T22:43:33Z
last_modified: 2026-11-19T22:43:33Z
approved_fingerprint: sha256:08463f52e6ecf2a2a8789d62f70c8f6b186349d43d101d3fb6161d3fa3fe1921
source_requirements_approved_at: 2026-11-19T22:39:18Z
source_requirements_fingerprint: sha256:041ac152455b6c43b29532f471d1604c730988bd510d5c6ca744989e7cae19c1
---

# Feature Design

## Architecture

Feature **F2 (`resolve-and-lock`)** introduces three deterministic internal packages and two CLI commands:

1. **`internal/source` (Local & Git Source Fetcher):**
   - `ResolveLevelSource(rootDir, id string, src schema.SourceRef, offline *schema.OfflinePolicy, locked bool, lockEntry *lock.Entry) (ResolvedSource, error)`:
     - **Local paths (`src.Path`):** Resolves workspace-relative paths or `~/` home paths (skipping `~/` with a note when `CI` is set, §9.2).
     - **Git sources (`src.Git` + `src.Ref`):** Checks `.gradus/cache/<id>/repo` via `os/exec` `git`. If a lock entry exists and matches the cached commit + directory hash, reuses the cache with zero network calls (§8). Otherwise runs `git clone`/`git fetch` + `git checkout` of `src.Ref` and records `Commit` SHA via `git rev-parse HEAD`.
     - **Offline policy:** When `git` fetch fails (unreachable remote), inspects `.gradus/cache/<id>/.fetched_at`. If `offline.Mode == "cache"` and elapsed time `<= MaxAge` (defaulting to `7d` if unset when `mode: cache`), returns cached path + diagnostic note. If `offline.Mode == "fail-closed"` or cache is missing/expired, returns an error mapped to exit code `2`.

2. **`internal/lock` (Normalized Hashing & `gradus.lock`):**
   - `HashDirectory(dir string) (string, error)`: Walks regular files (excluding `.git/` and hidden temp files), normalizes relative paths with `filepath.ToSlash`, sorts ASCII-lexicographically, normalizes `\r\n` -> `\n`, feeds `<relPath>\n<len>\n<bytes>\n` into SHA-256, and returns `"sha256:" + hex`.
   - `HashBytes(data []byte) string`: Computes `"sha256:" + hex` with `\r\n` -> `\n` normalization (used for `definition_hash` and `view_hash`).
   - `ReadLockfile(path string)` / `WriteLockfile(path string, lf Lockfile)`: Reads/writes `gradus.lock` YAML (§9.6). Excludes `user` level entries from `gradus.lock`.
   - **Integrity check:** When `gradus.lock` exists and a level/overlay is `locked: true`, compares computed directory hash against `entry.Hash`; any mismatch returns an error mapped to exit code `2`.

3. **`internal/resolve` (§10 Resolution Algorithm):**
   - `Resolve(rootDir string, opts Options) (ResolvedView, error)`:
     1. Loads & validates `gradus.yaml` (`ParsePyramidConfig`) and reads `gradus.lock` if present.
     2. Resolves all levels and overlays via `internal/source`, verifies `locked: true` hashes against `gradus.lock`.
     3. Parses `level.yaml` / `overlay.yaml` and checks inherited `constraints` (`require_levels`, `max_extra_levels`, `allowed_overlays` matched via `doublestar`). Violations become blocking `Conflict` entries (`kind: "constraint"`).
     4. Orders levels as declared in `gradus.yaml`; inserts each active overlay immediately after the level named by `strength`.
     5. Loads rules (`ParseRuleFile` with container-derived `level` per ADR 0004), drops `retired`, excludes `draft` unless `opts.IncludeDraft`.
     6. Loads `.gradus/exceptions/*.yaml`: marks matching rules as waived for their scopes if `expires >= today`; emits warnings for expired exceptions.
     7. Merges rules and enforces **P2 Restriction Invariant** + **ADR 0001**:
        - `overrides` targeting unknown ID or committed level targeting `user` rule -> error (exit code `2`).
        - Duplicate ID without `overrides` -> blocking conflict (`exit code 1`).
        - `overrides` lowering severity (`MUST > SHOULD > MAY`), widening `scope` (via `doublestar` / glob set check), weakening `enforcement` (`permission = hook > check > review > prompt`), or relaxing `guard.decision` (`deny > ask`) on a non-`overridable` rule -> blocking conflict. Between active overlays overriding the same ID, the more restrictive structural rule wins (D-56); otherwise a blocking conflict is recorded.
        - `overrides` targeting a `locked: true` level rule not listed in `editable_rules` -> blocking conflict.
     8. Loads confirmed semantic conflicts from `.gradus/findings/*.yaml` (`status: confirmed`) as blocking `Conflict` entries.
     9. Emits `ResolvedView` matching §9.7 (`schema`, `view_hash`, `lock_hash`, `scope`, `levels`, `overlays`, `profile`, `rules` with `text` and `provenance: {source, path, hash}`, `conflicts`, `exceptions`, and `budget` character count reporting Antigravity `12000` target limit).

4. **CLI Commands (`internal/cli/resolve.go`, `internal/cli/status.go`):**
   - `gradusctl resolve [--dir] [--write] [--json]`: Runs `resolve.Resolve`, writes `gradus.lock` on `--write` (when no errors occur), emits `ResolvedView` as JSON or human-readable summary. Exits `0` if clean, `1` if `len(Conflicts) > 0`, `2` on errors.
   - `gradusctl status [--dir] [--brief] [--json]`: Reads `gradus.yaml`, `gradus.lock`, `.gradus/findings/`, and `~/.gradus/lessons/<project-key>/` locally with zero network calls (`< 100ms`), reporting lock status, conflict count, and pending lesson count.

## Options Considered

- **Scope comparison for P2 (widening check):** Chose explicit glob containment heuristics backed by `github.com/bmatcuk/doublestar/v4` (exact match, `"**/*"` universal superset, prefix containment `a/b/**` inside `a/**`, and subset check of glob lists) over an SMT solver, keeping resolution fast (`< 10ms`) and dependency-light.
- **Git invocation:** Chose `os/exec` calling system `git` (D-69) over `go-git`, guaranteeing SSH agent, credential helpers, and `.gitconfig` compatibility with zero binary bloat.

## Simplicity And Elegance Review

- `internal/source`, `internal/lock`, and `internal/resolve` each have a single responsibility and clean Go interfaces, testable with `t.TempDir()` and local bare Git repos.
- `gradusctl status --brief` bypasses `git` network operations completely so session-start hooks stay well under `100ms`.

## Failure Modes And Tradeoffs

- **Unreachable Git remote:** Controlled explicitly per level (`offline.mode: cache` with `max_age` vs `fail-closed`), returning exit code `2` when `fail-closed` or expired.
- **Scope widening edge cases:** Complex intersecting globs that cannot be proven a strict subset are flagged conservatively as a potential widening conflict, letting a human owner approve or clarify.

## Verification Plan

- **`internal/source` tests:** Local path resolution, CI `~/` skip, bare Git repo clone/checkout via `os/exec`, offline cache hit within `max_age`, and offline `fail-closed` / expired cache error.
- **`internal/lock` tests:** Cross-platform CRLF vs LF identical `sha256:` hash, `gradus.lock` read/write excluding `user` level, and tampered locked level mismatch detection.
- **`internal/resolve` tests:** Overlay insertion by `strength`, `retired`/`draft` filtering, active/expired exceptions, inherited constraint violations, P2 restriction invariant (severity/scope/enforcement/`editable_rules`), ADR 0001 CI override rejection, and confirmed findings inclusion.
- **`internal/cli` tests:** `gradusctl resolve --write --json` (clean -> `0`, conflicts -> `1`, lock tamper -> `2`) and `gradusctl status --brief --json` latency/output check.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC4`: `internal/source` (`ResolveLevelSource`, `os/exec` git fetcher, `.gradus/cache/`, `offline` policy) |
| `R2` | `R2.AC1`–`R2.AC3`: `internal/lock` (`HashDirectory` CRLF->LF, `ReadLockfile`/`WriteLockfile`, locked integrity verification) |
| `R3` | `R3.AC1`–`R3.AC5`: `internal/resolve` (§10 pipeline, constraints, P2 restriction invariant, ADR 0001, exceptions, confirmed findings) |
| `R4` | `R4.AC1`–`R4.AC3`: `internal/cli/resolve.go` & `internal/cli/status.go` (`gradusctl resolve`, `gradusctl status --brief`) |
| `NFR1` | Zero-network local read path in `gradusctl status --brief` (`< 100ms`) |
| `NFR2` | `os/exec` system `git` subprocess in `internal/source` with `CGO_ENABLED=0` |
| `NFR3` | Exit codes `0` (clean), `1` (conflicts), `2` (lock/source/override errors) via `ExitErrorWithCode` |
