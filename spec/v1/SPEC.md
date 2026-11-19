# Gradus Open Specification (`gradus/v1`)

| Field | Value |
|---|---|
| Schema Identifier | `gradus/v1` |
| Status | Draft |
| Canonical Reference | `docs/design/gradus-design.md` |

## 1. Overview

The `gradus/v1` specification defines the file formats for layered AI coding agent governance:

1. **Pyramid Definition (`gradus.yaml`)**: Ordered list of levels, optional overlays, applicability profile, compilation targets, git hook manager integration, and evidence sinks.
2. **Level Definition (`level.yaml`)**: Level metadata, ownership, approvers, charter, inherited constraints for lower levels, overlay activation rules, and exception policy.
3. **Overlay Definition (`overlay.yaml`)**: Cross-cutting rule sets with declared `strength` and `activation` conditions.
4. **Rule Files (`rules/<ID>.md`)**: Markdown files with YAML frontmatter. A rule's effective `level` is strictly derived from its containing level container (ADR 0004).
5. **Lockfile (`gradus.lock`)**: Pinned versions, commit hashes, normalized SHA-256 content hashes, and resolved `view_hash`.
6. **Resolved View**: Merged, conflict-checked JSON view consumed by compilers and agents.
7. **Evidence Record**: Machine-readable verification/check/review verdict bound to commit and `lock_hash`.
8. **Lesson Record**: Local redacted observation captured during sessions.
9. **Exception Record**: Time-limited, approved waiver or break-glass bypass.

## 2. Forward Compatibility

Parsers MUST treat unknown YAML/JSON keys as diagnostic notes rather than fatal errors. Top-level reserved keys (`obligations`, `bindings`, `toolchain`, `models`, `adapters`) are reserved for future phases and emit an informational note when encountered in Phase 0.

## 3. Exit Codes

- `0`: Pass (no violations, no conflicts, no errors)
- `1`: Policy violation, drift, or unresolved conflict
- `2`: Technical error (invalid schema, missing checker, lock mismatch, unreachable `fail-closed` source)
