---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T08:50:33Z
last_modified: 2026-11-20T08:50:33Z
approved_fingerprint: sha256:d159228209801f61eb7ad4b1e605de72a53b27652823541320e93aa35db73f8e
---

# Current Contracts

## core-schema-init

Purpose: Provides the `gradus/v1` specification, JSON Schemas, forward-compatible YAML/Markdown parsers with ADR 0004 level derivation, and `gradusctl init`/`explain`.
Active: R1.AC1, R1.AC2, R2.AC1, R2.AC2, R2.AC3, R2.AC4, R3.AC1, R3.AC2, R3.AC3, R3.AC4, R4.AC1, R4.AC2, R4.AC3, R5.AC1, R5.AC2, R5.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0004-rule-id-and-level-derivation.md
Related: resolve-and-lock (consumes internal/schema configuration and rule parsers)

## resolve-and-lock

Purpose: Resolves local and `os/exec` Git pyramid sources, verifies normalized `gradus.lock` integrity, enforces §10 constraints and the P2 restriction invariant (with ADR 0001), and exposes `gradusctl resolve`/`status`.
Active: R1.AC1, R1.AC2, R1.AC3, R1.AC4, R2.AC1, R2.AC2, R2.AC3, R3.AC1, R3.AC2, R3.AC3, R3.AC4, R3.AC5, R4.AC1, R4.AC2, R4.AC3, NFR1, NFR2, NFR3, C1, C2
Reserved: none
Sources: docs/design/gradus-design.md, docs/adr/0001-level-ordering-and-ci-overrides.md, docs/adr/0004-rule-id-and-level-derivation.md
Related: core-schema-init (builds on internal/schema parsers and ExitErrorWithCode semantics)
