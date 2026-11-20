---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T14:32:13Z
last_modified: 2026-11-20T14:32:13Z
approved_fingerprint: sha256:36e285a5f61f96f284df32e4742a0ab51e96055d939251ec8ee8a0954e932e5b
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

## Coherence Review

### cli-lifecycle-state
Consistent with `resolve-and-lock#R3.AC4`, `agent-compilers#R4.AC1`, and `mvp-skills#R2.AC1`. `cli-lifecycle-state#R1.AC2` writes `.gradus/findings/*.yaml` in the exact format consumed by `resolve.Resolve`, `cli-lifecycle-state#R2.AC2` redacts secrets/PII before writing lessons (T4/P16), and `cli-lifecycle-state#R3.AC4` refuses mutations to `locked: true` levels with exit code `2`. No inconsistencies found.

### mvp-skills
Consistent with `agent-compilers#R2.AC3`, `agent-compilers#R3.AC3`, and `cli-lifecycle-state#R1.AC1`–`R3.AC4`. `mvp-skills#R1.AC1` replaces `gradus-placeholder` with the six canonical skills extracted via `skills/embed.go` into `.claude/skills/` and `.agents/skills/`, and `mvp-skills#R2.AC1`–`R2.AC3` delegate all state mutations to `gradusctl` without bypassing locked levels or approval gates. No inconsistencies found.
