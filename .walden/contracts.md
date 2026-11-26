---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-26T13:57:03Z
last_modified: 2026-11-26T13:57:03Z
approved_fingerprint: sha256:9cfbf31a3a2cfd267aa01b1ac2a973cd3c513d00755329147e242b5615f19c36
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

## security-hardening-and-lock-upgrades

Purpose: Hardens `gradusctl` for `v0.3.1` with D-80 `view_hash` excluding personal `user` rules, anti-tamper coverage for `.codex/config.toml` and `opencode.json`, `resolve --upgrade-lock`, and supply-chain/YAML hygiene.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R4.AC1, R4.AC2, R4.AC3, R4.AC4, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/threat-model.md
Related: core-schema-init (YAML parsing), resolve-and-lock (D-80 view_hash and lock upgrades), github-action-and-ci-agents (action.yml hardening), runtime-guard (anti-tamper targets)

## action-hardening-and-design-v1-1

Purpose: Hardens the GitHub Action for non-Go consumer repos (`v0.3.2`), expands git-hook anti-tamper (`core.hooksPath`, `git push --no-verify`, env vars, `lefthook-local.yml`), validates `exception request --expires` early, handles binary/large files in `check`, closes redaction gaps, and aligns Design 1.1.
Active: R1.AC1, R1.AC2, R1.AC3, R1.AC4, R2.AC1, R3.AC1, R3.AC2, R3.AC3, R4.AC1, R4.AC2
Reserved: none
Sources: docs/design/gradus-design.md, spec/v1/SPEC.md, docs/threat-model.md
Related: github-action-and-ci-agents (updates .github/actions/gradus/action.yml), runtime-guard (anti-tamper bypasses), overlays-and-exceptions (early duration check), security-hardening-and-lock-upgrades (follows v0.3.1 hardening)

## conditional-overlays-by-profile

Purpose: Activates profile-driven conditional overlays (`activation: profile`, `when` predicates, `activation_rules`, extensible `profile` attributes) with `OverlaySummary.Reason` and blocking `overlay` conflicts for missing/disabled required overlays.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R3.AC1
Reserved: none
Sources: docs/design/gradus-design.md, spec/v1/SPEC.md
Related: overlays-and-exceptions (extends overlay activation modes), resolve-and-lock (evaluates profile rules during Resolve)

## framework-mapping-and-audit-export

Purpose: Propagates `controls:` monotonically across rule overrides in `ResolvedView` and provides `gradusctl audit coverage` and `gradusctl audit export` (`--format json|markdown`, `--framework`, `--output`).
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2
Reserved: none
Sources: docs/design/gradus-design.md
Related: core-schema-init (parses controls field), resolve-and-lock (propagates controls monotonically)

## model-policy-and-tiers

Purpose: Activates `models:` capability tiers (`tier-high`, `tier-standard`, `tier-light`) in `gradus.yaml`/`level.yaml`, enforces monotonic tier resolution (`model_policy` conflicts), emits Claude skill `model:` frontmatter, records `--skill`/`--model-tier` in evidence, and verifies tiers via `gradusctl verify --check-model-tier`.
Active: R1.AC1, R1.AC2, R2.AC1, R3.AC1, R3.AC2
Reserved: none
Sources: docs/design/gradus-design.md, spec/v1/SPEC.md
Related: agent-compilers (emits skill model frontmatter), verify-check-evidence (adds --check-model-tier and evidence fields)

## managed-settings-and-chat-bundles

Purpose: Adds `gradusctl compile --managed-settings <path>` for enterprise Claude Code `managed-settings.json` rollout and `gradusctl compile --chat-bundle <path.zip>` for byte-reproducible shared-rule and skill ZIP bundles excluding `user` rules.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2
Reserved: none
Sources: docs/design/gradus-design.md, docs/compatibility-matrix.md
Related: agent-compilers (extends compile outputs), toolchain-governance (exports deniedMcpServers from toolchain.forbidden)

## mcp-server

Purpose: Exposes a zero-dependency JSON-RPC 2.0 stdio MCP server (`gradusctl serve --mcp`) providing five read-only governance tools (`resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, `get_toolchain`).
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R3.AC1
Reserved: none
Sources: docs/design/gradus-design.md, docs/compatibility-matrix.md
Related: resolve-and-lock (serves resolved rules and explanations), verify-check-evidence (in-memory check_text), toolchain-governance (serves get_toolchain)

## maturity-assessment-and-commit-trailers

Purpose: Provides process-only governance maturity assessment (`gradusctl assess`, `L1`–`L5` across 5 dimensions per P16), embedded `gradus-assess` skill (`SKILL-08`), and optional git commit attribution trailers (`gradusctl trailers generate|verify`).
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R3.AC1
Reserved: none
Sources: docs/design/gradus-design.md
Related: mvp-skills (adds eighth embedded skill gradus-assess), resolve-and-lock (uses lock/view hashes in trailers)

## enterprise-hardening-and-design-v1-2

Purpose: Makes `guard` and `status --hook` allow/exit `0` outside Gradus repos (D-89), hardens `serve --mcp` (4 MiB bounded reader, protocol negotiation, `version.Get().Version`, `propose_rule` validation, read-only annotations), verifies historical trailers at `<ref>`, guards `resolve --write` on conflicts, refines anti-tamper precision, and aligns Design 1.2 (`v0.4.1`).
Active: R1.AC1, R1.AC2, R1.AC3, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, R4.AC1
Reserved: none
Sources: docs/design/gradus-design.md, spec/v1/SPEC.md, docs/compatibility-matrix.md
Related: managed-settings-and-chat-bundles (makes machine-wide hooks safe outside Gradus), mcp-server (hardens stdio server), maturity-assessment-and-commit-trailers (historical ref verification), action-hardening-and-design-v1-1 (advances Design 1.1 to 1.2)

## anti-tamper-and-boundary-hardening

Purpose: Restores anti-tamper detection for `export` and standalone hook-bypass assignments without false positives, enforces `.git` boundaries in `resolveWorkspaceRoot`, keeps `status --hook` exit `0` on malformed `gradus.yaml` (D-82), refines MCP `propose_rule` diagnostics, and closes Design `O-1` (`v0.4.2`).
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R3.AC1, R3.AC2, R3.AC3, R4.AC1, R4.AC2, NFR1, NFR2, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, README.md
Related: enterprise-hardening-and-design-v1-2 (refines v0.4.1 anti-tamper, boundary, and MCP diagnostics), runtime-guard (updates hookBypassEnvRe and resolveWorkspaceRoot)

## Coherence Review

### action-hardening-and-design-v1-1
Consistent with `enterprise-hardening-and-design-v1-2#R1.AC1`, `enterprise-hardening-and-design-v1-2#R4.AC1`, `security-hardening-and-lock-upgrades#R1.AC1`, `core-schema-init#R1.AC1`, and `github-action-and-ci-agents#R1.AC1`. `action-hardening-and-design-v1-1` fixes non-Go `setup-go` resolution in `.github/actions/gradus/action.yml` without altering action inputs or exit codes, and its Design 1.1 alignment is cleanly superseded forward by `enterprise-hardening-and-design-v1-2#R4.AC1` (Design 1.2). No contradictions found.

### anti-tamper-and-boundary-hardening
Consistent with `enterprise-hardening-and-design-v1-2#R1.AC1`, `enterprise-hardening-and-design-v1-2#R3.AC3`, `action-hardening-and-design-v1-1#R1.AC1`, and `core-schema-init#R1.AC1`. Extends `hookBypassEnvRe` to cover `export`/standalone assignments, enforces `.git` boundaries in `resolveWorkspaceRoot` matching `findGradusRoot`, honors D-82 in `status --hook` on malformed `gradus.yaml` while keeping `guard` fail-closed, and closes Design `O-1`. No contradictions found.

### conditional-overlays-by-profile
Consistent with `overlays-and-exceptions#R1.AC1`, `resolve-and-lock#R3.AC1`, and `core-schema-init#R1.AC1`. Activates the `activation: profile` mode already reserved in `overlays-and-exceptions` with deterministic `OverlaySummary.Reason` and blocking `overlay` conflicts. No linked contradictions found.

### enterprise-hardening-and-design-v1-2
Consistent with `action-hardening-and-design-v1-1#R1.AC1`, `action-hardening-and-design-v1-1#R4.AC1`, `codex-copilot-opencode-targets#R4.AC1`, and `core-schema-init#R1.AC1`. Advances `docs/design/gradus-design.md` from Design 1.1 to Design 1.2 (`D-84`–`D-89`), keeps `guard` and `status --hook` safe outside Gradus repositories for enterprise `managed-settings.json` rollouts, and hardens `serve --mcp`, `trailers verify`, and `resolve --write`. No contradictions found.

### framework-mapping-and-audit-export
Consistent with `core-schema-init#R2.AC1` and `resolve-and-lock#R3.AC1`. Propagates `controls:` monotonically across rule overrides without weakening higher-level mappings and adds read-only `audit coverage` / `audit export` commands. No linked contradictions found.

### managed-settings-and-chat-bundles
Consistent with `agent-compilers#R1.AC1`, `toolchain-governance#R1.AC1`, and `enterprise-hardening-and-design-v1-2#R1.AC1`. Adds optional `--managed-settings` and `--chat-bundle` outputs to `compile` while excluding personal `user` rules from shared bundles. No linked contradictions found.

### maturity-assessment-and-commit-trailers
Consistent with `mvp-skills#R1.AC1`, `resolve-and-lock#R2.AC1`, and `enterprise-hardening-and-design-v1-2#R3.AC1`. Adds process-only `assess` metrics (strictly adhering to P16), embeds `gradus-assess` (`SKILL-08`), and emits optional git trailers verified against historical `gradus.lock`. No linked contradictions found.

### mcp-server
Consistent with `resolve-and-lock#R1.AC1`, `verify-check-evidence#R2.AC1`, and `enterprise-hardening-and-design-v1-2#R2.AC1`. Exposes read-only stdio JSON-RPC 2.0 tools without mutating repository files on disk, further hardened by `enterprise-hardening-and-design-v1-2` and `anti-tamper-and-boundary-hardening`. No linked contradictions found.

### model-policy-and-tiers
Consistent with `core-schema-init#R1.AC1`, `agent-compilers#R2.AC3`, and `verify-check-evidence#R3.AC1`. Activates `models:` capability-tier policy monotonically across levels, emits Claude skill `model:` frontmatter, and adds optional `--check-model-tier` enforcement. No linked contradictions found.

### security-hardening-and-lock-upgrades
Consistent with `action-hardening-and-design-v1-1#R1.AC1`, `core-schema-init#R1.AC1`, `github-action-and-ci-agents#R1.AC1`, and `resolve-and-lock#R2.AC1`. Implements D-80 (`view_hash` excludes personal `user` level), extends anti-tamper targets to `.codex/config.toml` and `opencode.json`, and adds `resolve --upgrade-lock`. No contradictions found.
