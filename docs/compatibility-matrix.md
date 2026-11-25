# Gradus Agent Compatibility Matrix

This matrix documents compiled artifacts and supported Gradus enforcement capabilities across all six official `targets` in `gradus/v1` (`docs/design/gradus-design.md` §20, §24.1).

| Target | Primary Instructions | Scoped Rules | Embedded Skills | User-Level Rules | `PreToolUse` Guard | `SessionStart` Hook | Permissions |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `claude-code` | `CLAUDE.md` (managed section) | `.claude/rules/gradus-*.md` (`paths:`) | `.claude/skills/gradus-*/SKILL.md` | `CLAUDE.local.md` (git-ignored) | `.claude/settings.json` (`gradusctl guard`) | `.claude/settings.json` (`gradusctl status --brief --hook`) | `permissions.deny` / `permissions.ask` |
| `antigravity` | `AGENTS.md` (managed section) | `.agents/rules/gradus-*.md` (`trigger: glob`, `<12k` chars) | `.agents/skills/gradus-*/SKILL.md` | `.agents/rules/gradus-user-*.md` | `.agents/hooks.json` (`gradusctl guard`) | `.agents/hooks.json` (`gradusctl status --brief --antigravity-hook`) | Via `gradusctl guard` |
| `agents-md` | `AGENTS.md` (managed section) | Included in `AGENTS.md` | Not emitted | Excluded (repo-shared) | Manual / git hooks | Not emitted | Not emitted |
| `codex` | `AGENTS.md` (managed section) | Included in `AGENTS.md` | `.agents/skills/gradus-*/SKILL.md` | Excluded (repo-shared) | Not supported (use `gradusctl hooks` & CI) | Not supported | Not supported |
| `copilot` | `.github/copilot-instructions.md` (managed section) | `.github/instructions/gradus-*.instructions.md` (`applyTo:`) | `.github/skills/gradus-*/SKILL.md` | Excluded (repo-shared) | Not supported (use `gradusctl hooks` & CI) | Not supported | Not supported |
| `opencode` | `AGENTS.md` (managed section) | Included in `AGENTS.md` | `.opencode/skills/gradus-*/SKILL.md` | Excluded (repo-shared) | Not supported (use `gradusctl hooks` & CI) | Not supported | Not supported |

## Phase 3 Enterprise & Protocol Surfaces

| Surface | Mechanism | Scope & Notes |
| --- | --- | --- |
| Enterprise Managed Settings | `gradusctl compile --managed-settings <path>` | Generates `managed-settings.json` for `claude-code` (`hooks`, `permissions`, `deniedMcpServers`, `disableBypassPermissionsMode`). `gradusctl guard` and `status --hook` safely allow/exit `0` outside Gradus repositories. |
| Chat Skill Bundles | `gradusctl compile --chat-bundle <path.zip>` | Byte-reproducible ZIP (`manifest.json`, `RULES.md` excluding `user` rules, embedded `skills/gradus-*/SKILL.md`). |
| Model Policy (`models:`) | `models.tiers` + `models.operations` | Compiled `model:` frontmatter emitted for `claude-code` skills; `gradusctl verify --check-model-tier` checks self-reported `model_tier` in local evidence. |
| MCP Server | `gradusctl serve --mcp` | Zero-dependency JSON-RPC 2.0 stdio server exposing `resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, `get_toolchain` (all read-only/non-mutating). |

## Notes

- All six targets support deterministic zero-diff compilation (`gradusctl compile`) and drift detection (`gradusctl verify`, `gradusctl status`, `gradusctl doctor`, `gradusctl reconcile`).
- Targets without native `PreToolUse` runtime hooks (`agents-md`, `codex`, `copilot`, `opencode`) enforce deterministic checks via Git hooks (`gradusctl hooks`) and CI (`gradusctl verify`, `gradusctl check`).
