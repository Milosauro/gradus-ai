---
paths:
  - ".github/workflows/*.yml"
  - ".github/workflows/*.yaml"
---

### PRJ-CI-001 — Pin GitHub Actions to 40-character commit SHAs
- **Level:** `project` | **Severity:** `MUST` | **Enforcement:** `prompt` | **Source:** `project@v0.1.0`
- **Scope:** `.github/workflows/*.yml`, `.github/workflows/*.yaml`

Every `uses:` reference in `.github/workflows/*.yml` MUST be pinned to a full 40-character immutable commit SHA with an accompanying version comment (Threat Model T3).

