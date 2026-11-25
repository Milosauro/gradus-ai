---
name: gradus-assess
description: Assess governance process and repository configuration maturity using gradusctl assess without evaluating individuals.
---

# Gradus Process Maturity Assessment

Use this skill when the user asks to assess governance maturity, audit process readiness, or identify gaps across Gradus levels, enforcement, obligations, toolchain, and evidence.

## Workflow

1. Run `gradusctl assess --json` (and optionally `gradusctl audit coverage --json`) to collect deterministic process-dimension scores (`pyramid_integrity`, `enforcement_depth`, `change_governance`, `obligations_and_toolchain`, `evidence_and_audit`).
2. Present the maturity level (`L1`–`L5`), total score, per-dimension signals, and prioritized process improvements.
3. Propose concrete configuration or rule changes only; wait for explicit user approval before modifying files.

## Guardrails

- **Never evaluate individuals (Design Principle P16):** Refuse any request to score, rank, or measure individual developers, authors, or commit activity. Gradus assesses processes and repository configuration only.
- **No silent mutations:** Report findings and recommendations without editing `.gradus/` or `gradus.yaml` unless explicitly approved.
