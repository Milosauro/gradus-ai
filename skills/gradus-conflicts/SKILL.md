---
name: gradus-conflicts
description: "Triage semantic conflict candidate pairs from gradusctl conflicts --candidates with justification and confidence, leaving confirm/dismiss decisions to a human."
---

# Gradus Conflicts Skill

## Inputs
- Candidate rule pairs from `gradusctl conflicts --candidates --json`.
- Full rule text and scopes from `gradusctl resolve --json`.

## Workflow
1. Fetch unjudged candidate pairs with `gradusctl conflicts --candidates --json`.
2. For each pair, compare rule intent, scopes, severities, and enforcement modes.
3. Classify each pair as `contradiction`, `duplicate`, `redundant`, or `compatible` with a concise justification and confidence score (`0.0`–`1.0`).
4. Present the classifications to the human maintainer and offer explicit commands:
   - `gradusctl conflicts --confirm --rule-a <A> --rule-b <B> --message "<reason>"`
   - `gradusctl conflicts --dismiss --rule-a <A> --rule-b <B> --message "<reason>"`
5. Execute `--confirm` or `--dismiss` only after explicit human decision.

## Outputs
- Classification table (`contradiction` | `duplicate` | `redundant` | `compatible`) with justification and confidence for human decision.

## Guardrails & Prohibitions
- **NEVER** autonomously confirm, dismiss, or block builds on suspected semantic conflicts without human confirmation.
- **NEVER** edit or delete rules on your own to "resolve" a conflict without user approval.

## Rejected Rationalizations
- "These two rules clearly contradict each other, so I'll run `--confirm` automatically to protect the build." -> Semantic conflict detection is advisory (§10.1); only a human decides whether to confirm, dismiss, or fix.
