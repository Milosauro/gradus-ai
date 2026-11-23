---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T11:02:51Z
last_modified: 2026-11-23T11:02:51Z
approved_fingerprint: sha256:043898d3b8b9d19515e3709b17fb410a78b57ca34e27a9bae6d3d42715670c90
source_requirements_approved_at: 2026-11-23T11:00:37Z
source_requirements_fingerprint: sha256:b08039619feee2ca10b577a31230bc6c2dd5961ae73baa9bb70bc27d1d8edf35
---

# Feature Design

## Architecture

This feature extends four existing layers with minimal surface area and zero new third-party dependencies:

1. **Schema & SemVer (`internal/schema/types.go`, `internal/schema/validate_config.go`, `internal/schema/semver.go`, `spec/v1/schema/`):**
   - Add optional `Signature` (`"" | "optional" | "required"`) and `MinVersion` (SemVer string) fields to `schema.LevelRef` (`gradus.yaml`) and `schema.LevelConfig` (`level.yaml`), plus matching JSON Schema properties in `spec/v1/schema/gradus.schema.json` and `spec/v1/schema/level.schema.json`.
   - Implement a lightweight stdlib SemVer parser/comparator in `internal/schema/semver.go` (`ParseSemVer`, `CompareSemVer`) supporting optional `v` prefix (`v1.2.3` / `1.2.3`), `major.minor.patch`, and optional prerelease ordering.
2. **Git Signature Verification (`internal/source/signature.go`):**
   - Add `VerifyGitSignature(repoDir, ref string) (string, error)` shelling out to system `git` via `os/exec` (`PRJ-GO-001`):
     - Rejects empty or `-`-prefixed refs (`NFR2`).
     - Checks object type via `git cat-file -t -- <ref>`: if `"tag"`, runs `git verify-tag -- <ref>`; otherwise runs `git verify-commit -- HEAD` (since the cached repo is already checked out at `ref`).
     - Returns `"verified"` on exit 0, or `"none"` with the git stderr diagnostic on failure.
   - Supports any Git-native signing backend configured in the user/CI environment (GPG or SSH `allowedSignersFile`).
3. **Resolution & Verification Integration (`internal/resolve/resolve.go`):**
   - After resolving a level source and loading `level.yaml`, compute the effective signature policy (`required` if either `lvl.Signature` or `lc.Signature` is `required`, else `optional`) and effective `min_version` (the stricter/higher SemVer if both `lvl.MinVersion` and `lc.MinVersion` are declared).
   - For git-backed levels:
     - When `opts.OfflineOnly` is true (latency-sensitive `gradusctl guard`) and `lockEntry.Signature == "verified"`, trust the lockfile record to stay within the `< 100ms` latency budget; otherwise run `source.VerifyGitSignature`.
     - If effective policy is `required` and verification fails, fail resolution/verification with an error (mapped to CLI exit `2`).
     - Record `Signature: "verified"` or `"none"` in the emitted `lock.Entry` (reusing the existing `lock.Entry.Signature` field).
   - If effective `min_version` is non-empty, resolve the level's version from `lc.Version` (falling back to `resSrc.Ref`), parse as SemVer, and fail with an error (exit `2`) if missing, invalid, or `< min_version`.
4. **Readable Level Changelog (`internal/changelog/changelog.go`, `internal/cli/changelog.go`):**
   - `changelog.DiffLevel(repoDir, levelID, fromRef, toRef string) (Report, error)`:
     - Validates refs do not start with `-`.
     - Materializes `fromRef` and `toRef` into two temporary directories via `git archive --format=tar <ref>` piped to Go's `archive/tar` (non-destructive, cross-platform, no external `tar` binary).
     - Loads `level.yaml` and `rules/*.md` from both directories using `schema.ParseLevelConfig` and `schema.LoadRulesDir`.
     - Computes deterministic, ID-sorted `Added`, `Removed`, `Modified` (tracking field-level changes in `title`, `severity`, `enforcement`, `scope`, `status`, `text`), and `MetadataChanges` (`version`, `min_version`, `charter`).
     - Renders Markdown via `FormatMarkdown(Report) string`.
   - CLI `gradusctl changelog <level-id> [--dir .] [--from <ref>] [--to <ref>] [--json]`:
     - Looks up `<level-id>` in `gradus.yaml`, requires a git-backed source, defaults `--from` to `lockEntry.Commit` in `gradus.lock` and `--to` to `lvl.Source.Ref` (or `HEAD`), ensures the cached repo under `.gradus/cache/<id>/repo` has both refs, and prints Markdown or JSON with exit `0` (or exit `2` on invalid level/ref/error).

## Options Considered

- **Chosen:** Use `os/exec` `git verify-tag` / `git verify-commit` for signatures and `git archive --format=tar` + stdlib `archive/tar` for changelog materialization.
  - Works with both GPG and SSH Git signatures using existing developer/CI trust stores, respects `PRJ-GO-001` (`CGO_ENABLED=0`), and never mutates the repository working tree or cached checkout.
- **Alternative:** Embed Go crypto/OpenPGP libraries or `go-git`, or perform `git checkout` in the cached repo during `changelog`.
  - Rejected: `go-git` violates `PRJ-GO-001`; embedding OpenPGP duplicates system git trust configuration; checking out refs in-place during `changelog` would mutate `.gradus/cache/<id>/repo` and risk breaking concurrent `verify`/`guard` calls.

## Simplicity And Elegance Review

- Reuses the already-reserved `lock.Entry.Signature` field (`none` | `verified`) rather than changing the lockfile schema.
- Keeps SemVer parsing to ~50 lines of stdlib Go (`internal/schema/semver.go`) instead of pulling in an external dependency.
- `git archive` + `archive/tar` extracts snapshots into ephemeral `os.MkdirTemp` folders cleaned via `defer os.RemoveAll`, guaranteeing zero side effects on both the workspace and `.gradus/cache/`.

## Failure Modes And Tradeoffs

- **Missing local trust store / unsigned commits in CI:** When `signature: required` is enabled, CI runners must configure their GPG keyring or `gpg.ssh.allowedSignersFile`; failure fails closed with exit `2` and includes git's stderr so operators can diagnose key trust issues immediately.
- **Annotated vs lightweight tags:** Lightweight tags are commit references (`git cat-file -t` returns `commit`), so `VerifyGitSignature` falls back cleanly to `git verify-commit`, supporting signed commits pointed to by lightweight tags as well as signed annotated tags.
- **Unreachable remote during `changelog`:** If both `--from` and `--to` commits already exist in `.gradus/cache/<id>/repo`, `changelog` succeeds completely offline; only if a ref is missing does it attempt a `git fetch --tags --force origin` in the cache repo.

## Verification Plan

- Unit tests in `internal/schema/semver_test.go` covering SemVer parsing/comparison, invalid `min_version`, and invalid `signature` values (`R1.AC4`, `R2.AC3`).
- Integration tests in `internal/source/signature_test.go` and `internal/resolve/` using temporary local Git repositories with SSH signing (`ssh-keygen` + `gpg.format=ssh` + `gpg.ssh.allowedSignersFile`) to verify signed tags, signed commits, unsigned refs with `required` vs `optional`, and `min_version` pass/fail (`R1.AC1`–`R1.AC3`, `R2.AC1`–`R2.AC2`, `NFR1`, `NFR2`).
- Unit and CLI tests in `internal/changelog/changelog_test.go` and `internal/cli/changelog_test.go` exercising `gradusctl changelog` with explicit `--from`/`--to`, default lock/config refs, added/modified/removed rules, metadata changes, invalid level/ref (`-evil`), JSON and Markdown output, and clean `git status` (`R3.AC1`–`R3.AC3`, `NFR3`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/schema/types.go`, `internal/schema/validate_config.go`, `internal/source/signature.go`, `internal/resolve/resolve.go` (`R1.AC1`–`R1.AC4`) |
| `R2` | `internal/schema/semver.go`, `internal/schema/validate_config.go`, `internal/resolve/resolve.go` (`R2.AC1`–`R2.AC3`) |
| `R3` | `internal/changelog/changelog.go`, `internal/cli/changelog.go` (`R3.AC1`–`R3.AC3`) |
| `NFR1` | `os/exec` `git` calls only in `internal/source/signature.go` and `internal/changelog/changelog.go`; verified by `CGO_ENABLED=0` build |
| `NFR2` | Leading `-` ref rejection and `--` separators in `internal/source/signature.go` and `internal/changelog/changelog.go` |
| `NFR3` | Sorted rule IDs and `os.MkdirTemp` + `git archive` extraction in `internal/changelog/changelog.go` |
| `C1` | Backward-compatible optional fields in `gradus/v1` Go types and `spec/v1/schema/*.json` |
| `C2` | Reuses `internal/schema`, `internal/source`, `internal/lock`, `internal/resolve` with stdlib only |
