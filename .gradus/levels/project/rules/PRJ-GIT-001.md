---
id: PRJ-GIT-001
title: Confirm before running git commit or git push
severity: MUST
enforcement: hook
scope:
  - "**/*"
guard:
  action: command
  pattern: '\bgit\b(?:\s+(?:-[Cc]\s+\S+|-\S+))*\s+(commit|push)\b'
  decision: ask
status: active
---

AI coding agents MUST NOT run `git commit` or `git push` without explicit user confirmation in chat.

## Why
Ensures human review before creating git history or publishing changes to remote branches.
