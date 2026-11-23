---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T14:34:17Z
last_modified: 2026-11-23T14:34:17Z
approved_fingerprint: sha256:902ebe639ebcf8d790065ed93c0db13420f1e119553b31e2f3d96f493a58c24a
---

# Requirements Document

## Introduction

Phase 2 ("Business unit") connects abstract `obligations` and `bindings` (`F16`) to real specification and workflow providers through declarative **Adapters** (`docs/design/gradus-design.md` §19, §24.1). Adapters inject the resolved Gradus rules into provider files inside managed sections (such as `.walden/constitution.md` for Walden and `.specify/memory/constitution.md` for spec-kit), detect direct edits/drift, check `binding.requires` CLI version constraints, and verify provider evidence. This feature establishes the declarative adapter manifest format, the embedded official `walden` and `spec-kit` adapters, custom project adapter loading, and the `gradusctl adapter` CLI (`list`, `sync`, `verify`).

## Requirements

### R1 Declarative Adapter Manifests and Official Adapters

**User Story:** As a platform engineer, I want Gradus to ship official declarative adapters for `walden` and `spec-kit` while supporting custom declarative manifests in `.gradus/adapters/`, so that teams can integrate both standard and internal tools without writing Go code.

#### Acceptance Criteria

1. `R1.AC1` The system SHALL embed official declarative adapter manifests under `adapters/` (`adapters/walden/adapter.yaml` targeting `.walden/constitution.md` and `adapters/spec-kit/adapter.yaml` targeting `.specify/memory/constitution.md`) using `go:embed` (`adapters/embed.go`).
   - Acceptance check: Listing adapters in a fresh repository returns both `walden` and `spec-kit` with source `official` and their respective target files.
2. `R1.AC2` WHEN custom adapter manifests exist under `.gradus/adapters/*.yaml` or `.gradus/adapters/<id>/adapter.yaml`, the system SHALL load and validate them (`schema: gradus/v1`, non-empty `id` and `target_file`, safe relative `target_file` without `..` traversal) alongside official adapters.
   - Acceptance check: Placing a valid custom adapter YAML under `.gradus/adapters/internal-sdd.yaml` makes `internal-sdd` discoverable in `gradusctl adapter list` and usable by `sync`/`verify`.
3. `R1.AC3` IF an adapter manifest is malformed, omits `id` or `target_file`, or declares an absolute or path-traversal `target_file`, THEN loading SHALL fail with exit code `2`.
   - Acceptance check: A custom manifest with `target_file: ../outside.md` fails with exit code `2` and refuses to write outside the repository.

### R2 Managed-Section Sync and Drift Verification

**User Story:** As a developer working with Walden or spec-kit, I want `gradusctl adapter sync` and `gradusctl adapter verify` to keep provider constitution files synchronized with the resolved Gradus pyramid without erasing human-written content, and to detect unauthorized edits, CLI version mismatches, or failing provider evidence.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl adapter sync [--provider <id>] [--json]` runs on a repository with bound providers (or an explicit `--provider`), the CLI SHALL inject the resolved non-user rules into each adapter's `target_file` strictly between `<!-- gradus:begin view_hash=... -->` and `<!-- gradus:end -->` markers, preserving surrounding human-written prose byte-for-byte and producing zero diff on a second consecutive run.
   - Acceptance check: Running `gradusctl adapter sync` twice on a repo with existing `.walden/constitution.md` prose preserves the human text and reports `changed: false` on the second run.
2. `R2.AC2` WHEN `gradusctl adapter verify [--provider <id>] [--skip-evidence] [--json]` runs, the CLI SHALL check for managed-section drift in each bound adapter's `target_file`, validate `binding.requires["cli"]` (e.g. `>=0.10.0`) against the adapter's `version_command` using SemVer comparison, and (unless `--skip-evidence` is set) execute the adapter's `evidence_command` when configured.
   - Acceptance check: After `adapter sync`, `adapter verify` exits `0` when the file is clean, version constraint is satisfied, and evidence check succeeds.
3. `R2.AC3` IF a bound adapter's `target_file` has drift or is missing, its CLI version is below `binding.requires["cli"]`, or its `evidence_command` exits non-zero, THEN `gradusctl adapter verify` SHALL report the specific failure(s) and exit with code `1`.
   - Acceptance check: Editing the managed section of `.walden/constitution.md` or setting an unsatisfied `requires.cli` constraint causes `gradusctl adapter verify` to exit `1` with structured findings.
4. `R2.AC4` IF a bound provider or `--provider` argument has no registered official or custom adapter, THEN `gradusctl adapter sync` and `gradusctl adapter verify` SHALL exit with code `2` with a clear diagnostic.
   - Acceptance check: Requesting `--provider unknown-tool` exits with code `2`.

### R3 Adapter Discovery CLI (`gradusctl adapter list`)

**User Story:** As a team lead, I want to inspect which adapters are available and which ones are actively bound in my repository, so that I can audit provider integrations at a glance.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl adapter list [--json]` runs, the CLI SHALL list all registered official and custom adapters sorted deterministically by `id`, showing `id`, `source` (`official` | `custom`), `target_file`, and `bound` (`true` | `false` based on `ResolvedView.Bindings`).
   - Acceptance check: `gradusctl adapter list --json` returns sorted adapter entries marking bound providers with `bound: true`.

## Non-Functional Requirements

- `NFR1` **Static Build & Zero External Dependencies:** Official adapters MUST be embedded via `go:embed` (`CGO_ENABLED=0`) and provider commands executed via `os/exec` with argv arrays (no shell interpolation).
- `NFR2` **Path Safety:** Adapter `target_file` paths MUST resolve strictly inside the repository root and reject absolute paths or `..` traversal.
- `NFR3` **Idempotency:** Consecutive `gradusctl adapter sync` invocations MUST produce zero byte differences (`E1`).

## Constraints And Dependencies

- `C1` Must reuse `internal/compile` managed-section helpers (`RenderPrecedenceHeader`, `RenderRuleMarkdown`, `RenderManagedBlock`, `ApplyManagedSection`, `WriteIfChanged`), `internal/resolve` (`ResolvedView.Bindings`), and `internal/schema.ParseSemVer`.
- `C2` Must preserve `gradus/v1` backward compatibility for repositories without bindings or adapters.

## Out Of Scope

- Interactive third-party skill/tool security audit and manifest generation (`gradus-link` skill) — covered by `F19`.
- Toolchain `required` / `recommended` / `forbidden` enforcement — covered by `F18`.
