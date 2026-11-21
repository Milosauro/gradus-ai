---
id: PRJ-DBG-001
title: Forbid leftover console.log statements in TypeScript sources
severity: SHOULD
enforcement: check
scope: ["src/**/*.ts"]
checker:
  name: pattern
  pattern: '\bconsole\.log\('
status: active
---

Remove temporary `console.log(` calls before committing TypeScript modules.

## Examples

### Compliant
```ts
logger.info("ready");
```

### Non-compliant
```ts
console.log("debug");
```
