---
id: CMP-PII-001
title: Forbid raw PAN/card numbers in test fixtures
severity: MUST
enforcement: check
scope: ["tests/**/*"]
checker:
  name: pattern
  pattern: '\b4[0-9]{12}(?:[0-9]{3})?\b'
status: active
---

Test fixtures MUST use synthetic tokenized placeholders rather than card-shaped 13/16-digit PANs.

## Examples

### Compliant
```json
{"card_token": "tok_test_synthetic"}
```

### Non-compliant
```json
{"pan": "4111111111111111"}
```
