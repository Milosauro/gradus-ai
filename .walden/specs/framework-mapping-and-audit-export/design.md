---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T20:04:37Z
last_modified: 2026-11-24T20:04:37Z
approved_fingerprint: sha256:20992e038c42d504432362d25a75411c8fc24565d672027a1fed92ec67e0e48c
source_requirements_approved_at: 2026-11-24T20:04:37Z
source_requirements_fingerprint: sha256:f2897bab33e268837c9d74383408d8b0f817ef82054510884cc8c65c6d33bce3
---

# Design: Framework Mapping & Audit Export (`F24`)

## Architecture

1. **Resolved View Propagation (`internal/resolve`):**
   - Add `Controls []string` (`json:"controls,omitempty"`) to `ResolvedRule` in `internal/resolve/types.go`.
   - Copy `rule.Controls` when constructing `ResolvedRule` in `internal/resolve/resolve.go`, and union parent + child controls monotonically during override merging in `internal/resolve/invariant.go`.
2. **Audit Engine (`internal/audit`):**
   - `ControlMapping`: parses `FRAMEWORK:CONTROL` (splitting on first `:`; unprefixed controls fall under framework `"UNSPECIFIED"`).
   - `BuildBundle(rootDir string, view resolve.ResolvedView, opts Options) (AuditBundle, error)`:
     - Groups active rules by framework (sorted alphabetically) and control ID (sorted alphabetically), recording rule ID, title, level, severity, enforcement, waived status, and provenance source.
     - Collects active rules with empty `Controls` into `UnmappedRules`.
     - Loads local evidence JSON files from `.gradus/evidence/*.json` (sorted by timestamp/ID) if present.
     - Applies optional case-insensitive `opts.Framework` filter to `Frameworks`.
   - `RenderMarkdown(bundle AuditBundle) string`: renders a deterministic Markdown audit report (Metadata, Framework Coverage tables, Unmapped Rules, Active Exceptions, Recent Evidence).
3. **CLI Commands (`internal/cli/audit.go`, `internal/cli/root.go`):**
   - `gradusctl audit coverage [--dir .] [--framework <name>] [--json]`
   - `gradusctl audit export [--dir .] [--framework <name>] [--format json|markdown] [--output <file>] [--json]`

## Options Considered

- **Option A (Chosen):** Dedicated `internal/audit` package + `gradusctl audit coverage|export` subcommands reading the resolved view and `.gradus/evidence/*.json` directly. Offline-first, zero external dependencies, deterministic output.
- **Option B (Rejected):** Embed framework control catalogs (full ISO/SOC2 control lists) inside `gradusctl`. Catalogs are copyrighted/version-specific and violate Gradus's non-certification principle ("not a certification tool; helps produce evidence").

## Simplicity And Elegance Review

- Reuses existing `schema.Rule.Controls`, `resolve.Resolve`, and `evidence.Record` types without introducing any new config files or third-party packages.

## Failure Modes And Tradeoffs

- **Missing `.gradus/evidence` directory:** Treated as empty evidence slice rather than error, so fresh repos can still run `gradusctl audit coverage` and `gradusctl audit export`.
- **Invalid `--format` flag:** Fails cleanly with exit code `2`.

## Verification Plan

- Unit tests in `internal/audit/report_test.go` (`TestAuditCoverageAndExport`).
- End-to-end CLI tests in `internal/cli/audit_test.go` (`TestAuditCLICommands`).
- Static build check (`CGO_ENABLED=0 go build`) and full `go test ./...`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/resolve/types.go`, `internal/resolve/resolve.go`, `internal/resolve/invariant.go` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/audit/report.go`, `internal/cli/audit.go`, `internal/cli/root.go` (`R2.AC1`, `R2.AC2`) |
| `NFR1` | Static `CGO_ENABLED=0` binary build and `git` via `os/exec` |
| `NFR2` | Process-only audit bundle (no individual metrics) and canonical English files |
| `C1` | Optional `controls` field with `omitempty` preserving backward compatibility |
| `C2` | Full Go test suite regression pass |
