---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:24:35Z
last_modified: 2026-11-25T09:27:08Z
approved_fingerprint: sha256:8dda1845f964b8b643b925ddc92836afc0dc5ec27e616bf5117865cbee8ac68e
source_design_approved_at: 2026-11-25T09:24:35Z
source_design_fingerprint: sha256:2c818b8f989d542b72bf78f4e9f20774178b7ca19835df3b0a3f50f64786217c
---

# Implementation Plan

- [x] 1. Implement zero-dependency MCP JSON-RPC 2.0 stdio server and 5 governed tools
  - [x] 1.1 Create `internal/mcp/server.go` and `internal/mcp/server_test.go`
    - Requirements: `R1.AC1`, `R1.AC2`, `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `C1`
    - Design: Architecture, Simplicity And Elegance Review, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestMCPServerProtocolAndTools$", "./internal/mcp"]
        expect_output: "--- PASS: TestMCPServerProtocolAndTools"
        covers: ["R1.AC1", "R1.AC2", "R2.AC1", "R2.AC2", "R2.AC3", "C1"]

- [x] 2. Wire `gradusctl serve --mcp` CLI command
  - [x] 2.1 Create `internal/cli/serve.go`, register in `internal/cli/root.go`, and add `TestServeMCPCommandCLI` in `internal/cli/serve_test.go`
    - Requirements: `R3.AC1`, `NFR1`, `NFR2`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestServeMCPCommandCLI$", "./internal/cli"]
        expect_output: "--- PASS: TestServeMCPCommandCLI"
        covers: ["R3.AC1"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
