---
id: TEAM-SEC-001
title: Forbid hardcoded private keys in source files
severity: MUST
enforcement: check
scope: ["**/*"]
checker:
  name: pattern
  pattern: '-----BEGIN [A-Z ]*PRIVATE KEY-----'
status: active
---

Private keys MUST NEVER be committed to any repository file.

## Examples

### Compliant
```text
KEY_REF=vault://team/signing-key
```

### Non-compliant
```text
-----BEGIN RSA PRIVATE KEY-----
```
