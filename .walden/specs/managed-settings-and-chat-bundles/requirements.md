---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:01:17Z
last_modified: 2026-11-25T09:01:17Z
approved_fingerprint: sha256:73cbc5775fde1679979c3077f2990539b77da2a71eb03d71e56b0f85a3a96f5b
---

# Requirements Document

## Introduction

Feature `F26` (`managed-settings-and-chat-bundles`) implements Design §20 and §24.1 for Phase 3 ("Organization"): generating enterprise managed settings (`managed-settings.json` for Claude Code, translating permissions, static hooks, and forbidden MCP servers from `toolchain.forbidden`) and packaging reproducible ZIP skill bundles for chat agents (`claude.ai` Team/Enterprise and compatible chat products).

## Requirements

### R1 Enterprise Managed Settings Generation

**User Story:** As a platform or IT security administrator, I want `gradusctl compile --managed-settings <path>` to generate an enterprise `managed-settings.json` from the resolved pyramid (permissions, static hooks, and forbidden MCP servers) without altering repository-local drift checks.

#### Acceptance Criteria

1. `R1.AC1` WHEN `compile.RenderClaudeManagedSettings` processes a `ResolvedView` containing active `permission` rules and `toolchain.forbidden` entries of `type: "mcp"`, it SHALL emit valid JSON containing `hooks` (`PreToolUse` `gradusctl guard`, `SessionStart` `gradusctl status --brief --hook`), `permissions` (`deny`/`ask`), and `deniedMcpServers` mapped from forbidden MCP IDs.
   - Acceptance check: `TestManagedSettingsGeneration` verifies hooks, permissions, and `deniedMcpServers` in generated JSON.
2. `R1.AC2` WHEN `gradusctl compile --managed-settings <path>` is invoked, the CLI SHALL write the generated `managed-settings.json` to `<path>` and exit `0` while keeping standard repository `verify` drift checks unaffected.
   - Acceptance check: `TestManagedSettingsAndChatBundleCLI` verifies `--managed-settings` writes the file cleanly and `gradusctl verify` passes.

### R2 Reproducible Chat Agent Skill Bundle ZIP

**User Story:** As an organization admin provisioning skills to chat products (e.g. claude.ai Team/Enterprise), I want `gradusctl compile --chat-bundle <path.zip>` to produce a deterministic, portable ZIP archive containing the embedded Gradus skills, compiled shared rules (`RULES.md`, excluding personal `user` rules), and a `manifest.json`.

#### Acceptance Criteria

1. `R2.AC1` WHEN `compile.WriteChatBundleZIP` packages a `ResolvedView` and `skillsFS`, it SHALL produce a byte-reproducible ZIP archive (sorted entries, normalized zero timestamps) containing `manifest.json`, `RULES.md` (excluding `user`-level rules), and all embedded `gradus-*/SKILL.md` files.
   - Acceptance check: `TestChatBundleZIPReproducibility` verifies two runs produce identical SHA-256 hashes, `user` rules are excluded from `RULES.md`, and all skills plus `manifest.json` are present.
2. `R2.AC2` WHEN `gradusctl compile --chat-bundle <path.zip>` is invoked, the CLI SHALL write the ZIP bundle to `<path.zip>`, report bundle metadata in `--json` output, and exit `0`.
   - Acceptance check: `TestManagedSettingsAndChatBundleCLI` verifies `--chat-bundle` creates a readable ZIP archive and emits metadata in `--json` mode.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary using only Go standard library `archive/zip` (zero new dependencies, `PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` Standard `gradusctl compile` without `--managed-settings` or `--chat-bundle` SHALL remain 100% unchanged.
2. `C2` All existing unit and integration tests SHALL pass without regressions.
