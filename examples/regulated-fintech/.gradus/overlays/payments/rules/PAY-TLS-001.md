---
id: PAY-TLS-001
title: Forbid insecure http:// payment gateway URLs
severity: MUST
enforcement: check
scope: ["src/**/*"]
checker:
  name: pattern
  pattern: 'http://[^\s"'']*pay'
status: active
---

Payment integration endpoints MUST use `https://` rather than plaintext `http://`.

## Examples

### Compliant
```text
GATEWAY_URL=https://api.payments.example/v1
```

### Non-compliant
```text
GATEWAY_URL=http://api.payments.example/v1
```
