---
name: gradus-reconcile
description: "Explain compiled-target drift reported by gradusctl reconcile and execute deterministic restore or move actions through the CLI without editing locked levels."
---

# Gradus Reconcile Skill

## Inputs
- Drift report from `gradusctl reconcile --json` and pyramid configuration (`gradus.yaml`).

## Workflow
1. Run `gradusctl reconcile --json` to list all drifted compiled targets (`AGENTS.md`, `CLAUDE.md`, `.agents/`, `.claude/`).
2. Explain each drift item clearly along with the allowed resolution options:
   - `restore`: regenerate deterministic compiled output (`gradusctl reconcile --restore`).
   - `move`: move a valid rule into an unlocked editable level (`gradusctl reconcile --move --rule-file <path> --target-level <id>`).
   - `upstream` / `exception` / `postpone` (never postpone `fail-closed` violations).
3. Ask the user which option to apply for each drift item.
4. Execute the chosen action strictly via `gradusctl reconcile` and verify cleanliness with `gradusctl verify --json`.

## Outputs
- Plain-language explanation of drift items and deterministic reconciliation executed via `gradusctl reconcile`.

## Guardrails & Prohibitions
- **NEVER** hand-edit generated managed sections or `.agents/`/`.claude/` files directly instead of using `gradusctl reconcile`.
- **NEVER** attempt to move or write rules into a `locked: true` level.
- **NEVER** postpone or silence `fail-closed` source or lock verification errors.

## Rejected Rationalizations
- "I'll just edit `AGENTS.md` manually so `gradusctl verify` stops complaining." -> Manual edits inside managed markers drift immediately; always reconcile through `gradusctl reconcile`.
