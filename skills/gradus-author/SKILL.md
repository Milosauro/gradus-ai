---
name: gradus-author
description: "Author or refine Gradus rules with valid frontmatter, level charter checks, rejected rationalizations, and examples without modifying locked levels directly."
---

# Gradus Author Skill

## Inputs
- User intent, design documents, or promoted lessons from `gradusctl lessons list --json`.
- Target level charters (`level.yaml`) and `gradusctl resolve --json` output.

## Workflow
1. Inspect `gradus.yaml` and the target level's `level.yaml` charter to verify the rule belongs at that level.
2. If the rule belongs to another level, redirect it to the appropriate level and explain why.
3. If the target level has `locked: true`, draft a pull-request proposal for the upstream level repository instead of editing local locked caches.
4. Author the Markdown rule with valid `gradus/v1` frontmatter (`id`, `title`, `severity`, `enforcement`, `scope`, `status: active`), clear statement, `## Why`, `## Rationalizations`, and `## Examples` (`### Compliant` / `### Non-compliant`).
5. Validate the pyramid with `gradusctl resolve --json` and `gradusctl conflicts --candidates --json`.

## Schema Templates

**`level.yaml` (`charter` must be an object, not a string):**
```yaml
schema: gradus/v1
id: project
version: "0.1.0"
charter:
  purpose: Project-specific engineering rules.
  allowed_severities: [MUST, SHOULD, MAY]
```

**`enforcement: check` (deterministic pattern or external checker):**
```yaml
---
id: PRJ-BE-001
title: Forbid debug print statements in backend code
severity: MUST
enforcement: check
scope: ["backend/**/*.py"]
checker:
  name: pattern
  pattern: '\bprint\('
status: active
---
```

**`enforcement: hook` (runtime `PreToolUse` guard):**
```yaml
---
id: PRJ-GIT-001
title: Confirm before running git commit or git push
severity: MUST
enforcement: hook
scope: ["**/*"]
guard:
  action: command
  pattern: '\bgit\b(?:\s+(?:-[Cc]\s+\S+|-\S+))*\s+(commit|push)(?:\s|$)'
  decision: ask
status: active
---
```

## Outputs
- Validated Markdown rule file (or upstream PR proposal for `locked: true` levels).

## Guardrails & Prohibitions
- **NEVER** write directly to a `locked: true` level or modify `.gradus/cache/`.
- **NEVER** self-approve rules or bypass `gradusctl resolve` constraint checks.
- **NEVER** include a `level:` field in rule frontmatter (level is derived strictly from the container directory per ADR 0004).

## Rejected Rationalizations
- "I'll just patch the cached file in `.gradus/cache/` to test it quickly." -> Locked levels are verified by `gradus.lock`; propose changes upstream cleanly.
