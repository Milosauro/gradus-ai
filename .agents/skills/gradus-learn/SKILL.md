---
name: gradus-learn
description: "Capture session corrections and repeated mistakes as redacted local lessons via gradusctl lessons and propose rule promotions only with explicit user approval."
---

# Gradus Learn Skill

## Inputs
- Current session signals (user corrections, repeated mistakes, explicit preferences) or user-supplied conversation excerpts.
- Existing lessons from `gradusctl lessons list --json`.

## Workflow
1. Watch for session learning signals: explicit user corrections ("don't do X", "use Y instead"), repeated mistakes across turns, or explicit workflow/style preferences.
2. Redact any secrets, API tokens, credentials, passwords, IBANs, phone numbers, or emails before passing text to `gradusctl`.
3. Persist or update the lesson locally via `gradusctl lessons add --kind <kind> --statement "<redacted>" --level <suggested> --enforcement <suggested>` (or `gradusctl lessons update --increment` when seen again).
4. Once a lesson reaches `occurrences >= 2` (or on explicit user request), present a rule promotion proposal (suggested level, enforcement, and draft rule) to the user.
5. Only after explicit human approval, hand off to `gradus-author` to create the rule and mark the lesson `promoted` via `gradusctl lessons update --status promoted`.

## Outputs
- Redacted local lesson record under `~/.gradus/lessons/<project-key>/` and optional promotion proposal.

## Guardrails & Prohibitions
- **NEVER** promote a lesson into a project, team, or user rule without explicit user approval.
- **NEVER** store unredacted secrets, tokens, or PII in lesson statements or evidence.
- **NEVER** record personal user-activity metrics (P16).

## Rejected Rationalizations
- "The user corrected me twice, so I'll automatically add a rule to `.gradus/levels/project/rules/`." -> Always record the lesson first and wait for explicit approval before promoting.
