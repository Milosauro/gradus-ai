# Gradus — Phase 2 ("Business Unit") Implementation Plan

| Field | Value |
|---|---|
| Scope | Phase 2 (`v0.3.0` – `v0.3.2`) |
| Status | Completed and released |
| Design Reference | [`docs/design/gradus-design.md`](../design/gradus-design.md) (Design 1.1) |

## Goal

Extend Gradus from single-team governance (Phase 1) to multi-team business-unit governance with cryptographically verifiable remote levels, abstract process obligations bound to external SDD providers, monotonic toolchain governance, third-party tool auditing, and expanded coding-agent compilation targets.

## Feature Breakdown (Walden `F15`–`F22`)

| Feature | Walden Spec | Scope |
|---|---|---|
| **F15** | `remote-levels-signatures-and-changelog` | Git tag/commit signature verification (`optional` / `required`) via `os/exec`, SemVer 2.0.0 `min_version` constraints, and non-destructive `gradusctl changelog <level-id>`. |
| **F16** | `obligations-and-bindings` | Abstract `obligations` (`allowed_providers`, `scope`), concrete `bindings` (`provider`, `requires`, `scope`), narrowing validation in `resolve`/`verify`, and handbook rendering. |
| **F17** | `adapters-walden-and-speckit` | Official embedded adapters (`walden`, `spec-kit`), custom declarative `.gradus/adapters/` manifests with path containment, and `gradusctl adapter list | sync | verify` (managed-block drift detection). |
| **F18** | `toolchain-governance` | Monotonic `required`, `recommended`, and `forbidden` skills/tools/MCPs/plugins in `gradus.yaml`/`level.yaml`, conflict detection in `resolve`, and offline `gradusctl toolchain check`. |
| **F19** | `gradus-link-skill` | Embedded `gradus-link` agent skill + Skill-TDD scenario `SKILL-07` for third-party tool compatibility/security auditing and binding proposals. |
| **F20** | `codex-copilot-opencode-targets` | Compilers for `codex`, `copilot` (`.github/copilot-instructions.md` + scoped `.github/instructions/gradus-*.instructions.md`), and `opencode`, plus `docs/compatibility-matrix.md`. |
| **F21** | `security-hardening-and-lock-upgrades` | Human-only governance guard, exception `approvers` validation, project-only `view_hash`, controlled `gradusctl resolve --update`, extended anti-tamper, handbook numbering, and GoReleaser `go mod tidy -diff` (`v0.3.1`). |
| **F22** | `action-hardening-and-design-v1-1` | Non-Go GitHub Action fix (`github.action_path`) + CI smoke job, residual git-hook anti-tamper bypasses, `internal/check` edge-case coverage, early `max_duration` validation in `exception request`, redaction fixes, `go.yaml.in/yaml/v3` migration, `PRJ-CI-001` check promotion, and Design 1.1 (`v0.3.2`). |

## Exit Verification

- Full Go unit/integration test suite (`go test ./...`), `go mod tidy -diff`, and static `CGO_ENABLED=0` binary build.
- Self-governance checks on `gradus-ai` (`gradusctl verify`, `gradusctl check`, `gradusctl test`).
- Walden specification validation across all features (`walden validate --all --json`).
- Live CLI and session trials on Claude Code and Antigravity against a real consumer repository (`wallet-tracker`), with sandboxed CLI validation for `codex`, `copilot`, and `opencode` compilation targets.
