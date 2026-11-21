---
id: PRJ-GIT-001
title: Confirm before running git commit or git push
severity: MUST
enforcement: hook
scope: ["**/*"]
guard:
  action: command
  pattern: '\bgit\b(?:\s+(?:-[Cc]\s+\S+|-\S+))*\s+(commit|push)(?:\s|$)'
  decision: ask
status: active
---

AI coding agents MUST NOT run `git commit` or `git push` without explicit human confirmation.

## Examples

### Compliant
```bash
git status
```

### Non-compliant
```bash
git commit -m "wip"
```
