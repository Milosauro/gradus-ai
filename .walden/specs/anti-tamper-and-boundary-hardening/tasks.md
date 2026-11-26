---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-26T12:20:01Z
last_modified: 2026-11-26T12:22:19Z
approved_fingerprint: sha256:31fd3fd4b164aea86d07abc8c439c73990b573686020d9998aa3268dee8d987e
source_design_approved_at: 2026-11-26T12:18:22Z
source_design_fingerprint: sha256:2e4c24723af385a0f5f8456a96169bbc5d4a22607377387bdb435732ec2171ed
---

# Implementation Plan

- [x] 1. Harden anti-tamper detection, nested Git boundaries, and hook status resilience
  - [x] 1.1 Update `hookBypassEnvRe` in `internal/guard/antitamper.go`, enforce `.git` boundaries in `resolveWorkspaceRoot` (`internal/guard/payload.go`), make `status --hook` / `--antigravity-hook` exit `0` with `lock=invalid` on malformed `gradus.yaml` (`internal/cli/status.go`), and add unit tests `TestAntiTamperExportAndStandaloneV042` and `TestNestedGitBoundaryAndInvalidConfigV042`
    - Requirements: `R1.AC1`, `R1.AC2`, `R2.AC1`, `R2.AC2`, `R3.AC1`, `R3.AC2`, `NFR1`, `NFR2`
    - Design: Architecture, Failure Modes And Tradeoffs, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-race", "-count=1", "-run", "^TestAntiTamperExportAndStandaloneV042$", "./internal/guard"]
        expect_output: "--- PASS: TestAntiTamperExportAndStandaloneV042"
        covers: ["R1.AC1", "R1.AC2", "NFR1"]
      - command: ["go", "test", "-v", "-race", "-count=1", "-run", "^TestNestedGitBoundaryAndInvalidConfigV042$", "./internal/cli"]
        expect_output: "--- PASS: TestNestedGitBoundaryAndInvalidConfigV042"
        covers: ["R2.AC1", "R2.AC2", "R3.AC1", "R3.AC2", "NFR1"]

- [x] 2. Refine MCP diagnostics, close Design `O-1`, and update `v0.4.2` documentation
  - [x] 2.1 Split `propose_rule` traversal vs. path-separator error messages in `internal/mcp/server.go`, document fail-closed manual repair for malformed `gradus.yaml` in `README.md`, mark §24.3 validated on `wallet-tracker` and close `O-1` in `docs/design/gradus-design.md`, add `v0.4.2` to `CHANGELOG.md`, and add `TestMCPProposeRuleLevelDiagnosticsV042` and `TestDocsAndDesignClosureV042`
    - Requirements: `R3.AC3`, `R4.AC1`, `R4.AC2`, `NFR1`, `NFR2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-race", "-count=1", "-run", "^TestMCPProposeRuleLevelDiagnosticsV042$", "./internal/mcp"]
        expect_output: "--- PASS: TestMCPProposeRuleLevelDiagnosticsV042"
        covers: ["R4.AC1", "NFR1"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestDocsAndDesignClosureV042$", "./internal/cli"]
        expect_output: "--- PASS: TestDocsAndDesignClosureV042"
        covers: ["R3.AC3", "R4.AC2", "NFR2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl && go mod tidy -diff"]
        expect_exit: 0
        covers: ["NFR1"]
