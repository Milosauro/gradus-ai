---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:24:35Z
last_modified: 2026-11-25T09:24:35Z
approved_fingerprint: sha256:2c818b8f989d542b72bf78f4e9f20774178b7ca19835df3b0a3f50f64786217c
source_requirements_approved_at: 2026-11-25T09:24:35Z
source_requirements_fingerprint: sha256:582093e3a8b45597ff743fea3855b97c54f69f23c20fdcd7d5327454b5fa553a
---

# Design: MCP Server (`gradusctl serve --mcp`, `F27`)

## Architecture

1. **MCP Server Package (`internal/mcp/server.go`):**
   - Pure Go stdlib JSON-RPC 2.0 stdio transport (`bufio.Scanner` with 4 MiB max line buffer + `json.Encoder` writing newline-delimited JSON-RPC responses).
   - Supported MCP lifecycle methods (`protocolVersion: "2024-11-05"`):
     - `initialize` → returns `protocolVersion`, `capabilities: { tools: {} }`, `serverInfo: { name: "gradusctl", version: version.Version }`.
     - `notifications/initialized` → no response (JSON-RPC notification).
     - `ping` → returns `{}`.
     - `tools/list` → returns the 5 tool descriptors with JSON Schema `inputSchema`.
     - `tools/call` → dispatches to the 5 handlers, returning MCP `content: [{ type: "text", text: "<json>" }]` and `isError` on domain/input errors.
   - **Tool Handlers (all strictly read-only on disk):**
     1. `resolve_rules`: calls `resolve.Resolve(rootDir, resolve.Options{OfflineOnly: offline})`, optionally filters rules by `level` and/or `enforcement`, returns `view_hash`, `levels`, `rules`, `conflicts`.
     2. `explain_rule`: resolves pyramid, finds rule by `id`, returns rule metadata, provenance, level charter, and active waiver status (or `isError: true` if not found).
     3. `check_text`: resolves pyramid, matches active non-waived built-in `regex`/`pattern` check rules whose `scope` matches `path` (default `"snippet.txt"` via `doublestar.Match`), scans `content` line-by-line in memory, returns `{ verdict, path, findings }`.
     4. `propose_rule`: renders a `gradus/v1` rule Markdown document from input arguments, validates it with `schema.ParseRuleBytes`, checks the target level's charter `allowed_severities` (if declared) and whether the target level is `locked`, and returns `{ valid, locked_target, charter_warnings, proposed_path, markdown }` without touching disk.
     5. `get_toolchain`: resolves pyramid, returns `view.Toolchain`, and when `include_check: true` runs `toolchain.Check(rootDir, view.Toolchain)` and includes findings.
2. **CLI Command (`internal/cli/serve.go`, `internal/cli/root.go`):**
   - `gradusctl serve --mcp [--dir .] [--offline=true]` reads from `cmd.InOrStdin()` and writes to `cmd.OutOrStdout()`.
   - Calling `gradusctl serve` without `--mcp` returns `ExitError` (`2`) explaining that `--mcp` is required.

## Options Considered

- **Option A (Chosen):** Minimal, zero-dependency Go stdlib JSON-RPC 2.0 MCP stdio server (`~300` lines). Keeps `go.mod` clean, preserves `CGO_ENABLED=0` static builds, and is trivially testable with `bytes.Buffer` pipes.
- **Option B (Rejected):** Pull in an external MCP Go SDK. Adds transitive dependencies and breaking-change risk for a 5-tool stdio server.

## Simplicity And Elegance Review

- Reuses `resolve.Resolve`, `schema.ParseRuleBytes`, `toolchain.Check`, and `doublestar.Match` directly; `check_text` runs purely in memory without temp files.

## Failure Modes And Tradeoffs

- **Malformed JSON line on stdin:** Returns JSON-RPC `-32700` Parse error and continues reading subsequent lines until EOF.
- **External checkers in `check_text`:** External CLI checkers (e.g. `gitleaks`) require filesystem paths and are skipped in in-memory `check_text` with a note, while built-in `regex`/`pattern` rules execute deterministically.

## Verification Plan

- Unit tests in `internal/mcp/server_test.go` (`TestMCPServerProtocolAndTools`).
- CLI stdio integration tests in `internal/cli/serve_test.go` (`TestServeMCPCommandCLI`).
- `go mod tidy -diff` and `CGO_ENABLED=0 go build`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/mcp/server.go` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/mcp/server.go` (`R2.AC1`–`R2.AC3`) |
| `R3` | `internal/cli/serve.go`, `internal/cli/root.go` (`R3.AC1`) |
| `NFR1` | Stdlib JSON-RPC 2.0, `go mod tidy -diff`, static `CGO_ENABLED=0` build |
| `NFR2` | Canonical English across all repository files |
| `C1` | Strictly non-mutating handlers verified in `TestMCPServerProtocolAndTools` |
| `C2` | Full Go test suite regression pass |
