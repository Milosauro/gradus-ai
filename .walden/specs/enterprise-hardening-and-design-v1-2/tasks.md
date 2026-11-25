---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T16:18:16Z
last_modified: 2026-11-25T16:20:05Z
approved_fingerprint: sha256:b537d14bb9ba831dca118b33bec0b39737ebc396186c09740b6c0b7deb20d24a
source_design_approved_at: 2026-11-25T16:18:16Z
source_design_fingerprint: sha256:10e234dc8690006f800818a4f0cab1ff1e9d1ceaa653560aa2a9e99e0806bbf3
---

# Implementation Plan

- [x] 1. Make `guard` and `status --hook` safe in subdirectories and non-Gradus repos, and refine anti-tamper
  - [x] 1.1 Update `internal/cli/guard.go`, `internal/cli/status.go`, `internal/guard/antitamper.go`, and add `TestGuardAndStatusOutsideGradusV041` and `TestAntiTamperPrecisionV041`
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `R3.AC3`, `C1`, `C2`
    - Design: Architecture, Failure Modes And Tradeoffs
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestGuardAndStatusOutsideGradusV041$", "./internal/cli"]
        expect_output: "--- PASS: TestGuardAndStatusOutsideGradusV041"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAntiTamperPrecisionV041$", "./internal/guard"]
        expect_output: "--- PASS: TestAntiTamperPrecisionV041"
        covers: ["R3.AC3"]

- [x] 2. Harden MCP server, historical trailer verification, and `resolve --write` conflict guard
  - [x] 2.1 Update `internal/mcp/server.go`, `internal/trailers/trailers.go`, `internal/cli/resolve.go`, and add `TestMCPHardeningV041` and `TestTrailersHistoricalAndResolveGuardV041`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `R3.AC1`, `R3.AC2`, `NFR1`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestMCPHardeningV041$", "./internal/mcp"]
        expect_output: "--- PASS: TestMCPHardeningV041"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3"]
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestTrailersHistoricalAndResolveGuardV041$", "./internal/cli"]
        expect_output: "--- PASS: TestTrailersHistoricalAndResolveGuardV041"
        covers: ["R3.AC1", "R3.AC2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]

- [x] 3. Publish Design 1.2 and synchronize `SPEC.md`, `compatibility-matrix.md`, `README.md`, and `CHANGELOG.md`
  - [x] 3.1 Update `docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/compatibility-matrix.md`, `README.md`, `CHANGELOG.md`, and add `TestDesign12SyncV041`
    - Requirements: `R4.AC1`, `NFR2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestDesign12SyncV041$", "./internal/cli"]
        expect_output: "--- PASS: TestDesign12SyncV041"
        covers: ["R4.AC1", "NFR2"]
