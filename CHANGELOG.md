# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.2] - 2026-11-21

### Fixed
- **Live Antigravity hooks integration:** Updated `.agents/hooks.json` compiler output to Antigravity's native top-level named hook schema (`gradus` with grouped `PreToolUse` matching `run_command|write_file|edit_file|write_to_file|replace_file_content|multi_replace_file_content` and flat `SessionStart` calling `gradusctl status --brief --antigravity-hook`).
- **Native Antigravity `PreToolUse` payload & workspace CWD resolution:** `gradusctl guard` now parses Antigravity camelCase `toolCall.name`/`toolCall.args` payloads alongside Claude Code `tool_name`/`tool_input` and resolves the repository root from `cwd`/`workspacePaths` (including when hooks run from `.agents/`).
- **Anti-tamper hardening (T2):** Protected `.claude/settings.local.json`, `.git/hooks/**`, and `.githooks/**`; expanded shell anti-tamper for `git config core.hooksPath`, `git checkout <ref> -- .gradus`, whitespace-free redirections (`echo x>gradus.yaml`), `find -delete`, `ln`, `unlink`, and `gradus.*` globs; clarified defense-in-depth boundary in `README.md`.
- **Redaction precision (T4):** Added PEM private key blocks, Italian Codice Fiscale, and `+39 333 1234567` mobile phone formats while eliminating false positives on plain space-separated numbers.
- **Conflict candidate noise reduction:** Expanded stopword filtering in `gradusctl conflicts --candidates` to reduce generic lexical false positives.
- **Version & authoring UX:** `gradusctl version` now falls back to `runtime/debug.ReadBuildInfo()` when installed via `go install`; `gradus-author/SKILL.md` now includes full YAML templates for `checker`, `guard`, and `level.yaml` (`charter`); `internal/source/git.go` rejects refs starting with `-`; `gradusctl resolve --write` reminds users to re-run `gradusctl compile`.

## [0.1.1] - 2026-11-20

### Security
- **Symlink bypass prevention (Threat Model T2):** `internal/lock.HashDirectory`, `internal/resolve.Resolve`, and `internal/guard` now fail closed (`exit 2`) on symlinks inside governed level/overlay directories.
- **Shell anti-tamper protection (Threat Model T2):** `gradusctl guard` now detects mutating or destructive shell commands targeting `gradus.yaml`, `gradus.lock`, `.gradus/`, `.walden/`, `.agents/hooks.json`, and `.claude/settings.json` and enforces an `ask` confirmation floor.
- **Git flag bypass hardening (`PRJ-GIT-001`):** Updated command guard pattern to catch `git` global flags before `commit`/`push` (e.g., `git -C . commit`, `git -c user.name=x commit`).
- **Expanded lesson redaction (Threat Model T4):** `internal/state/redact.go` now redacts key-value passwords/secrets, IBANs, and phone numbers in addition to API tokens and emails.

### Fixed
- `gradusctl status` now reports structural conflicts from offline resolution and counts pending JSON lessons from `state.LessonsDir`.
- `gradusctl resolve --write` now computes `view.lock_hash` from the written `gradus.lock` bytes.
- Added `--` argument separators to `git clone`/`checkout`/`diff` subprocess invocations in `internal/source` and `internal/check`.
- Clarified `gradus-learn` occurrence threshold (`>= 2`) and aligned `README.md` / `evals/benchmark/RESULTS.md` honestly with technical preview status (noting that live-agent `E9` and real-project `O-1` remain open).

## [0.1.0] - 2026-11-20

### Added
- **Core Schema & CLI (`F1`):** `spec/v1` JSON Schemas, forward-compatible YAML/frontmatter parser with ADR 0004 container-based level derivation, `gradusctl init`, `gradusctl explain`, and `gradusctl version [--json]`.
- **Pyramid Resolution & Lockfile (`F2`):** Local and `os/exec` Git source resolution, offline cache policies, normalized cross-platform SHA-256 `gradus.lock`, P2 restriction invariant (with ADR 0001), `gradusctl resolve`, and `gradusctl status [--brief]`.
- **Agent Compilers (`F3`):** Deterministic `gradusctl compile` generating `AGENTS.md`, `CLAUDE.md`, `.claude/rules/`, `.claude/settings.json`, `.agents/rules/` (`<11,500`-char chunking), `.agents/hooks.json`, and embedded skill extraction.
- **Verification, Checks, Evidence & Git Hooks (`F4`):** `gradusctl verify`, `gradusctl check [--staged | --base]`, 5 evidence sinks (`none`, `local`, `ci-artifact`, `repo`, `command`), and `gradusctl hooks` (`pre-commit`, `lefthook`, `husky`, `native`).
- **Runtime Guard (`F5`):** `<100ms` offline `PreToolUse` hook evaluator (`gradusctl guard`) for Claude Code and Antigravity with built-in anti-tamper protection (Threat Model T2) and `deny > ask > allow` precedence.
- **Lifecycle State (`F6`):** `gradusctl conflicts`, `gradusctl lessons` with automatic secret/PII redaction (Threat Model T4, P16), and `gradusctl reconcile` (`--restore`, `--move`).
- **Canonical MVP Skills & Evals (`F7`, `F8`):** Six embedded Agent Skills (`gradus-onboard`, `gradus-author`, `gradus-review`, `gradus-learn`, `gradus-reconcile`, `gradus-conflicts`), `evals/skills/` Skill-TDD suite, and `evals/benchmark/` comparative evaluation (`E1`–`E10`).
- **Supply-Chain Hardening (`T3`):** `.goreleaser.yaml` (`CGO_ENABLED=0` static binaries, `checksums.txt`, `syft` SBOM, keyless `cosign` signing) and SHA-pinned `.github/workflows/release.yml` with build provenance attestations.
