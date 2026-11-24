---
trigger: glob
globs: ".github/workflows/*.yml, .github/workflows/*.yaml, .github/actions/**/*.yml, .github/actions/**/*.yaml"
---

### PRJ-CI-001 — Pin GitHub Actions to 40-character commit SHAs
- **Level:** `project` | **Severity:** `MUST` | **Enforcement:** `check` | **Source:** `project@v0.1.0`
- **Scope:** `.github/workflows/*.yml`, `.github/workflows/*.yaml`, `.github/actions/**/*.yml`, `.github/actions/**/*.yaml`

Every `uses:` reference in `.github/workflows/*.yml` and `.github/actions/**/*.yml` MUST be pinned to a full 40-character immutable commit SHA with an accompanying version comment (Threat Model T3).

