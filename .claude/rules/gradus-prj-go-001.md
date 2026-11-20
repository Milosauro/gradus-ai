---
paths:
  - "go.mod"
  - "**/*.go"
---

### PRJ-GO-001 — Build static CGO_ENABLED=0 binaries and invoke git via os/exec
- **Level:** `project` | **Severity:** `MUST` | **Enforcement:** `check` | **Source:** `project@v0.1.0`
- **Scope:** `go.mod`, `**/*.go`

Gradus MUST compile as a static binary with `CGO_ENABLED=0` and execute Git operations by shelling out to the system `git` binary via `os/exec` rather than embedding `go-git` or `libgit2` (D-69).

