# ADR 0004: Rule ID and Level Derivation

- **Status:** Decided
- **Date:** 2026-11-19
- **Deciders:** Alessandro Enrico Milani, Core Architecture
- **Context:** `gradus-design.md` §9.4, §9.7, §14

## Context and Problem Statement

In `gradus-design.md` §9.4, the specification states: *"The level is derived from the file location and never written in the rule."* At the same time, §9.4 recommends the ID convention `<LEVEL>-<AREA>-<NNN>` (e.g. `ORG-SEC-004`). In §9.7, the example JSON payload for the resolved view shows a level `team`, but the rule `ORG-SEC-004` carries `"level": "org"` despite provenance pointing to `team@v1.2.0`.

This creates potential ambiguity:
1. Is a rule's runtime level determined by its ID prefix or by its directory/source location?
2. If a rule originates at the `project` level (`PRJ-SEC-001`) and is later promoted to the `team` level, does its ID have to change, thereby breaking references, history, and active waivers?

## Decision Drivers

- Level derivation must be strictly deterministic and unforgeable.
- Rule IDs must be stable identifiers across promotions, audit logs, and exception waivers.
- Inconsistencies between an ID's alphanumeric prefix and its container must not invalidate rule processing.

## Decision Outcome

Chosen approach:

1. **Strict Level Derivation:** A rule's canonical level is **strictly derived from the containing level configuration** (`level.yaml` / directory location in the pyramid), never inferred from its ID prefix or frontmatter text.
2. **Stable Rule IDs:** The format `<LEVEL>-<AREA>-<NNN>` is a recommended authoring convention for initial drafting, but the ID string itself is an opaque, unique identifier across the resolved view.
3. **Promotion Stability:** When a rule is promoted from `project` to `team` or `org`, its unique ID MAY be preserved to maintain traceability, or aliased/superseded explicitly. The resolved view always reports the rule's actual current level based on its containing hierarchy.
4. The schema discrepancy noted in §9.7 is clarified: the resolved view outputs `"level"` equal to the ID of the level container in which the rule is defined.

## Consequences

- **Positive:** Renaming or reorganizing levels does not require mass-renaming of all rule IDs.
- **Positive:** System guarantees match the specification's core invariant.
