---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:24:35Z
last_modified: 2026-11-25T09:24:35Z
approved_fingerprint: sha256:582093e3a8b45597ff743fea3855b97c54f69f23c20fdcd7d5327454b5fa553a
---

# Requirements Document

## Introduction

Feature `F27` (`mcp-server`) implements Design §20 (`gradusctl serve --mcp`): a zero-dependency stdio JSON-RPC 2.0 Model Context Protocol (MCP) server exposing the resolved Gradus pyramid to chat agents and MCP-capable tools via five governed, read-only/proposal-only tools (`resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, `get_toolchain`).

## Requirements

### R1 MCP JSON-RPC 2.0 Stdio Protocol Core

**User Story:** As a developer or chat-agent user, I want `gradusctl serve --mcp` to speak standard MCP JSON-RPC 2.0 over stdin/stdout so any MCP client can connect directly without an external runtime or server daemon.

#### Acceptance Criteria

1. `R1.AC1` WHEN an MCP client sends `initialize`, `notifications/initialized`, `ping`, or `tools/list` JSON-RPC 2.0 requests over stdio, `mcp.Server` SHALL respond with valid MCP protocol metadata (`protocolVersion`, `serverInfo`, `capabilities.tools`) and list all five Gradus tools with JSON Schemas.
   - Acceptance check: `TestMCPServerProtocolAndTools` verifies `initialize`, `ping`, and `tools/list` return all 5 tools.
2. `R1.AC2` WHEN an MCP client sends an unknown method, malformed JSON, or unknown tool name, `mcp.Server` SHALL return a structured JSON-RPC 2.0 error (or MCP `isError: true` tool result) without crashing or exiting prematurely.
   - Acceptance check: `TestMCPServerProtocolAndTools` verifies unknown method/tool handling returns structured errors and keeps the server alive.

### R2 Governed Pyramid Tools (`resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, `get_toolchain`)

**User Story:** As an organization architect, I want MCP clients to inspect rules, check text snippets, draft charter-checked rule proposals, and query the declared toolchain without ever mutating repository files or bypassing human approval.

#### Acceptance Criteria

1. `R2.AC1` WHEN `tools/call` invokes `resolve_rules` (with optional `level` or `enforcement` filter) or `explain_rule` (`id`), the server SHALL resolve the pyramid offline and return structured JSON rule data, provenance, and charter metadata.
   - Acceptance check: `TestMCPServerProtocolAndTools` verifies `resolve_rules` filtering and `explain_rule` lookup.
2. `R2.AC2` WHEN `tools/call` invokes `check_text` (`content`, optional `path`), the server SHALL evaluate active built-in `regex`/`pattern` check rules matching `path` against `content` in memory without modifying disk files and return `verdict` (`pass`|`fail`) plus findings.
   - Acceptance check: `TestMCPServerProtocolAndTools` verifies `check_text` detects a seeded violation and passes clean text.
3. `R2.AC3` WHEN `tools/call` invokes `propose_rule` (`id`, `title`, `level`, `severity`, `enforcement`, `statement`) or `get_toolchain` (optional `include_check`), the server SHALL validate `propose_rule` drafts against `gradus/v1` schema and target level charter without writing any file, or return the resolved monotonic toolchain plus optional check findings for `get_toolchain`.
   - Acceptance check: `TestMCPServerProtocolAndTools` verifies `propose_rule` returns a validated Markdown draft (flagging charter severity violations) without creating files, and `get_toolchain` returns resolved required/forbidden items.

### R3 CLI Integration (`gradusctl serve --mcp`)

**User Story:** As a user configuring an MCP client (e.g. Claude Desktop, Cursor, Antigravity), I want to run `gradusctl serve --mcp --dir <repo>` as a standard stdio MCP server command.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl serve --mcp --dir <repo>` runs with MCP JSON-RPC requests on stdin, it SHALL process requests, write JSON-RPC responses to stdout, exit `0` on EOF, and exit `2` when `--mcp` is omitted.
   - Acceptance check: `TestServeMCPCommandCLI` verifies stdio round-trip via `gradusctl serve --mcp` and exit `2` when `--mcp` is omitted.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary using only Go standard library packages for MCP transport (`encoding/json`, `bufio`) with zero new external dependencies (`PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` All five MCP tools SHALL be strictly non-mutating on the repository filesystem (agents propose; humans approve).
2. `C2` All existing unit and integration tests SHALL pass without regressions.
