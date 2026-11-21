---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:42:31Z
last_modified: 2026-11-21T17:42:31Z
approved_fingerprint: sha256:17ef5367df3c7b5743b97df94bb4cfae3fc328f4938cb8ea075778a3bf52af43
---

# Requirements Document

## Introduction

`handbook-and-templates` (`F14`, §16 `handbook`, X-11, X-12, D-38) implements `gradusctl handbook [--output <path>] [--json]` to deterministically generate a human-readable team onboarding handbook in Markdown directly from the resolved pyramid, and ships non-prescriptive starter templates under `examples/` (`personal`, `team`, and `regulated-fintech`) verified by automated tests.

## Requirements

### R1 Human Onboarding Handbook (`gradusctl handbook`)

**User Story:** As a team lead onboarding new engineers, I want `gradusctl handbook [--output <path>] [--json]` to generate a readable Markdown onboarding handbook from the resolved pyramid, so that human documentation stays synchronized with the actual governed rules without manual copying.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl handbook [--dir <path>]` is invoked, the CLI SHALL render a deterministic Markdown handbook to stdout (or to `--output <file>` when specified) containing the pyramid hierarchy, level charters (`purpose`, `includes`, `excludes`, `allowed_severities`), active overlays, active rules grouped by level with provenance/enforcement/severity, and exception/break-glass instructions.
   - Acceptance check: Running `gradusctl handbook` on a repository with levels, a charter, and active rules outputs Markdown containing each level ID, charter purpose, rule ID, severity, enforcement, and `gradusctl exception` workflow guidance.
2. `R1.AC2` WHEN `gradusctl handbook --json` is invoked, the CLI SHALL output structured JSON containing `view_hash`, `markdown`, `levels`, `rules_count`, and optional `output_path`.
   - Acceptance check: `gradusctl handbook --json` returns valid JSON with non-empty `view_hash` and `markdown`.

### R2 Non-Prescriptive Starter Templates (`examples/`)

**User Story:** As a developer or team adopting Gradus, I want verified example pyramids under `examples/` (`personal`, `team`, `regulated-fintech`), so that I can copy and adapt a working baseline instead of starting from a blank file.

#### Acceptance Criteria

1. `R2.AC1` WHEN the repository is inspected under `examples/`, it SHALL provide `examples/personal/`, `examples/team/`, and `examples/regulated-fintech/`, each containing a valid `gradus.yaml`, `level.yaml` with a charter, and sample `rules/*.md` files with `### Compliant` and `### Non-compliant` examples.
   - Acceptance check: Automated tests resolve and run `gradusctl test` on all three example directories with zero conflicts and zero test failures (`exit 0`).

## Non-Functional Requirements

- `NFR1` **Canonical English & Static Build:** All handbook output, template files, and Go code MUST be in canonical English (`PRJ-LANG-001`) and build with `CGO_ENABLED=0` (`PRJ-GO-001`).
- `NFR2` **Deterministic Output:** Running `gradusctl handbook` twice on the same pyramid MUST produce byte-identical Markdown output.

## Constraints And Dependencies

- `C1` Starter templates in `examples/` MUST be explicitly marked as non-prescriptive starting points (per README and `gradus-design.md` X-11).

## Out Of Scope

- HTML/PDF export or static-site generators for the handbook.
