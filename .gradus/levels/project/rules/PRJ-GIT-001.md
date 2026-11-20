---
id: PRJ-GIT-001
title: Confirm before running git commit or git push
severity: MUST
enforcement: hook
scope:
  - "**/*"
guard:
  action: command
  pattern: "git\\s+(commit|push)"
  decision: ask
status: active
---

AI coding agents MUST NOT run `git commit` or `git push` without explicit user confirmation in chat.

## Why
Ensures human review before creating git history or publishing changes to remote branches.
