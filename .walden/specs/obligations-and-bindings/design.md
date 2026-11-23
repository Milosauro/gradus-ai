---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T13:57:03Z
last_modified: 2026-11-23T13:57:03Z
approved_fingerprint: sha256:bd8cd863e1e028e5dc0a36da4eb671441506bd6e608e430e3cbdb03227623976
source_requirements_approved_at: 2026-11-23T13:54:43Z
source_requirements_fingerprint: sha256:61a300d0fbca174cde82c47ff984b1f1b7fa3354a717711eb94acf3c95719306
---

# Feature Design

## Architecture

This feature adds abstract `obligations` and concrete `bindings` (`docs/design/gradus-design.md` §19) across three existing packages with zero new third-party dependencies:

1. **Schema & Validation (`internal/schema/types.go`, `internal/schema/validate_config.go`, `spec/v1/schema/`):**
   - Define `Obligation` (`ID`, `Rule`, `AllowedProviders []string`, `Scope []string`) and `Binding` (`Obligation`, `Provider`, `Requires map[string]string`, `Scope []string`).
   - Add optional `Obligations []Obligation` and `Bindings []Binding` slices to both `schema.LevelConfig` (`level.yaml`) and `schema.PyramidConfig` (`gradus.yaml`), plus matching JSON Schema properties in `spec/v1/schema/gradus.schema.json`, `level.schema.json`, and `resolved-view.schema.json`.
   - Validate in `ValidatePyramidConfig` and `ValidateLevelConfig`: every obligation must have non-empty `id` and `rule` with no duplicate `id` within the same file; every binding must have non-empty `obligation` and `provider`.

2. **Resolution & Restriction Invariant (`internal/resolve/types.go`, `internal/resolve/obligations.go`, `internal/resolve/resolve.go`):**
   - Add `ResolvedObligation` (`ID`, `Rule`, `Level`, `AllowedProviders`, `Scope`, `Bindings`, `Satisfied`) and `ResolvedBinding` (`Obligation`, `Provider`, `Level`, `Requires`, `Scope`) to `ResolvedView` with `omitempty` JSON tags so existing pyramids without obligations/bindings keep their exact `view_hash` (`NFR2`).
   - Implement `resolveObligationsAndBindings(cfg schema.PyramidConfig, steps []PyramidStep) ([]ResolvedObligation, []ResolvedBinding, []Conflict)` in `internal/resolve/obligations.go`:
     - Walk levels in pyramid order (highest authority to most specific), then `gradus.yaml` top-level declarations (treated as `project` / pyramid-level).
     - When a lower level re-declares an existing obligation ID:
       - If the upper level declared non-empty `allowed_providers` and the lower level omits `allowed_providers` or introduces any provider not in the upper level's set, emit a blocking `Conflict` (`kind: "obligation"`) for widening `allowed_providers`.
       - Otherwise narrow `allowed_providers` to the intersection (or the lower level's restricted list when upper was empty).
     - Match each binding to its target obligation:
       - Unknown obligation ID -> blocking `Conflict` (`kind: "obligation"`).
       - `binding.provider` not in effective `allowed_providers` (when non-empty) -> blocking `Conflict` (`kind: "obligation"`).
       - Valid binding -> attach to the obligation's `Bindings` slice and mark `Satisfied = true`.
     - After attaching all bindings, any obligation with `!Satisfied` emits a blocking `Conflict` (`kind: "obligation"`).
     - Sort obligations by `ID`, `AllowedProviders` alphabetically, and bindings by `(Obligation, Provider, Level)` for deterministic `view_hash` (`NFR1`).

3. **Handbook Rendering (`internal/compile/handbook.go`):**
   - When `len(view.Obligations) > 0 || len(view.Bindings) > 0`, append an `## Obligations & Bindings` section listing each obligation's ID, declaring level, rule text, allowed providers, bound provider(s), `requires` (sorted keys), and `scope`.

## Options Considered

- **Chosen:** Model `obligations` and `bindings` as structured YAML lists directly on `level.yaml` and `gradus.yaml`, resolved in a dedicated pure helper `internal/resolve/obligations.go`.
  - Matches `docs/design/gradus-design.md` §19 syntax verbatim, keeps `resolve.go` focused, and provides clean `ResolvedView.Obligations` / `ResolvedView.Bindings` inputs for `F17` adapters (`walden`, `spec-kit`).
- **Alternative:** Store obligations as Markdown files under `rules/` with special frontmatter.
  - Rejected: contradicts §19 YAML schema (`obligations:` / `bindings:`) and conflates process obligations with individual agent rules.

## Simplicity And Elegance Review

- Isolating obligation/binding resolution in `internal/resolve/obligations.go` keeps `resolve.Resolve` minimal (one function call) and makes every invariant unit-testable in isolation.
- Using `omitempty` on `ResolvedView.Obligations` and `ResolvedView.Bindings` guarantees zero impact on `view_hash` or compiled outputs of existing repositories.

## Failure Modes And Tradeoffs

- **Unbound obligation in a newly adopting repo:** By design (`R2.AC3`), an unbound obligation produces a blocking `obligation` conflict so teams cannot silently ignore an upstream process requirement; teams satisfy it by adding a `bindings:` entry in `gradus.yaml` or their project `level.yaml`.
- **Provider CLI version verification (`requires: { cli: ">=1.4" }`):** In `F16`, `requires` is parsed, preserved in `ResolvedView`, and displayed in `handbook`; executing provider CLI probes and verifying provider evidence belongs to `F17` (`adapters-walden-and-speckit`).

## Verification Plan

- Unit tests in `internal/schema/obligations_test.go` covering valid/invalid `obligations` and `bindings` in `gradus.yaml` and `level.yaml`, plus `spec/v1` JSON Schema validation (`R1.AC1`, `R1.AC2`, `C1`).
- Unit tests in `internal/resolve/obligations_test.go` covering satisfied bindings, disallowed providers, unknown obligations, widening `allowed_providers`, narrowing `allowed_providers`, unbound obligations, deterministic ordering, and unchanged `view_hash` when absent (`R2.AC1`–`R2.AC3`, `R3.AC1`, `NFR1`, `NFR2`).
- CLI/compile test in `internal/cli/handbook_test.go` verifying `gradusctl resolve --json`, `gradusctl verify`, and `gradusctl handbook` output for pyramids with obligations and bindings (`R2.AC2`, `R3.AC1`, `R3.AC2`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/schema/types.go`, `internal/schema/validate_config.go`, `spec/v1/schema/*.json` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/resolve/obligations.go`, `internal/resolve/resolve.go` (`R2.AC1`–`R2.AC3`) |
| `R3` | `internal/resolve/types.go`, `internal/compile/handbook.go` (`R3.AC1`, `R3.AC2`) |
| `NFR1` | Deterministic sorting of obligations, providers, bindings, and `requires` keys in `internal/resolve/obligations.go` and `internal/compile/handbook.go` |
| `NFR2` | `omitempty` JSON tags on `ResolvedView.Obligations` and `ResolvedView.Bindings` verified by existing `resolve` tests |
| `C1` | Backward-compatible optional properties in `spec/v1/schema/gradus.schema.json`, `level.schema.json`, `resolved-view.schema.json` |
| `C2` | Stdlib-only implementation with `CGO_ENABLED=0` build |
