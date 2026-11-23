---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T14:57:55Z
last_modified: 2026-11-23T14:57:55Z
approved_fingerprint: sha256:d53e93ee0f8ca36b6607cb46b3e2733000d310da412cbdc35d210d247651fed7
---

# Requirements Document

## Introduction

Phase 2 ("Business unit") introduces **Toolchain Governance** (`docs/design/gradus-design.md` §19, §24.1). Levels can declare skills, CLI tools, MCP servers, and plugins as `required`, `recommended`, or `forbidden`. Lower levels may add items, never remove an inherited `required` item or enable a `forbidden` one. This feature defines the `gradus/v1` toolchain schema, monotonic pyramid resolution in `ResolvedView`, and the `gradusctl toolchain check` CLI command.

## Requirements

### R1 Toolchain Schema and Validation

**User Story:** As a security or platform lead, I want to declare `required`, `recommended`, and `forbidden` skills, tools, MCP servers, and plugins in `level.yaml` and `gradus.yaml`, so that team toolchains follow a governed baseline.

#### Acceptance Criteria

1. `R1.AC1` WHEN a `level.yaml` or `gradus.yaml` declares a `toolchain` block with `required`, `recommended`, and/or `forbidden` lists of items (`type`: `skill` | `tool` | `mcp` | `plugin`, `id`, optional `reason`), the system SHALL parse and preserve them in `schema.LevelConfig` and `schema.PyramidConfig` while maintaining `gradus/v1` JSON Schema validity.
   - Acceptance check: Valid `level.yaml` and `gradus.yaml` files containing `toolchain` declarations parse cleanly and pass `spec/v1` JSON Schema validation.
2. `R1.AC2` IF a toolchain item omits `type` or `id`, or specifies a `type` outside `skill | tool | mcp | plugin`, THEN configuration parsing SHALL fail with exit code `2` and a clear diagnostic.
   - Acceptance check: Parsing a config with `type: unknown` or empty `id` returns a validation error and exits with code `2`.

### R2 Monotonic Pyramid Resolution

**User Story:** As an engineering manager, I want `gradusctl resolve` and `gradusctl verify` to enforce that lower levels never enable a forbidden tool or remove a required one, so that upper-level security boundaries cannot be weakened.

#### Acceptance Criteria

1. `R2.AC1` WHEN `resolve.Resolve` evaluates a pyramid with `toolchain` declarations across multiple levels, the system SHALL union all `required`, `recommended`, and `forbidden` entries in pyramid order and expose them deterministically in `ResolvedView.Toolchain` (`omitempty` when absent).
   - Acceptance check: Upper-level `required` and `forbidden` items are preserved alongside lower-level additions in `gradusctl resolve --json`, while pyramids without `toolchain` keep their exact `view_hash`.
2. `R2.AC2` IF a level declares a `required` or `recommended` item whose `type` matches and whose `id` matches any `forbidden` pattern (glob matching via `doublestar`) from the same or an upper level, THEN `resolve.Resolve` SHALL emit a blocking `Conflict` (`kind: "toolchain"`) causing `gradusctl verify` and `gradusctl compile` to exit with code `1`.
   - Acceptance check: Declaring `forbidden: [{type: mcp, id: "*-public-pastebin"}]` at `team` and `required: [{type: mcp, id: "acme-public-pastebin"}]` at `project` produces a `toolchain` conflict and exits `1` on `verify`.

### R3 Toolchain Inspection CLI (`gradusctl toolchain check`)

**User Story:** As a developer or CI pipeline, I want `gradusctl toolchain check [--json]` to verify that all required skills/tools/MCPs/plugins are present and no forbidden items are configured in the workspace, so that non-compliant environments are caught automatically.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl toolchain check [--dir .] [--json]` runs, the CLI SHALL inspect the workspace for installed skills (`.claude/skills/*/SKILL.md`, `.agents/skills/*/SKILL.md`), CLI tools (`PATH` lookup), MCP servers (`.mcp.json`, `.claude/settings.json` `mcpServers`, `.agents/mcp.json`), and plugins (`.claude/settings.json` `enabledPlugins`), exiting with code `0` when all `required` items are present and no `forbidden` item matches.
   - Acceptance check: A workspace with all required skills/tools installed and no forbidden MCP/skill/plugin exits `0` with `verdict: "pass"`.
2. `R3.AC2` IF any `required` item is missing or any detected item matches a `forbidden` pattern, THEN `gradusctl toolchain check` SHALL report the violations in text or JSON output and exit with code `1` (suppressing stderr `Error:` when `--json` is set).
   - Acceptance check: A missing required skill or a configured forbidden MCP server in `.mcp.json` causes `gradusctl toolchain check --json` to exit `1` with valid JSON and `verdict: "fail"`.
3. `R3.AC3` WHEN a `recommended` item is missing, `gradusctl toolchain check` SHALL report it as a non-blocking warning and exit with code `0` (assuming no `required`/`forbidden` violations exist).
   - Acceptance check: A missing recommended tool appears in `warnings` while exit code remains `0`.

## Non-Functional Requirements

- `NFR1` **Deterministic Ordering & Backward Compatibility:** `ResolvedView.Toolchain` slices MUST be sorted deterministically and omitted (`omitempty`) when no toolchain declarations exist (`E1`, `C1`).
- `NFR2` **Static Build & Zero Network I/O:** `gradusctl toolchain check` MUST perform local filesystem/`PATH` inspection only with `CGO_ENABLED=0` and zero new third-party dependencies.

## Constraints And Dependencies

- `C1` Must preserve `gradus/v1` backward compatibility across `spec/v1/schema/gradus.schema.json`, `level.schema.json`, and `resolved-view.schema.json`.
- `C2` Must reuse `github.com/bmatcuk/doublestar/v4`, `internal/schema`, and `internal/resolve`.

## Out Of Scope

- Organization-wide managed settings / MDM provisioning files (Phase 3).
- Interactive security auditing of third-party skills (`gradus-link`) — covered by `F19`.
