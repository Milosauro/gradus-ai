---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T16:15:52Z
last_modified: 2026-11-23T16:15:52Z
approved_fingerprint: sha256:adbba152ec55f0c23e803fc2ed4518f8ae0544ed7810b1ea27168599d0cf1289
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

## remote-levels-signatures-and-changelog

Purpose: Adds `os/exec` Git signature verification (`signature: optional | required` recorded in `gradus.lock`), strict SemVer `min_version` enforcement for remote levels, and non-destructive `gradusctl changelog` diffs via `git archive` + `archive/tar`.
Active: R1.AC1, R1.AC2, R1.AC3, R1.AC4, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: core-schema-init (adds SemVer and signature/min_version fields), resolve-and-lock (enforces signature and min_version during Resolve)

## obligations-and-bindings

Purpose: Adds abstract `obligations` and concrete `bindings` schemas (`gradus.yaml`, `level.yaml`), enforces `allowed_providers` narrowing and binding coverage in `resolve.Resolve`, and renders an `Obligations & Bindings` section in `gradusctl handbook`.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: core-schema-init (extends config schemas), resolve-and-lock (emits obligation conflicts and ResolvedView fields), handbook-and-templates (renders Obligations & Bindings)

## adapters-walden-and-speckit

Purpose: Embeds official declarative `walden` and `spec-kit` adapter manifests (`adapters/`), loads custom `.gradus/adapters/` manifests with path-containment validation, and provides `gradusctl adapter list | sync | verify` for managed-section injection, CLI SemVer checks, and evidence verification.
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R2.AC3, R2.AC4, R3.AC1, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: obligations-and-bindings (consumes ResolvedView.Bindings), agent-compilers (reuses PlanAgentsMD and WriteIfChanged)

## toolchain-governance

Purpose: Declares `required`, `recommended`, and `forbidden` skills, tools, MCP servers, and plugins in `gradus.yaml`/`level.yaml`, enforces monotonic pyramid resolution via `doublestar` glob matching, and provides offline `gradusctl toolchain check [--json]`.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R3.AC1, R3.AC2, R3.AC3, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: core-schema-init (extends config schemas), resolve-and-lock (emits toolchain conflicts), agent-compilers (inspects .claude/skills and .agents/skills)

## gradus-link-skill

Purpose: Authors and embeds the seventh official Agent Skill (`skills/gradus-link/SKILL.md`) for auditing third-party skills, tools, MCP servers, and workflow providers against the pyramid and drafting declarative adapter manifests or binding proposals with explicit approval, backed by Skill-TDD scenario `SKILL-07`.
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: mvp-skills (extends embedded skills and Skill-TDD suite), adapters-walden-and-speckit (drafts .gradus/adapters/ manifests), toolchain-governance (audits against toolchain.forbidden)

## codex-copilot-opencode-targets

Purpose: Adds deterministic `codex`, `copilot` (`.github/copilot-instructions.md` + `.github/instructions/gradus-*.instructions.md` with pruning), and `opencode` compilers to `internal/compile` alongside `docs/compatibility-matrix.md`.
Active: R1.AC1, R2.AC1, R2.AC2, R3.AC1, R4.AC1, R4.AC2, R4.AC3, R4.AC4, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md
Related: agent-compilers (extends PlanCompile/ApplyCompile and reuses managed-section helpers), verify-check-evidence (automatic drift detection via PlanCompile)

## Coherence Review

### adapters-walden-and-speckit
Consistent with `obligations-and-bindings#R2.AC1`, `core-schema-init#R1.AC1`, `remote-levels-signatures-and-changelog#R2.AC1`, `toolchain-governance#R1.AC1`, `gradus-link-skill#R1.AC2`, and `codex-copilot-opencode-targets#R4.AC1`. `adapters-walden-and-speckit` consumes `ResolvedView.Bindings` and reuses `compile.PlanAgentsMD` and `schema.ParseSemVer` without altering existing schemas or exit codes. No inconsistencies found.

### codex-copilot-opencode-targets
Consistent with `core-schema-init#R1.AC1`, `adapters-walden-and-speckit#R2.AC1`, `gradus-link-skill#R2.AC1`, `obligations-and-bindings#R3.AC1`, `remote-levels-signatures-and-changelog#R1.AC1`, and `toolchain-governance#R3.AC1`. `codex`, `copilot`, and `opencode` extend `compile.PlanCompile` and `ApplyCompile` while preserving byte-identical output for `claude-code`, `antigravity`, and `agents-md`. No inconsistencies found.

### gradus-link-skill
Consistent with `mvp-skills#R1.AC1`, `agent-compilers#R2.AC3`, `agent-compilers#R3.AC3`, `adapters-walden-and-speckit#R1.AC2`, `obligations-and-bindings#R2.AC1`, and `toolchain-governance#R3.AC1`. `skills/gradus-link/SKILL.md` embeds cleanly via `skills/embed.go` and delegates all verification to `gradusctl`. No inconsistencies found.

### obligations-and-bindings
Consistent with `core-schema-init#R1.AC1`, `doctor-and-guard-hardening#R1.AC1`, `adapters-walden-and-speckit#R2.AC1`, and `toolchain-governance#R2.AC1`. `obligations` and `bindings` use `omitempty` JSON fields in `ResolvedView`, preserving `view_hash` for existing repositories without obligations. No inconsistencies found.

### remote-levels-signatures-and-changelog
Consistent with `core-schema-init#R1.AC1`, `doctor-and-guard-hardening#R2.AC1`, `obligations-and-bindings#R1.AC1`, and `adapters-walden-and-speckit#R2.AC2`. Reuses `lock.Entry.Signature` and `os/exec` `git` (`PRJ-GO-001`) with non-destructive `git archive` materialization. No inconsistencies found.

### toolchain-governance
Consistent with `core-schema-init#R1.AC1`, `agent-compilers#R2.AC2`, `runtime-guard#C3`, `doctor-and-guard-hardening#R1.AC2`, `gradus-link-skill#R1.AC2`, and `obligations-and-bindings#R2.AC1`. `ResolvedView.Toolchain` is omitted when unused and enforces monotonic restriction via `doublestar` matching. No inconsistencies found.
