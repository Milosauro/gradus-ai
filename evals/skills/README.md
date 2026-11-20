# Gradus Skill-TDD Regression Suite (`evals/skills/`)

This directory contains structured behavioral regression scenarios for all six canonical MVP Gradus Agent Skills (`gradus-design.md` §23).

## Methodology (TDD for Skills)

1. **Baseline (Red):** Run realistic scenarios without the skill attached and record observed failures and verbatim agent rationalizations (`baseline_failures`, `baseline_rationalizations`).
2. **Minimal Skill (Green):** Author the smallest `skills/gradus-*/SKILL.md` workflow and guardrails addressing the observed failures.
3. **Regression Suite:** Keep each scenario in `evals/skills/*.yaml` with `required_behaviors` and `forbidden_behaviors`, validated deterministically in CI by `skills/skills_test.go` and exercised live in `evals/benchmark/`.
