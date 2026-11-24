---
id: PRJ-CI-001
title: Pin GitHub Actions to 40-character commit SHAs
severity: MUST
enforcement: check
scope:
  - ".github/workflows/*.yml"
  - ".github/workflows/*.yaml"
  - ".github/actions/**/*.yml"
  - ".github/actions/**/*.yaml"
checker:
  name: regex
  pattern: 'uses:\s*[^#\s]+@(?:v[0-9]|main|master|[a-zA-Z0-9._-]{1,39}(?:\s|$))'
status: active
---

Every `uses:` reference in `.github/workflows/*.yml` and `.github/actions/**/*.yml` MUST be pinned to a full 40-character immutable commit SHA with an accompanying version comment (Threat Model T3).

## Why
Prevents supply-chain tag-mutation attacks against CI and release workflows.

## Rejected Rationalizations
- "Using `@v4` is shorter and official." — Mutable tags can be force-pushed if an upstream action account is compromised; always pin the full commit SHA.

## Examples

### Compliant
```yaml
- uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
```

### Non-compliant
```yaml
- uses: actions/checkout@v4
```
