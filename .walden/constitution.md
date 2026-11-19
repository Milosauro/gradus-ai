# Project Constitution

This file captures stable project-wide context that applies across all features. It is the repository-level invariant definition for Gradus.

## Project Summary

**Gradus** provides layered, governed rules for AI coding agents — from enterprise and organizational policies down to individual developer habits. It introduces a configurable pyramid of steps, a strict restriction invariant, deterministic compile-time resolution, lockfiles, runtime action guards, and multi-agent instruction compilation.

- **Repository:** `github.com/Milosauro/gradus-ai`
- **Owner (Phase 0):** Alessandro Enrico Milani (Milosauro)
- **Design Baseline:** `docs/design/gradus-design.md` (v1.0.2, schema `gradus/v1` draft)
- **Specification Workflow:** Walden (`walden v0.14.0+`)

## Tech Stack

- **Primary Language:** Go (`CGO_ENABLED=0`, `-trimpath`, static binary)
- **CLI Framework:** `github.com/spf13/cobra`
- **YAML Parser:** `gopkg.in/yaml.v3`
- **Glob Matching:** `github.com/bmatcuk/doublestar/v4` (required for `**/*` recursive pattern support)
- **VCS Integration:** Subprocess execution of system `git` CLI via `os/exec` (reusing existing user credentials and configs; no `go-git` or `libgit2`)
- **Build & Packaging:** Single static executable binary (`gradusctl`), GoReleaser, Syft (SBOM), Cosign (keyless signing)
- **Specification & Validation:** JSON Schema (`spec/v1/schema/`), Walden CLI (`walden validate --all`)

## Conventions

- **Language:** **Canonical English** for all repository artifacts without exception (source code, comments, identifiers, CLI output, error messages, documentation, tests, specs, commit messages, PRs).
- **File & Monorepo Layout:**
  - `go.mod` at repository root (`module github.com/Milosauro/gradus-ai`).
  - `cmd/gradusctl/main.go` for CLI entrypoint.
  - `internal/` for private Go packages (`cli`, `version`, `schema`, `source`, `lock`, `resolve`, `compile`, `check`, `guard`, `evidence`, `hooks`, `state`).
  - `skills/` for canonical Agent Skills definitions (`skills/gradus-*/SKILL.md`) and embedded packaging (`skills/embed.go`).
  - `spec/v1/` for open specification and JSON Schema definitions.
  - `docs/` for product design (`docs/design/gradus-design.md`), ADRs (`docs/adr/`), implementation plans (`docs/plan/`), and threat modeling (`docs/threat-model.md`).
- **Branching Strategy:**
  - Trunk-based development: `main` is always green and protected.
  - Short-lived working branches: `feat/<feature>`, `fix/<slug>`, `docs/<slug>`, `spec/<slug>`, `ci/<slug>`, `chore/<slug>`.
  - Release branches on demand: `release/vX.Y` (e.g. `release/v0.1`), bugfixes applied upstream-first on `main` and cherry-picked.
- **Commit Messages:** Conventional Commits in English (`feat:`, `fix:`, `docs:`, `spec:`, `test:`, `ci:`, `refactor:`, `chore:`).
- **Versioning:** Semantic Versioning 2.0.0 (`v0.1.0-alpha.x` for development, `v0.1.0` for Phase 0 completion).

## Sanity Checks

```bash
# Build binary
CGO_ENABLED=0 go build -trimpath ./cmd/gradusctl

# Run unit and race tests
go test -v -race -count=1 ./...

# Vet and linting
go vet ./...
golangci-lint run ./...

# Walden portfolio validation
walden validate --all --json
```

## Key Files

- `README.md`: Public project overview, vision, and quickstart.
- `docs/design/gradus-design.md`: Canonical design specification v1.0.2.
- `docs/plan/phase-0-plan.md`: Master Phase 0 implementation guide.
- `docs/adr/`: Architectural Decision Records (0001–0004).
- `cmd/gradusctl/main.go`: Entrypoint for the CLI.
- `spec/v1/SPEC.md`: Specification of the `gradus/v1` schema and formats.
- `.walden/constitution.md`: This document.
- `CONTRIBUTING.md`: Contributor workflow guidelines.
- `SECURITY.md`: Security policies and responsible disclosure.

## Hard Rules

1. **English Only:** Every repository file MUST be written in English.
2. **Separation of Judgment and Determinism:** Skills and models supply judgment; `gradusctl` deterministically enforces validation and executes all state mutations.
3. **No Autonomous Commits:** The agent must never perform git commit or push without explicit user request.
4. **Exit Code Semantics:** CLI commands strictly follow `0` (pass), `1` (rule violation / drift / conflict), `2` (technical error / invalid schema / missing checker / lock tamper). An error is NEVER a pass.
5. **Restriction Invariant:** Lower pyramid levels may only restrict or specialize upper levels; they may never loosen higher-level rules unless explicitly declared `overridable: true`.
6. **No Self-Approval:** Agent and skills never approve specs, tasks, rule changes, exceptions, or releases. Explicit human approval is required.
7. **Forward Compatibility:** Unknown YAML/JSON keys emit non-fatal diagnostic notes, never fatal errors.
8. **Static Hook Dispatchers:** Agent hook files (`.agents/hooks.json`, `.claude/settings.json`) register only a static dispatcher to `gradusctl guard`; local/user-level rules are resolved at runtime and must never dirty git-tracked hook definitions.
9. **Zero Personal Telemetry:** No tracking, metrics, or ranking of human activity or individual performance (Principle P16).
10. **Pinned CI Actions:** All GitHub Actions workflows MUST use 40-character commit SHAs.
