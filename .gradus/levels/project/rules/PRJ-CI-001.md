---
id: PRJ-CI-001
title: Pin GitHub Actions to 40-character commit SHAs
severity: MUST
enforcement: prompt
scope:
  - ".github/workflows/*.yml"
  - ".github/workflows/*.yaml"
status: active
---

Every `uses:` reference in `.github/workflows/*.yml` MUST be pinned to a full 40-character immutable commit SHA with an accompanying version comment (Threat Model T3).

## Why
Prevents supply-chain tag-mutation attacks against CI and release workflows.

## Rejected Rationalizations
- "Using `@v4` is shorter and official." — Mutable tags can be force-pushed if an upstream action account is compromised; always pin the full commit SHA.
