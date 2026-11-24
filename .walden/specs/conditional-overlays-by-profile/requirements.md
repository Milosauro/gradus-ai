---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T16:42:52Z
last_modified: 2026-11-24T16:42:52Z
approved_fingerprint: sha256:2934348402470723fe717d0ab14da2dc2dbd46da10a65d23b8259f3f67c48c82
---

# Requirements Document

## Introduction

Feature `F23` (`conditional-overlays-by-profile`) initiates Phase 3 ("Organization") by enabling deterministic, profile-driven conditional activation of overlays (`activation: profile`, `level.yaml` `activation_rules`, and `overlay.yaml` `when` predicates) against extensible repository `profile` facts (`data`, `exposure`, `domain`, `client`, plus custom keys per D-62).

## Requirements

### R1 Extensible Profile and Activation Rule Schema

**User Story:** As an organization architect, I want to declare extensible `profile` attributes in `gradus.yaml`, structured `activation_rules` in `level.yaml`, and optional `when` predicates in `overlay.yaml` with full schema validation and zero false unknown-key warnings.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradus.yaml` declares standard `profile` fields (`data`, `exposure`, `domain`, `client`) and arbitrary custom attributes (e.g. `region`, `regulated`, `stack`), `schema.ParsePyramidConfig` SHALL preserve all standard and custom profile attributes without emitting unknown-key errors.
   - Acceptance check: `TestProfileAndActivationRuleParsing` verifies standard and custom keys round-trip through `Profile`.
2. `R1.AC2` WHEN `level.yaml` declares `activation_rules` entries (`overlay`, `when`, `required`) or `overlay.yaml` declares `when`, the schema parsers SHALL validate required fields (`overlay` and non-empty `when`) and emit zero unknown-key warnings.
   - Acceptance check: `TestProfileAndActivationRuleParsing` verifies valid and invalid `activation_rules` and `overlay.yaml` `when` parsing.

### R2 Profile-Driven Overlay Activation and Mandatory Enforcement

**User Story:** As a compliance lead, I want overlays with `activation: profile` to activate automatically when a repository's `profile` matches `activation_rules` or `overlay.yaml` `when`, and I want `required: true` activation rules to block when a required overlay is missing or disabled.

#### Acceptance Criteria

1. `R2.AC1` WHEN an overlay has `activation: profile` and a `level.yaml` `activation_rules` entry or `overlay.yaml` `when` predicate matches `gradus.yaml`'s `profile` (AND across predicate keys, OR across list values, case-insensitive string comparison), `resolve.Resolve` SHALL mark the overlay `active: true`, include its rules in the resolved view respecting `strength` order, and record a human-readable activation `reason`.
   - Acceptance check: `TestConditionalOverlayActivation` verifies matching overlays activate with rules merged at declared `strength` and non-matching overlays remain `active: false` without loading their rules.
2. `R2.AC2` IF a level's `activation_rules` entry has `required: true` and matches `gradus.yaml`'s `profile`, AND the referenced overlay is absent from `gradus.yaml` or explicitly set to `activation: disabled`, THEN `resolve.Resolve` SHALL emit a blocking conflict (`kind: "overlay"`) explaining which level requires the overlay.
   - Acceptance check: `TestConditionalOverlayActivation` verifies missing or disabled mandatory overlays produce blocking `overlay` conflicts (exit code `1` in `gradusctl verify`).

### R3 JSON Schemas and CLI Integration

**User Story:** As a developer, I want `spec/v1/schema/*.json`, `gradusctl resolve --json`, and `gradusctl verify --json` to reflect conditional overlay activation and reasons cleanly.

#### Acceptance Criteria

1. `R3.AC1` WHEN `spec/v1/schema/level.schema.json`, `overlay.schema.json`, `gradus.schema.json`, and `resolved-view.schema.json` are validated by `spec/v1` schema tests, they SHALL accept structured `activation_rules`, `overlay.yaml` `when`, extensible `profile`, and `overlays[].reason`.
   - Acceptance check: `go test -v -count=1 ./spec/v1` and `TestConditionalOverlaysCLI` pass.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary invoking `git` via `os/exec` (`PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` Existing `activation: always`, `explicit`, `disabled`, and omitted `activation` behavior SHALL remain 100% backward compatible.
2. `C2` All existing unit and integration tests SHALL pass without regressions.
