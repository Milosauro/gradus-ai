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
4. Author the Markdown rule with valid `gradus/v1` frontmatter (`id`, `title`, `severity`, `enforcement`, `scope`, `status: active`), clear statement, `## Why`, `## Rejected Rationalizations`, and compliant/non-compliant examples.
5. Validate the pyramid with `gradusctl resolve --json` and `gradusctl conflicts --candidates --json`.

## Outputs
- Validated Markdown rule file (or upstream PR proposal for `locked: true` levels).

## Guardrails & Prohibitions
- **NEVER** write directly to a `locked: true` level or modify `.gradus/cache/`.
- **NEVER** self-approve rules or bypass `gradusctl resolve` constraint checks.
- **NEVER** include a `level:` field in rule frontmatter (level is derived strictly from the container directory per ADR 0004).

## Rejected Rationalizations
- "I'll just patch the cached file in `.gradus/cache/` to test it quickly." -> Locked levels are verified by `gradus.lock`; propose changes upstream cleanly.
