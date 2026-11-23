---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T14:58:56Z
last_modified: 2026-11-23T14:58:56Z
approved_fingerprint: sha256:7a5b1d6b09c85ce251e643cec23b31c8d8bde607770500c7fa669a9ccee4b5cf
source_requirements_approved_at: 2026-11-23T14:57:55Z
source_requirements_fingerprint: sha256:d53e93ee0f8ca36b6607cb46b3e2733000d310da412cbdc35d210d247651fed7
---

# Feature Design

## Architecture

This feature adds monotonic toolchain governance (`docs/design/gradus-design.md` §19) across three focused layers:

1. **Schema & Validation (`internal/schema/types.go`, `internal/schema/validate_config.go`, `spec/v1/schema/`):**
   - Add `ToolchainItem` (`Type`, `ID`, `Reason`) and `ToolchainConfig` (`Required`, `Recommended`, `Forbidden` slices of `ToolchainItem`).
   - Add optional `Toolchain *ToolchainConfig` (`yaml:"toolchain,omitempty" json:"toolchain,omitempty"`) to `schema.PyramidConfig` and `schema.LevelConfig`, plus matching JSON Schema properties in `spec/v1/schema/gradus.schema.json`, `level.schema.json`, and `resolved-view.schema.json`.
   - Validate in `ValidatePyramidConfig` and `ValidateLevelConfig`: each item's `type` must be one of `skill`, `tool`, `mcp`, `plugin` and `strings.TrimSpace(id) != ""`.

2. **Monotonic Pyramid Resolution (`internal/resolve/types.go`, `internal/resolve/toolchain.go`, `internal/resolve/resolve.go`):**
   - Add `ResolvedToolchainItem` (`Type`, `ID`, `Level`, `Reason`) and `ResolvedToolchain` (`Required`, `Recommended`, `Forbidden`) to `ResolvedView` as `Toolchain *ResolvedToolchain` with `json:"toolchain,omitempty"` so pyramids without `toolchain` keep their exact `view_hash` (`NFR1`).
   - Implement `resolveToolchain(cfg schema.PyramidConfig, steps []PyramidStep) (*ResolvedToolchain, []Conflict)` in `internal/resolve/toolchain.go`:
     - Walk levels in pyramid order then `gradus.yaml` top-level `cfg.Toolchain` (`level = "project"`).
     - Accumulate `Required`, `Recommended`, and `Forbidden` items (deduplicating identical `(type, id)` pairs per list, preserving earliest declaring level).
     - After collecting all `Forbidden` patterns, check every `Required` and `Recommended` item against every `Forbidden` item of the same `Type` using `doublestar.Match(forbidden.ID, item.ID)`: any match emits a blocking `Conflict` (`kind: "toolchain"`).
     - Sort all three slices deterministically by `(Type, ID, Level)` (`NFR1`).

3. **Workspace Inspection CLI (`internal/toolchain/check.go`, `internal/cli/toolchain.go`, `internal/cli/root.go`):**
   - `toolchain.Check(rootDir string, tc *resolve.ResolvedToolchain) Report`:
     - Discovers local workspace inventory with zero network I/O (`NFR2`):
       - **Skills:** directory names under `.claude/skills/` and `.agents/skills/` containing `SKILL.md`.
       - **Tools:** `exec.LookPath(id)` for each declared `tool` item (plus matching `forbidden` tool names via `exec.LookPath` when `id` has no glob wildcards).
       - **MCP servers:** keys of `mcpServers` in `.mcp.json`, `.claude/settings.json`, and `.agents/mcp.json` (when present).
       - **Plugins:** enabled entries (`true`) of `enabledPlugins` in `.claude/settings.json` (when present).
     - Evaluates:
       - Missing `required` items -> `Findings` (`kind: "missing_required"`).
       - Detected items matching `forbidden` patterns via `doublestar.Match` -> `Findings` (`kind: "forbidden_present"`).
       - Missing `recommended` items -> `Warnings` (non-blocking, `R3.AC3`).
     - Returns `Report{Verdict: "pass"|"fail", Findings, Warnings, Checked}`.
   - CLI `gradusctl toolchain check [--dir .] [--json]`:
     - Resolves the pyramid (`resolve.Resolve`), runs `toolchain.Check`, outputs JSON or human-readable report, exits `0` on `pass`, `1` on `fail` (suppressing stderr `Error:` when `--json` is set), and `2` on config/resolve errors.

## Options Considered

- **Chosen:** Pure local inspection (`internal/toolchain/check.go`) + monotonic `ResolvedView.Toolchain` resolution (`internal/resolve/toolchain.go`).
  - Zero network calls, cross-platform (`os.ReadDir`, `exec.LookPath`, `encoding/json`), directly reuses `doublestar` and `internal/resolve`.
- **Alternative:** Execute external agent CLIs to query installed plugins/MCPs.
  - Rejected: agent CLIs are not always installed in CI containers, add latency, and lack uniform machine-readable output.

## Simplicity And Elegance Review

- Keeping toolchain discovery in a small, pure package `internal/toolchain` makes it trivial to unit-test with temporary directories and zero external mocks.
- Using `omitempty` on `*ResolvedToolchain` guarantees zero `view_hash` drift on existing repositories.

## Failure Modes And Tradeoffs

- **Glob-pattern `forbidden` CLI tools (e.g. `*-pastebin`):** Scanning every binary across all `PATH` directories is expensive and noisy; `toolchain.Check` checks exact `forbidden` tool names via `exec.LookPath` while full glob matching applies to `skill`, `mcp`, and `plugin` inventories (where workspace lists are finite) and always applies during `resolve` invariant checks (`R2.AC2`).
- **Malformed `.mcp.json` / `.agents/mcp.json`:** Treated as a warning/empty inventory rather than crashing unrelated checks, while `gradus.yaml` / `level.yaml` errors fail closed with exit `2`.

## Verification Plan

- Unit tests in `internal/schema/toolchain_test.go` covering valid/invalid `toolchain` blocks in `gradus.yaml` and `level.yaml` plus `spec/v1` JSON Schema validation (`R1.AC1`, `R1.AC2`, `C1`).
- Unit tests in `internal/resolve/toolchain_test.go` covering union across levels, deterministic sorting, `omitempty` when absent, and `toolchain` conflict when a lower level enables a `forbidden` pattern (`R2.AC1`, `R2.AC2`, `NFR1`).
- Unit and CLI tests in `internal/toolchain/check_test.go` and `internal/cli/toolchain_test.go` exercising `gradusctl toolchain check [--json]` for pass, missing required skill/tool (exit 1), forbidden MCP/plugin present (exit 1), missing recommended warning (exit 0), and `CGO_ENABLED=0` build (`R3.AC1`–`R3.AC3`, `NFR2`, `C2`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/schema/types.go`, `internal/schema/validate_config.go`, `spec/v1/schema/*.json` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/resolve/toolchain.go`, `internal/resolve/types.go`, `internal/resolve/resolve.go` (`R2.AC1`, `R2.AC2`) |
| `R3` | `internal/toolchain/check.go`, `internal/cli/toolchain.go` (`R3.AC1`–`R3.AC3`) |
| `NFR1` | Deterministic sorting and `omitempty` pointer `*ResolvedToolchain` in `internal/resolve/toolchain.go` |
| `NFR2` | Local filesystem/`LookPath` inspection only in `internal/toolchain/check.go`; `CGO_ENABLED=0` build |
| `C1` | Backward-compatible optional properties in `spec/v1/schema/{gradus,level,resolved-view}.schema.json` |
| `C2` | Reuses `doublestar`, `internal/schema`, and `internal/resolve` |
