---
id: PRJ-GO-001
title: Build static CGO_ENABLED=0 binaries and invoke git via os/exec
severity: MUST
enforcement: check
scope:
  - "go.mod"
  - "**/*.go"
checker:
  name: pattern
  pattern: "github\\.com/go-git|libgit2"
status: active
---

Gradus MUST compile as a static binary with `CGO_ENABLED=0` and execute Git operations by shelling out to the system `git` binary via `os/exec` rather than embedding `go-git` or `libgit2` (D-69).

## Why
Avoids CGO cross-compilation failures across Linux, macOS, and Windows and keeps the dependency tree minimal and auditable.

## Examples

### Compliant
```go
cmd := exec.Command("git", "rev-parse", "HEAD")
```

### Non-compliant
```go
import "github.com/go-git/go-git/v5"
```
