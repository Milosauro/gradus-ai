# Gradus

**Layered, governed rules for AI coding agents — from a single developer to the whole organization.**

> **Status: early design (pre-alpha).** Nothing is ready to install yet. The design is being worked out in the open; formats and commands will change.

## The problem

Teams adopting AI coding agents (Claude Code, Antigravity, Codex, GitHub Copilot, OpenCode, Cursor and others) quickly end up with every developer running their own setup: personal instruction files, ad-hoc rules, skills installed at will. The same repository gets changed by agents following different, often contradictory rules, and company policies on data, security and compliance are easy to miss.

## The idea

Gradus organizes the rules agents must follow into a **pyramid of levels** (*gradus* is Latin for "step"), from the most general to the most specific — for example organization → team → project → developer. Every organization defines its own levels.

What Gradus focuses on is not just layering rules, but **governing** them:

- **Configurable levels**: each level can be local or fetched from a versioned remote source, locked or editable, with its own owners and approvers.
- **Restriction invariant**: lower levels can restrict or specialize the levels above, never relax them, unless a rule explicitly allows it.
- **Level charters**: each level states what belongs there and what does not.
- **Conditional overlays**: cross-cutting rule sets (e.g. data protection, payments, a specific client, a tech stack) that apply only where they are relevant.
- **Obligations and bindings**: a level can require an outcome (e.g. "an approved spec before code") and let teams choose the tool that satisfies it.
- **Deterministic where possible**: rules that can be checked mechanically are enforced through hooks, git hooks and CI, reusing existing checkers instead of reinventing them.
- **Learning loop**: repeated corrections become lessons, and lessons can be promoted to rules at the right level — always with human approval.
- **Tool-agnostic**: built on open formats (Agent Skills, AGENTS.md, MCP) to work with as many coding agents as possible.

Gradus is meant to be useful for **a single developer with personal projects** and to grow, without changing tools, to teams, business units and the whole company.

## Planned components

| Component | Purpose |
|---|---|
| `spec/` | Open specification of the formats (pyramid, levels, rules, overlays, lockfile, evidence) |
| `skills/` | Agent skills: onboarding, loading, authoring, review, learning, linking, reconciliation |
| `gradusctl` | CLI that resolves, verifies and compiles the pyramid for each agent |
| `adapters/` | Integrations with agent tools and development workflows |
| `templates/`, `examples/` | Starting points to adapt — never a one-size-fits-all scheme |

## Roadmap

1. **Phase 0** — personal repositories: core format, CLI basics, first skills (Claude Code and Antigravity, plus AGENTS.md for other agents).
2. **Phase 1** — team: shared levels, charters, rule tests, CI checks.
3. **Phase 2** — business unit: locked remote levels, lockfile, obligations and bindings.
4. **Phase 3** — organization: overlays, compliance mappings, chat agents and MCP.

## Contributing

The project is in its design phase. Feedback and discussion are welcome through issues.

## License

Licensed under the [Apache License 2.0](LICENSE).
