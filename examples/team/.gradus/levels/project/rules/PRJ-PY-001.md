---
id: PRJ-PY-001
title: Forbid debug print statements in Python backend modules
severity: SHOULD
enforcement: check
scope: ["backend/**/*.py"]
checker:
  name: pattern
  pattern: '\bprint\('
status: active
---

Use structured logging instead of `print(` in backend Python modules.

## Examples

### Compliant
```python
logger.info("processed batch")
```

### Non-compliant
```python
print("processed batch")
```
