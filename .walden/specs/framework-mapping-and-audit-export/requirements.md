---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T20:04:37Z
last_modified: 2026-11-24T20:04:37Z
approved_fingerprint: sha256:f2897bab33e268837c9d74383408d8b0f817ef82054510884cc8c65c6d33bce3
---

# Requirements Document

## Introduction

Feature `F24` (`framework-mapping-and-audit-export`) implements Design capability `X-02`: propagating rule `controls:` mappings into the resolved view, computing deterministic per-framework coverage reports, and exporting structured JSON/Markdown audit bundles combining resolved rules, provenance, active exceptions, and local evidence records.

## Requirements

### R1 Propagate Rule `controls` into ResolvedView

**User Story:** As a compliance engineer, I want rule `controls:` metadata to be preserved in `ResolvedView` (including monotonically across overrides) so downstream tools and auditors can trace every control to its governing rules.

#### Acceptance Criteria

1. `R1.AC1` WHEN a rule file declares `controls:` (e.g. `ISO27001:A.8.28`, `SOC2:CC8.1`), `resolve.Resolve` SHALL include those `controls` on `ResolvedRule.Controls` in `ResolvedView`.
   - Acceptance check: `TestAuditCoverageAndExport` verifies `ResolvedRule.Controls` contains declared controls in `gradusctl resolve --json`.
2. `R1.AC2` WHEN a lower-level rule overrides a higher-level rule that declares `controls:`, `resolve.Resolve` SHALL preserve the parent rule's controls and merge any additional controls declared by the overriding rule without duplicates.
   - Acceptance check: `TestAuditCoverageAndExport` verifies overridden rules retain parent controls plus child controls.

### R2 Framework Coverage Report and Structured Audit Export

**User Story:** As an auditor or security lead, I want to generate a framework coverage report and a self-contained audit export bundle (JSON or Markdown) from `gradusctl` without relying on any external server.

#### Acceptance Criteria

1. `R2.AC1` WHEN `audit.BuildBundle` processes a resolved view and local evidence directory, it SHALL group active rules by framework prefix (`<FRAMEWORK>:<CONTROL>`), report covered controls, list unmapped active rules, include active exceptions and local evidence records, and support optional case-insensitive filtering by framework name.
   - Acceptance check: `TestAuditCoverageAndExport` verifies framework grouping, unmapped rule reporting, framework filtering, and Markdown rendering.
2. `R2.AC2` WHEN `gradusctl audit coverage` or `gradusctl audit export` is invoked with `--json`, `--format markdown|json`, `--framework`, or `--output <path>`, the CLI SHALL emit the requested report and exit `0` (or `2` on invalid format/technical error).
   - Acceptance check: `TestAuditCLICommands` verifies `audit coverage` and `audit export` in JSON and Markdown modes, including `--output` file writing and `--framework` filtering.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary invoking `git` via `os/exec` (`PRJ-GO-001`).
2. `NFR2` Audit exports SHALL never include individual developer activity metrics (Design Principle P16) and all repository files SHALL be in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` Existing `gradus/v1` rule frontmatter and resolved-view consumers SHALL remain 100% backward compatible (`controls` is optional and omitted when empty).
2. `C2` All existing unit and integration tests SHALL pass without regressions.
