# ADR 0003: Static Guard Dispatcher

- **Status:** Decided
- **Date:** 2026-11-19
- **Deciders:** Alessandro Enrico Milani, Core Architecture
- **Context:** `gradus-design.md` §9.1, §9.8, §11.3, §14

## Context and Problem Statement

When an agent learns a personal correction (e.g. *"Do not commit without explicit request"*, §9.8), it is promoted to a rule at the `user` level with `enforcement: hook`. However, agent hook configuration files in the workspace (such as `.agents/hooks.json` for Antigravity or `.claude/settings.json` for Claude Code) are typically checked into Git to provide shared team-level hooks.

If `gradusctl compile` directly inscribed personal rule patterns and guards into `.agents/hooks.json`, personal developer habits would dirty the repository's git status and leak across team members.

## Decision Drivers

- Personal preferences (`~/.gradus/levels/user/`) must remain local and never contaminate shared git-tracked repository files.
- Agent hooks must execute deterministically with near-zero latency (`< 100ms`).
- Hook definitions should survive configuration changes without requiring repository re-commits.

## Considered Options

1. **Inline Rule Guards in `.agents/hooks.json`:** Compile active regexes and tool names directly into the hook configuration.
2. **Static Dispatcher Command:** Register a static call to `gradusctl guard` in `.agents/hooks.json`; `gradusctl guard` dynamically evaluates the local resolved view (including the user level) at runtime.

## Decision Outcome

Chosen option: **Option 2 (Static Dispatcher Command)**.

1. `gradusctl compile` writes or merges a generic dispatcher hook into `.agents/hooks.json` and `.claude/settings.json`. For example, on `PreToolUse` for shell execution and file-writing tools:
   ```json
   {
     "type": "command",
     "command": "gradusctl guard"
   }
   ```
2. When triggered, `gradusctl guard` reads the tool invocation payload from `stdin`.
3. `gradusctl guard` loads the cached local resolved view (which incorporates both the repository pyramid and `~/.gradus/levels/user/`).
4. It evaluates all active `hook` guards against the invocation payload and returns the decision (`allow`, `ask`, `deny`) via stdout/exit code.

## Consequences

- **Positive:** `.agents/hooks.json` remains completely static and agnostic to personal developer rules.
- **Positive:** Adding, removing, or tweaking user-level guards requires zero git-tracked file modifications.
- **Requirement:** `gradusctl guard` must load its cached view and evaluate rules in under 100ms so agent responsiveness is not impaired.
