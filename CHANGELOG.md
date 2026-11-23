# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-11-23

### Added
- **Remote Level Signatures, `min_version` & `gradusctl changelog` (`F15`):** Verify GPG/SSH signatures via `os/exec` `git verify-tag`/`verify-commit` (`signature: optional | required` recorded in `gradus.lock`), enforce strict SemVer 2.0.0 `min_version` constraints, and generate non-destructive Markdown/JSON level diffs via `gradusctl changelog <level-id>`.
- **Obligations & Bindings (`F16`):** Declare abstract `obligations` (`allowed_providers`, `scope`) and concrete `bindings` (`provider`, `requires`, `scope`) in `gradus.yaml`/`level.yaml`, enforce narrowing and binding coverage in `resolve`/`verify`, and render an `Obligations & Bindings` section in `gradusctl handbook`.
- **Declarative Provider Adapters (`F17`):** Embed official `walden` (`.walden/constitution.md`) and `spec-kit` (`.specify/memory/constitution.md`) adapters under `adapters/`, load custom `.gradus/adapters/` manifests with strict path containment, and add `gradusctl adapter list | sync | verify`.
- **Monotonic Toolchain Governance (`F18`):** Declare `required`, `recommended`, and `forbidden` skills, tools, MCP servers, and plugins in `gradus.yaml`/`level.yaml` with `doublestar` conflict detection in `resolve`, plus offline `gradusctl toolchain check [--json]`.
- **`gradus-link` Agent Skill (`F19`):** Add and embed `skills/gradus-link/SKILL.md` (with Skill-TDD scenario `SKILL-07`) for compatibility/security auditing of third-party tools and drafting declarative adapter manifests or bindings with explicit approval.
- **Codex, GitHub Copilot & OpenCode Targets (`F20`):** Add `codex` (`AGENTS.md` + `.agents/skills/`), `copilot` (`.github/copilot-instructions.md`, scoped `.github/instructions/gradus-*.instructions.md` with pruning, `.github/skills/`), and `opencode` (`AGENTS.md` + `.opencode/skills/`) compilers plus `docs/compatibility-matrix.md`.

## [0.2.2] - 2026-11-23

### Fixed
- **Non-failing `SessionStart` hook for Claude Code (`gradusctl status --brief --hook`):** Added `--hook` to `gradusctl status` and updated the Claude Code compiler to emit `gradusctl status --brief --hook` (upgrading legacy `gradusctl status --brief` entries in place), so Claude Code injects the `SessionStart` banner cleanly even when `drift > 0` or `conflicts > 0`.
- **Scoped rules pointer in `CLAUDE.md`:** Added a summary note in compiled `CLAUDE.md` listing path-scoped rules compiled into `.claude/rules/gradus-*.md` so readers inspecting `CLAUDE.md` directly see that scoped rules are active.

## [0.2.1] - 2026-11-22

### Fixed
- **Accurate drift reporting in `gradusctl status` and `SessionStart`:** `gradusctl status` now computes compiled target drift (`drift`) and live `view_hash` (when compiled targets exist on disk) so `status --brief` and `SessionStart` stay consistent with `gradusctl verify`, while suppressing stderr `Error:` text when `--json` is enabled.
- **Upgrade drift check in `gradusctl doctor`:** Added a `compiled_drift` diagnostic to `gradusctl doctor` so out-of-date compiled files after upgrading `gradusctl` are surfaced with a `gradusctl compile` fix hint.
- **Clearer empty output in `gradusctl test`:** Emit `(0 passed, 0 failed — no rule examples found)` in text output when no rules declare `### Compliant` / `### Non-compliant` examples.

## [0.2.0] - 2026-11-21

### Added
- **Level Charters, Rule Lifecycle & Context Budgets (`F9`):** Enforce `charter.allowed_severities` and optional `context_budget` (`max_rules`, `max_chars`) during `resolve` and `verify`; partition `draft`/`retired` rules into `ResolvedView.InactiveRules`; validate `YYYY-MM-DD` `review_by`/`expires` dates; emit non-fatal warnings for `deprecated` and overdue `review_by` rules and fail on expired rules; expose level charters in `resolve --json`, `explain`, `gradus-author`, and `gradus-review`.
- **Explicit Overlays, Break-Glass & `gradusctl exception` (`F10`):** Support `activation: explicit | always | disabled | profile` on overlays; enforce `exceptions.max_duration`, pending unapproved exceptions, and `kind: break-glass` policies (`post_review_due`, `break_glass: false`, `fail-closed`); add `gradusctl exception request | approve | break-glass | list | expire`.
- **`gradusctl doctor`, Strict Anti-Tamper & Hygiene (`F11`):** Add `gradusctl doctor [--json]` diagnosing `gradus.yaml`, `gradus.lock`, `PATH`, `.claude/settings.json` hook types/guard registration, and `.agents/hooks.json`; add configurable `anti_tamper.mode` (`ask` | `deny`) for hard-blocking governance edits even in Claude Code `auto` mode; add `gradusctl evidence prune`, `gradusctl lessons prune`, and `user_rules` in `gradusctl status --json`.
- **Rule Testing & Impact Simulation (`F12`):** Add `gradusctl test [--rule <id>]` running `### Compliant` and `### Non-compliant` examples for `check` and `hook` rules, and `gradusctl simulate [--commits N | --since ref]` scanning git history non-destructively via `os/exec` `git`.
- **Composite GitHub Action & Team Agent Profiles (`F13`):** Add SHA-pinned composite GitHub Action `.github/actions/gradus/action.yml` and `--agent-profile` / `GRADUS_AGENT_PROFILE` support on `resolve`, `compile`, and `guard` replacing personal `user` rules in CI/background agents.
- **Onboarding Handbook & Starter Templates (`F14`):** Add deterministic `gradusctl handbook [--output <file>] [--json]` and non-prescriptive starter templates under `examples/personal/`, `examples/team/`, and `examples/regulated-fintech/`.

## [0.1.4] - 2026-11-21

### Fixed
- **Native Claude Code `PreToolUse` JSON schema compliance:** `gradusctl guard` now detects native Claude Code payloads (`hook_event_name` / `session_id`) and emits `{}` on `allow` (preserving normal Claude Code permission prompts) or strict `{"hookSpecificOutput": ...}` without top-level `decision`, `matched_rules`, or `anti_tamper` fields on `ask`/`deny` with exit code `0`, preventing Claude Code's strict hook JSON validator from rejecting Gradus guard responses.

## [0.1.3] - 2026-11-21

### Fixed
- **Antigravity `force_ask` hook decision:** `gradusctl guard` now maps `ask` decisions to `force_ask` for native Antigravity `toolCall` payloads so cached "Always Allow" tool approvals do not silently bypass Gradus confirmation guards.
- **Clean `--json` output on violations:** `gradusctl check --json`, `verify --json`, `resolve --json`, and `reconcile --json` no longer emit trailing `Error: ...` text to `stderr` on exit code `1` (`ExitViolation`), keeping combined `2>&1` output valid JSON.
- **Session reload & enforcement documentation (`README.md`):** Documented restarting/opening a new agent session after `gradusctl compile` so `.agents/hooks.json` and `.claude/settings.json` hooks take effect, and clarified `hook` vs `check` vs `review` enforcement timing.

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
