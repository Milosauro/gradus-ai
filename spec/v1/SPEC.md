# Gradus Open Specification (`gradus/v1`)

| Field | Value |
|---|---|
| Schema Identifier | `gradus/v1` |
| Status | Active (`v0.3.x`, Phase 2) |
| Normative Formats | `spec/v1/schema/*.json` |
| Normative Semantics | `docs/design/gradus-design.md` (Design 1.1) and `docs/adr/` |

## 1. Normative Precedence

When specification sources overlap, precedence follows Design Decision **D-75**:

1. **`spec/v1/schema/*.json`** is normative for file formats, field names, and structural validation.
2. **`docs/design/gradus-design.md` and accepted ADRs (`docs/adr/`)** are normative for semantics (precedence, invariants, enforcement, exit codes, and governance workflows).
3. **`spec/v1/SPEC.md`** (this document) is a descriptive guide summarizing the schemas and conventions.

## 2. Overview

The `gradus/v1` specification defines the file formats for layered AI coding agent governance:

1. **Pyramid Definition (`gradus.yaml`)**: Ordered list of levels (`user` always last per ADR 0001), optional overlays, applicability profile, obligations, bindings, declared toolchain, compilation targets, anti-tamper policy, git hook manager integration, and evidence sinks.
2. **Level Definition (`level.yaml`)**: Level metadata, ownership, approvers, charter, context budget, inherited constraints for lower levels, overlay activation rules, obligations, bindings, toolchain, and exception policy.
3. **Overlay Definition (`overlay.yaml`)**: Cross-cutting rule sets with declared `strength` and `activation` conditions.
4. **Rule Files (`rules/<ID>.md`)**: Markdown files with YAML frontmatter. A rule's `id` is opaque and its effective `level` is strictly derived from its containing level container (ADR 0004).
5. **Lockfile (`gradus.lock`)**: Pinned versions, commit hashes, signature verification status, normalized SHA-256 content hashes, and project-only `view_hash` (excluding `user` and `lock_hash`, D-80).
6. **Resolved View**: Merged, conflict-checked JSON view consumed by compilers and agents.
7. **Evidence Record**: Machine-readable verification/check/review verdict bound to commit and `lock_hash`.
8. **Lesson Record**: Local redacted JSON observation captured under `~/.gradus/lessons/<project-key>/`.
9. **Exception Record**: Time-limited, approved waiver or break-glass bypass.

## 3. Forward Compatibility

Parsers MUST treat unknown YAML/JSON keys as diagnostic notes rather than fatal errors. `obligations`, `bindings`, and `toolchain` are active in Phase 2 (`v0.3.x`); `models` and `adapters` remain reserved top-level keys for Phase 3 and emit informational notes when encountered.

## 4. Exit Codes

- `0`: Pass (no violations, no conflicts, no errors)
- `1`: Policy violation, drift, or unresolved conflict
- `2`: Technical error (invalid schema, missing/failing checker, lock mismatch, unauthorized approver, invalid exception duration, unreachable `fail-closed` source)
