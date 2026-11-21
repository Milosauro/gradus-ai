---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:43:32Z
last_modified: 2026-11-21T17:43:32Z
approved_fingerprint: sha256:ad2ebcd789f60fa649f2d671e1419a79659c02f9b4822708eb90a0dda484f597
source_requirements_approved_at: 2026-11-21T17:42:31Z
source_requirements_fingerprint: sha256:17ef5367df3c7b5743b97df94bb4cfae3fc328f4938cb8ea075778a3bf52af43
---

# Feature Design

## Architecture

### 1. Deterministic Handbook Renderer (`internal/compile/handbook.go`, `internal/cli/handbook.go`)
- **`compile.RenderHandbook(view resolve.ResolvedView) string`:**
  - Pure, deterministic function (no timestamps, sorted level/rule iteration) rendering a human-readable Markdown onboarding handbook from `resolve.ResolvedView`:
    1. Title & `view_hash` metadata.
    2. **Pyramid Levels & Charters:** ordered list of levels showing `id`, `version`, `charter.purpose`, `includes`, `excludes`, and `allowed_severities`.
    3. **Active Overlays:** list of active overlays and their strength (when present).
    4. **Active Rules by Level:** grouped in pyramid order; each rule shows `ID`, `Title`, `Severity`, `Enforcement`, `Scope`, `Provenance.Source`, `Text`, and `Why` (when present).
    5. **Exceptions & Break-Glass Procedure:** standard reference to `gradusctl exception request`, `approve`, `break-glass`, `list`, and `expire`.
- **CLI (`internal/cli/handbook.go`, registered in `root.go`):**
  - `gradusctl handbook [--dir .] [--output <path>] [--json]` resolves the pyramid via `resolve.Resolve`, renders Markdown via `compile.RenderHandbook`, optionally writes `--output` (creating parent dirs), and emits either raw Markdown or structured JSON (`view_hash`, `markdown`, `levels`, `rules_count`, `output_path`).

### 2. Non-Prescriptive Starter Templates (`examples/`)
- Three self-contained, non-prescriptive starter pyramids (each with a `README.md` noting it is an adaptable example per `X-11`):
  1. `examples/personal/`: 1-level `project` pyramid with `gradus.yaml`, `level.yaml` (charter), and 2 sample rules (`PRJ-GIT-001` hook, `PRJ-DBG-001` pattern check) with `### Compliant` / `### Non-compliant` examples.
  2. `examples/team/`: 2-level (`team` + `project`) pyramid with charters, `context_budget`, and `exceptions` policy + sample rules with examples.
  3. `examples/regulated-fintech/`: 2-level (`compliance` + `project`) + explicit `payments` overlay with strict `anti_tamper.mode: deny`, `break_glass: false` on `compliance`, and sample rules with examples.

## Options Considered

- **Chosen: Pure deterministic `compile.RenderHandbook(view)` in `internal/compile/handbook.go`.**
  - Guarantees byte-identical output across repeated runs (`NFR2`) and keeps rendering decoupled from CLI I/O.
- **Alternative rejected: Embed Go `text/template` files for handbook rendering.**
  - Rejected because a small `strings.Builder` renderer has zero template-escaping surprises and is trivial to test and maintain.

## Simplicity And Elegance Review

- Reuses `resolve.ResolvedView` directly (already carrying `Levels[].Charter`, `Overlays`, and `Rules`), avoiding any duplicate file parsing.
- Starter templates in `examples/` are validated automatically by the same `resolve` and `gradusctl test` commands that users run.

## Failure Modes And Tradeoffs

- **Unwritable `--output` path:** Fails closed with exit `2` and an actionable error message.
- **No timestamps in handbook body:** Omitting wall-clock timestamps guarantees byte-identical determinism (`NFR2`) so teams can commit `HANDBOOK.md` and check it for drift.

## Verification Plan

- `TestHandbookCommand` in `internal/cli/handbook_test.go` verifying stdout Markdown, `--output` file creation, byte-identical determinism across two runs, and `--json` output fields.
- `TestExampleTemplates` in `internal/cli/handbook_test.go` running `resolve.Resolve` (0 conflicts) and `gradusctl test` (0 failures) on `examples/personal`, `examples/team`, and `examples/regulated-fintech`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `compile.RenderHandbook` in `internal/compile/handbook.go` and `internal/cli/handbook.go` verified by `TestHandbookCommand` |
| `R2` | `R2.AC1`: `examples/personal/`, `examples/team/`, and `examples/regulated-fintech/` verified by `TestExampleTemplates` |
| `NFR1` | Canonical English across handbook/templates and `CGO_ENABLED=0` static build |
| `NFR2` | Pure deterministic rendering verified by byte-equality check in `TestHandbookCommand` |
| `C1` | Explicit non-prescriptive disclaimer in each `examples/*/README.md` |
