---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T14:41:35Z
last_modified: 2026-11-23T14:44:02Z
approved_fingerprint: sha256:9109a1f920f54990989783e7413a3234678f980b819793514d5e7f4ec7d23546
source_design_approved_at: 2026-11-23T14:39:41Z
source_design_fingerprint: sha256:afb4d3aa33b06bb67296f98175f8cf211a8d315406a526af658557d2bad658a7
---

# Implementation Plan

- [x] 1. Embed official `walden` and `spec-kit` adapter manifests and implement registry loader
  - [x] 1.1 Create `adapters/embed.go`, `adapters/walden/adapter.yaml`, `adapters/spec-kit/adapter.yaml`, and `internal/adapter/registry.go` with unit tests
    - Requirements: `R1.AC1`, `R1.AC2`, `R1.AC3`, `NFR1`, `NFR2`
    - Design: Architecture
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAdapterRegistry$", "./internal/adapter"]
        expect_output: "--- PASS: TestAdapterRegistry"
        covers: ["R1.AC1", "R1.AC2", "R1.AC3", "NFR1", "NFR2"]

- [x] 2. Implement adapter sync, drift detection, SemVer CLI checks, and evidence verification
  - [x] 2.1 Implement `internal/adapter/sync.go` and `internal/adapter/verify.go` with unit tests
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `NFR1`, `NFR3`, `C1`
    - Design: Architecture, Simplicity And Elegance Review
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAdapterSyncAndVerify$", "./internal/adapter"]
        expect_output: "--- PASS: TestAdapterSyncAndVerify"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "NFR3", "C1"]

- [x] 3. Wire `gradusctl adapter` CLI (`list`, `sync`, `verify`) and end-to-end tests
  - [x] 3.1 Implement `internal/cli/adapter.go`, register in `internal/cli/root.go`, and add CLI tests in `internal/cli/adapter_test.go`
    - Requirements: `R2.AC1`, `R2.AC2`, `R2.AC3`, `R2.AC4`, `R3.AC1`, `C2`
    - Design: Architecture, Verification Plan
    - Verification:
      - command: ["go", "test", "-v", "-count=1", "-run", "^TestAdapterCommand$", "./internal/cli"]
        expect_output: "--- PASS: TestAdapterCommand"
        covers: ["R2.AC1", "R2.AC2", "R2.AC3", "R2.AC4", "R3.AC1", "C2"]
      - command: ["sh", "-c", "CGO_ENABLED=0 go build -o /dev/null ./cmd/gradusctl"]
        expect_exit: 0
        covers: ["NFR1"]
