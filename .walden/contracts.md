---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T10:48:34Z
last_modified: 2026-11-23T10:48:34Z
approved_fingerprint: sha256:3e7414ec0bc9beecb037dc7f1cb153f1a25839ede45e71ab5bfd105f78363166
---

# Current Contracts

## core-schema-init

Purpose: Provides the `gradus/v1` specification, JSON Schemas, forward-compatible YAML/Markdown parsers with ADR 0004 level derivation, and `gradusctl init`/`explain`.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R2.AC4, R3.AC1, R3.AC2, R3.AC3, R3.AC4, R4.AC1, R4.AC2, R4.AC3, R5.AC1, R5.AC2, R5.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0004-rule-id-and-level-derivation.md
Related: resolve-and-lock (consumes internal/schema configuration and rule parsers)

## resolve-and-lock

Purpose: Resolves local and `os/exec` Git pyramid sources, verifies normalized `gradus.lock` integrity, enforces §10 constraints and the P2 restriction invariant (with ADR 0001), and exposes `gradusctl resolve`/`status`.
Active: R1.AC1, R1.AC2, R1.AC3, R1.AC4, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, R3.AC4, R3.AC5, R4.AC1, R4.AC2, R4.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0001-level-ordering-and-ci-overrides.md, docs/adr/0004-rule-id-and-level-derivation.md
Related: core-schema-init (builds on internal/schema parsers and ExitErrorWithCode semantics), agent-compilers (provides ResolvedView input), verify-check-evidence (provides ResolvedView and lock verification), runtime-guard (provides OfflineOnly local resolution)

## agent-compilers

Purpose: Deterministically compiles the resolved pyramid view into managed Markdown sections (`AGENTS.md`, `CLAUDE.md`), scoped rules, Antigravity `<11,500`-char chunked rules, static hook dispatchers (`.claude/settings.json`, `.agents/hooks.json`), and embedded skills (`gradusctl compile`).
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, R4.AC1, R4.AC2, R4.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0002-cumulative-enforcement-semantics.md, docs/adr/0003-static-guard-dispatcher.md
Related: resolve-and-lock (consumes ResolvedView), verify-check-evidence (reuses PlanCompile for drift detection), runtime-guard (emits static PreToolUse hook dispatchers in .claude/settings.json and .agents/hooks.json)

## verify-check-evidence

Purpose: Provides drift and lock verification (`gradusctl verify`), scoped deterministic checks via built-in regex and external checkers (`gradusctl check`), privacy-safe multi-sink evidence recording (`gradusctl evidence`), and git hook manager integration (`gradusctl hooks`).
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R2.AC3, R2.AC4, R3.AC1, R3.AC2, R3.AC3, R4.AC1, R4.AC2, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: resolve-and-lock (consumes ResolvedView and lock integrity), agent-compilers (reuses PlanCompile for byte-for-byte drift checks)

## runtime-guard

Purpose: Evaluates Claude Code and Antigravity `PreToolUse` JSON payloads locally in `<100ms` without network I/O, enforcing built-in anti-tamper protection on governance files and active project/user `enforcement: hook` rules with `deny > ask > allow` precedence (`gradusctl guard`).
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R3.AC1, R3.AC2, R3.AC3, R4.AC1, R4.AC2, NFR1, NFR2, NFR3, C1, C2, C3
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0003-static-guard-dispatcher.md, docs/threat-model.md
Related: resolve-and-lock (extends resolve.Options with OfflineOnly cache-only path), agent-compilers (consumes static PreToolUse dispatchers and protects .claude/settings.json and .agents/hooks.json via anti-tamper)

## cli-lifecycle-state

Purpose: Manages semantic conflict candidate discovery and hash-indexed findings (`gradusctl conflicts`), local secret/PII-redacted lessons (`gradusctl lessons`), and deterministic drift reconciliation (`gradusctl reconcile`).
Active: R1.AC1, R1.AC2, R1.AC3, R1.AC4, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, R3.AC4, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/threat-model.md
Related: resolve-and-lock (reads/writes .gradus/findings/*.yaml consumed by resolve), agent-compilers (reuses PlanCompile/ApplyCompile for reconcile), mvp-skills (provides CLI backing state for conflicts, learn, and reconcile skills)

## mvp-skills

Purpose: Authors and embeds the six canonical MVP Agent Skills (`gradus-{onboard,author,review,learn,reconcile,conflicts}`) via `skills/embed.go`, removes the bootstrap placeholder, and establishes the §23 Skill-TDD regression suite under `evals/skills/`.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: agent-compilers (extracted by compile into .claude/skills/ and .agents/skills/), cli-lifecycle-state (skills delegate state mutations strictly to gradusctl conflicts, lessons, and reconcile)

## phase0-eval-release

Purpose: Establishes the offline deterministic benchmark suite (`evals/benchmark/`), `README.md`/`CONTRIBUTING.md` onboarding docs, `.goreleaser.yaml`, SHA-pinned `.github/workflows/release.yml`, and `CHANGELOG.md` for `v0.1.0` readiness.
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R3.AC1, R3.AC2, R3.AC3, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/threat-model.md
Related: agent-compilers (exercises compile idempotency E1), resolve-and-lock (exercises resolution and conflict benchmarks), runtime-guard (exercises anti-tamper E2)

## charters-and-lifecycle

Purpose: Enforces level `charter.allowed_severities` and `context_budget` (`max_rules`, `max_chars`), partitions `draft`/`retired` rules into `InactiveRules`, validates `review_by`/`expires` dates with deprecation/expiry diagnostics, and exposes charters in `resolve`, `explain`, and authoring/review skills.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R3.AC1, R3.AC2, R3.AC3, R4.AC1, R4.AC2, R4.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: core-schema-init (extends level and rule schemas), resolve-and-lock (enforces charter/budget/lifecycle during Resolve), mvp-skills (exposes charters in gradus-author and gradus-review)

## overlays-and-exceptions

Purpose: Supports explicit overlay `activation` modes (`explicit | always | disabled | profile`), enforces `exceptions.max_duration`, approval state, and `break-glass` policies (`post_review_due`, `break_glass: false`, `fail-closed`), and provides `gradusctl exception request|approve|break-glass|list|expire`.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, R3.AC4, NFR1, NFR2, C1
Reserved: none
Sources: docs/design/gradus-design.md, docs/threat-model.md
Related: core-schema-init (extends overlay and exception schemas), resolve-and-lock (evaluates overlay activation and exception waivers during Resolve)

## doctor-and-guard-hardening

Purpose: Adds `gradusctl doctor [--json]` environment/hook diagnostics, configurable `anti_tamper.mode` (`ask | deny`) for hard-blocking governance edits even in Claude Code `auto` mode, `evidence prune`/`lessons prune` hygiene commands, and `user_rules` visibility in `status --json`.
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, NFR1, NFR2, C1
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0003-static-guard-dispatcher.md, docs/threat-model.md
Related: agent-compilers (validates .claude/settings.json and .agents/hooks.json emitted by compile), runtime-guard (extends anti-tamper evaluation with configurable mode: ask|deny)

## rule-tests-and-impact

Purpose: Provides `gradusctl test [--rule <id>] [--json]` to verify `### Compliant` and `### Non-compliant` examples for `check` and `hook` rules, and non-destructive `gradusctl simulate [--commits N | --since ref] [--json]` over git history via `os/exec` `git`.
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R2.AC3, NFR1, NFR2, C1
Reserved: none
Sources: docs/design/gradus-design.md
Related: verify-check-evidence (reuses internal/check for example and commit scanning), runtime-guard (reuses internal/guard.Evaluate for hook rule examples)

## github-action-and-ci-agents

Purpose: Provides a SHA-pinned composite GitHub Action (`.github/actions/gradus/action.yml`) and `--agent-profile` / `GRADUS_AGENT_PROFILE` support on `resolve`, `compile`, and `guard` to replace personal `user` rules in CI and background agents.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, NFR1, NFR2, C1
Reserved: none
Sources: docs/design/gradus-design.md, docs/threat-model.md
Related: resolve-and-lock (extends Resolve with AgentProfile), agent-compilers (propagates --agent-profile to compile), runtime-guard (propagates --agent-profile to guard)

## handbook-and-templates

Purpose: Provides deterministic `gradusctl handbook [--output <file>] [--json]` rendering the resolved pyramid hierarchy, level charters, and active rules, plus starter templates under `examples/personal/`, `examples/team/`, and `examples/regulated-fintech/`.
Active: R1.AC1, R1.AC2, R2.AC1, NFR1, NFR2, C1
Reserved: none
Sources: docs/design/gradus-design.md
Related: charters-and-lifecycle (renders level charters in handbook), resolve-and-lock (consumes ResolvedView)

## Coherence Review

### charters-and-lifecycle
Consistent with `core-schema-init#R1.AC1`, `resolve-and-lock#R3.AC1`, and `mvp-skills#R2.AC1`. `charters-and-lifecycle` extends level/rule frontmatter and `resolve.Resolve` without altering existing exit-code or lock semantics, and surfaces charters in `gradus-author` and `gradus-review`. No inconsistencies found.

### doctor-and-guard-hardening
Consistent with `agent-compilers#R2.AC2`, `agent-compilers#R3.AC3`, `runtime-guard#R2.AC1`, `runtime-guard#R2.AC2`, `runtime-guard#R3.AC3`, and `runtime-guard#C3`. `doctor-and-guard-hardening#R1.AC2` validates the exact `.claude/settings.json` and `.agents/hooks.json` static hook dispatchers generated by `agent-compilers`, and `doctor-and-guard-hardening#R2.AC1`–`R2.AC3` refine anti-tamper decision strictness (`ask` default, optional `deny`) while preserving `runtime-guard` `deny > ask > allow` precedence and ADR 0003 static dispatchers. No inconsistencies found.

### github-action-and-ci-agents
Consistent with `resolve-and-lock#R4.AC1`, `agent-compilers#R4.AC1`, and `runtime-guard#R3.AC1`. `--agent-profile` cleanly substitutes the optional `user` level in CI/background agents without weakening upper levels or lock verification, and `.github/actions/gradus/action.yml` pins all actions to 40-char SHAs (T3). No inconsistencies found.

### handbook-and-templates
Consistent with `charters-and-lifecycle#R1.AC1` and `resolve-and-lock#R4.AC1`. `gradusctl handbook` deterministically renders the resolved pyramid and charters read-only unless `--output` is passed, and all starter templates under `examples/` pass `resolve` and `verify`. No inconsistencies found.

### overlays-and-exceptions
Consistent with `core-schema-init#R1.AC1` and `resolve-and-lock#R3.AC1`. Explicit overlay `activation` modes and `exceptions` (`max_duration`, `break-glass`, `fail-closed`) extend `resolve.Resolve` while preserving the P2 restriction invariant and deterministic `gradus.lock` verification. No inconsistencies found.

### phase0-eval-release
Consistent with `agent-compilers#R4.AC1`, `runtime-guard#R2.AC1`, and `verify-check-evidence#R1.AC1`. The offline benchmark suite and release workflows verify existing Phase 0 guarantees (`E1`–`E10`, `T3`) without modifying runtime contracts. No inconsistencies found.

### rule-tests-and-impact
Consistent with `verify-check-evidence#R2.AC1` and `runtime-guard#R3.AC2`. `gradusctl test` and `gradusctl simulate` reuse `internal/check` and `internal/guard.Evaluate` non-destructively via temporary directories and `os/exec` `git` (`PRJ-GO-001`). No inconsistencies found.
