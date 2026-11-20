---
id: PRJ-LANG-001
title: Use canonical English in all repository files
severity: MUST
enforcement: prompt
scope:
  - "**/*"
status: active
---

All repository files (Go code, comments, CLI messages, documentation, `.walden/*`, `.gradus/*`, commit messages, and PR descriptions) MUST be written in canonical English.

## Why
Keeps the open-source codebase, specifications, and governance artifacts consistent for all contributors and automated tooling.

## Rejected Rationalizations
- "This comment or doc is only temporary." — Drafts committed to the repository become permanent history; write them in English from the start.
