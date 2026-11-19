# ADR 0002: Cumulative Enforcement Semantics

- **Status:** Decided
- **Date:** 2026-11-19
- **Deciders:** Alessandro Enrico Milani, Core Architecture
- **Context:** `gradus-design.md` §4 (Principle P3), §9.4, §11.2

## Context and Problem Statement

In `gradus-design.md` §9.4, `enforcement` in rule frontmatter is defined as a single enum value: `prompt`, `review`, `check`, `hook`, `permission`. However, §11.2 outlines an enforcement map where `MUST` rules with mechanical checks combine prompt instructions with deterministic hooks, permissions, or CI checks (e.g. `hook + check + prompt`). Principle P3 explicitly states: *"Policy is loaded wherever output is produced, not only where output is checked."*

There was a potential ambiguity regarding whether declaring `enforcement: check` or `enforcement: hook` excludes the rule from the agent prompt instructions view.

## Decision Drivers

- Agents must be informed of constraints upfront so they do not generate non-compliant code only to be blocked downstream.
- Mechanical enforcement (hooks, CI checks) must guarantee compliance even if an agent ignores the prompt.
- Rule authors should declare the primary enforcement tier cleanly without combinatorial enum explosion in YAML frontmatter.

## Decision Outcome

Chosen approach: **Cumulative Enforcement Semantics toward instruction generation**.

1. Every active rule in the resolved view is included in compiled agent instruction files (`AGENTS.md`, `CLAUDE.md`, `.agents/rules/`), regardless of its `enforcement` level. Prompt guidance is the universal base layer (Principle P3).
2. The `enforcement` frontmatter field specifies the highest deterministic tier applied to the rule:
   - `prompt`: Informative instruction only.
   - `review`: Evaluated during advisory semantic review (`gradus-review`).
   - `check`: Evaluated deterministically by `gradusctl check` in git pre-commit hooks and CI pipelines; requires a `checker` definition.
   - `hook`: Evaluated deterministically at runtime by `gradusctl guard` on agent `PreToolUse` actions; requires a `guard` definition.
   - `permission`: Compiled into native agent tool-use permissions where supported.
3. A single rule may declare both a runtime `guard` and a repository `checker`. In this case, `enforcement: hook` or `check` applies, and both engines execute their respective verification phases.

## Consequences

- **Positive:** Agents receive full constraint context before executing actions.
- **Positive:** Deterministic checks and guards back up prompts unconditionally.
- **Positive:** Clean rule frontmatter schemas without redundant flags.
