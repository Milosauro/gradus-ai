---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-19T17:43:32Z
last_modified: 2026-11-19T17:43:32Z
approved_fingerprint: sha256:d10e8cd72d2da81b700126d042932fb121d09a194a2b8204c406c320f19b281e
source_requirements_approved_at: 2026-11-19T17:22:13Z
source_requirements_fingerprint: sha256:660f966777e8e9aed853659fb6d02260dd3245f9ba3395ed9726bd35629cc63c
---

# Feature Design

## Architecture

Feature **F1 (`core-schema-init`)** introduces three cohesive layers:

1. **Open Specification & JSON Schemas (`spec/v1/`):**
   - `spec/v1/SPEC.md` documents the `gradus/v1` format for `gradus.yaml`, `level.yaml`, `overlay.yaml`, rule Markdown frontmatter, `gradus.lock`, resolved views, evidence, lessons, and exceptions.
   - `spec/v1/schema/*.schema.json` (9 Draft 2020-12 JSON Schema files) define machine-readable contracts. Schema tests validate both the schemas themselves and canonical fixtures.

2. **Schema & Rule Parser (`internal/schema/`):**
   - `types.go`: Typed Go structs (`PyramidConfig`, `LevelRef`, `SourceRef`, `OfflinePolicy`, `OverlayRef`, `Profile`, `LevelConfig`, `Charter`, `Constraints`, `ExceptionPolicy`, `OverlayConfig`, `Rule`, `OverridableSpec`, `CheckerConfig`, `GuardConfig`, `ExamplePair`, `ParseResult[T]`). `OverridableSpec` implements `UnmarshalYAML` to accept either a boolean (`false`/`true`) or an object with constraints (§9.4).
   - `parse.go`:
     - Uses `gopkg.in/yaml.v3` (`yaml.Node` tree inspection) so every struct parser walks mapping keys against its known field set and records any unknown key path (e.g., `levels[0].custom_ext`) into `Notes []string` instead of failing (forward compatibility, `R2.AC3`, `R3.AC4`). Reserved top-level keys (`obligations`, `bindings`, `toolchain`, `models`, `adapters` per §9.11) are recognized in the schema and emit an explicit diagnostic note (`reserved key '<key>' is not active in Phase 0`).
     - `ParseRuleFile(path, derivedLevel, sourceLabel)`: splits YAML frontmatter (`---` delimiters) from the Markdown body, parses frontmatter via `yaml.Node`, sets `Rule.Level = derivedLevel` strictly from the caller's level container (ADR 0004, `R3.AC3`), and parses body sections (`statement` = first non-empty paragraph before any `##` heading, `## Why`, `## Rationalizations`, `## Examples` with `### Compliant` and `### Non-compliant` fenced code blocks).
   - `validate.go`:
     - Validates `gradus.yaml` (`schema == "gradus/v1"`, non-empty `levels`, unique IDs matching `^[a-z][a-z0-9-]*$`).
     - Validates `level.yaml` and `overlay.yaml` (`schema == "gradus/v1"`, valid `id`, valid `strength` for overlays).
     - Validates `Rule` (`id` matching `^[A-Z][A-Z0-9]*(-[A-Z0-9]+)+$`, non-empty `title`, `severity` in `MUST|SHOULD|MAY`, `enforcement` in `prompt|review|check|hook|permission` defaulting to `prompt`, `scope` defaulting to `["**/*"]`, `status` defaulting to `active`, non-empty `Statement`, `enforcement == "check"` requires `Checker.Name != ""`, `enforcement == "hook"` requires `Guard.Action != ""`, `Guard.Pattern != ""`, and `Guard.Decision` in `deny|ask`; if both `Checker` and `Guard` are present per ADR 0002, both are validated).
   - `loader.go`:
     - `LoadLocalPyramid(rootDir string)`: loads `gradus.yaml` and reads rules from local `source.path` levels (expanding `~/` only when `CI` env var is not set, §9.2), returning `[]Rule`, `Notes`, or an error.

3. **CLI Commands (`internal/cli/`):**
   - Typed `ExitErrorWithCode` in `internal/cli/root.go` so `Execute()` exits with `ExitViolation (1)` or `ExitError (2)` cleanly instead of always `ExitError`.
   - `internal/cli/init.go` (`gradusctl init [--force]`):
     - Checks if `gradus.yaml` exists; if present without `--force`, returns exit code `2`.
     - Writes `gradus.yaml` (`schema: gradus/v1`, `project` level, `targets: [claude-code, antigravity, agents-md]`), `.gradus/levels/project/level.yaml`, and `.gradus/levels/project/rules/PRJ-GEN-001.md` (an initial example rule).
     - Appends missing Gradus ignore entries (`.gradus/cache/`, `.gradus/evidence/`, `CLAUDE.local.md`, `.agents/rules/gradus-user-*.md`) to `.gitignore` without duplicating or removing existing lines.
     - Emits text or `--json` summary.
   - `internal/cli/explain.go` (`gradusctl explain <id>`):
     - Calls `schema.LoadLocalPyramid(".")`, searches for `<id>`, and prints metadata (including `overrides` and any lower-level rules that declare `overrides: <id>` as `overridden_by`, §16) plus body sections in human-readable format or `--json` (including `notes`). Returns exit code `2` if `<id>` is not found or config/rule parsing fails.

## Options Considered

- **Unknown-key detection:** Chose `yaml.Node` key traversal over `KnownFields(true)` unmarshaling because `KnownFields(true)` rejects unknown keys as hard errors, whereas `gradus/v1` (§9.2) requires emitting diagnostic notes without failing.
- **Markdown body parsing:** Chose a lightweight line-based section scanner (`## Why`, `## Rationalizations`, `## Examples`, `### Compliant`, `### Non-compliant`) over pulling in a full Markdown AST dependency, keeping dependencies minimal and deterministic.

## Simplicity And Elegance Review

- Zero external dependencies beyond `cobra` and `yaml.v3` (and `jsonschema/v6` only if needed in tests).
- `yaml.Node` walking is localized in a single helper `collectUnknownKeys(node *yaml.Node, allowed map[string]bool, prefix string) []string` reused across `gradus.yaml`, `level.yaml`, `overlay.yaml`, and rule frontmatter.
- `init` and `explain` share `internal/cli` JSON output and exit-code error handling without global mutable state.

## Failure Modes And Tradeoffs

- **Malformed YAML or missing frontmatter delimiters:** Returns explicit file-and-field error mapped to exit code `2`; never silently falls back to empty defaults.
- **Partial writes during `gradusctl init`:** `gradus.yaml` existence is checked upfront before creating `.gradus/` directories; `--force` is required to overwrite an existing `gradus.yaml`.
- **Local-only lookup in `explain`:** Remote `git` sources are out of scope until F2 (`resolve-and-lock`); `explain` inspects local `source.path` levels in F1 and skips uncached remote levels with a diagnostic note rather than failing offline, while still failing with exit code `2` if the requested ID is not found.

## Verification Plan

- **JSON Schema validation tests (`spec/v1/schema`):** Go tests compile all 9 `.schema.json` files and validate positive and negative fixtures.
- **Schema unit tests (`internal/schema`):** Table-driven tests covering valid configs, invalid schemas/IDs/enums, missing `checker`/`guard`, unknown YAML keys populating `Notes`, Markdown section extraction, and ADR 0004 level derivation (`ORG-SEC-004` inside `team` directory -> `level: "team"`).
- **CLI integration tests (`internal/cli`):** Tests using `t.TempDir()` for `gradusctl init` (fresh dir, existing `.gitignore` preservation, duplicate run without `--force` -> exit code `2`, `--json` output) and `gradusctl explain` (text output, `--json` output, unknown ID -> exit code `2`, invalid config -> exit code `2`).
- **Static build & race checks:** `CGO_ENABLED=0 go build -trimpath ./cmd/gradusctl`, `go test -v -race -count=1 ./...`, `go vet ./...`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `spec/v1/SPEC.md` and `spec/v1/schema/*.schema.json` validated by schema test suite |
| `R2` | `R2.AC1`–`R2.AC4`: `internal/schema` (`ParsePyramidConfig`, `ParseLevelConfig`, `ParseOverlayConfig`, `collectUnknownKeys`) |
| `R3` | `R3.AC1`–`R3.AC4`: `internal/schema` (`ParseRuleFile`, `ValidateRule`, ADR 0004 derived level, `collectUnknownKeys`) |
| `R4` | `R4.AC1`–`R4.AC3`: `internal/cli/init.go` (`gradusctl init [--force] [--json]`, `.gitignore` safe append) |
| `R5` | `R5.AC1`–`R5.AC3`: `internal/cli/explain.go` (`gradusctl explain <id> [--json]`, exit code `2` on missing ID/config) |
| `NFR1` | `filepath` cross-platform handling + `CGO_ENABLED=0` build check |
| `NFR2` | English-only docs, errors, and CLI output |
| `NFR3` | `ExitErrorWithCode` in `internal/cli` enforcing exit code `2` on errors |
