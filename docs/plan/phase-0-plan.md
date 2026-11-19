# Gradus — Phase 0 Master Implementation Plan

| Field | Value |
|---|---|
| Project | **Gradus** (`github.com/Milosauro/gradus-ai`) |
| Design baseline | `docs/design/gradus-design.md` (v1.0.2, schema `gradus/v1` draft) |
| SDD Workflow | Walden (`walden v0.14.0+`) |
| Target Milestone | **Phase 0 MVP (`v0.1.0`)** — Exit criteria `E1`–`E10` |
| Status | Active (Wave 0 implementation) |

---

## 1. Golden Rules for Developers

1. **Language:** Every file in this repository (Go code, comments, identifiers, CLI output, error messages, documentation, `.walden/*`, ADRs, tests, commit messages, PRs) MUST be written in **English**.
2. **Approval Gates:** The agent must never commit, push, or run `walden review approve` without explicit human confirmation in chat.
3. **State Ownership:** Walden state is modified exclusively by the `walden` CLI. Gradus state (`gradus.lock`, evidence, exceptions, findings, lessons) is modified exclusively by `gradusctl`.
4. **Exit Codes:** Commands strictly adhere to `0` (pass), `1` (rule violation / drift / conflict), `2` (technical error / invalid schema / missing checker / lock tamper). An error is NEVER a pass.
5. **Forward Compatibility:** Unknown YAML/JSON keys emit non-fatal diagnostic notes, never fatal errors.
6. **Go Build & Git:** Builds use `CGO_ENABLED=0` (static binary). Git operations use the system `git` CLI via `os/exec`.
7. **Approved Dependencies:** `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/bmatcuk/doublestar/v4`, `github.com/santhosh-tekuri/jsonschema/v6`.
8. **Privacy (P16):** Zero personal activity metrics anywhere.
9. **Performance:** `gradusctl guard` and `gradusctl status --brief` must complete in `< 100ms`.
10. **Branching & Commits:** Trunk-based (`main`), short-lived branches, Conventional Commits in English.

---

## 2. Monorepo Architecture

```text
gradus-ai/
├── go.mod                          # module github.com/Milosauro/gradus-ai
├── go.sum
├── cmd/
│   └── gradusctl/
│       └── main.go                 # minimal CLI entrypoint calling internal/cli
├── internal/
│   ├── cli/                        # cobra commands, JSON envelope, exit codes 0/1/2
│   ├── version/                    # build metadata (version, commit, date)
│   ├── schema/                     # types, YAML/frontmatter parsing, validation
│   ├── source/                     # local and git (os/exec) fetchers, cache, offline policy
│   ├── lock/                       # normalized SHA-256 hashing, gradus.lock management
│   ├── resolve/                    # §10 resolution algorithm, restriction invariant P2
│   ├── compile/                    # Claude Code, Antigravity (<12k split), AGENTS.md generators
│   ├── check/                      # built-in and external checker plugin runner
│   ├── guard/                      # PreToolUse evaluator (<100ms) + anti-tamper
│   ├── evidence/                   # sinks: none, local, ci-artifact, repo, command
│   ├── hooks/                      # pre-commit, lefthook, husky, native hooks integration
│   └── state/                      # findings, lessons, drift reconciliation
├── skills/
│   ├── embed.go                    # //go:embed gradus-*/SKILL.md
│   └── gradus-*/SKILL.md           # canonical Agent Skills definitions
├── spec/v1/
│   ├── SPEC.md
│   └── schema/*.schema.json        # JSON Schemas for configuration and evidence
├── examples/
├── evals/
└── docs/
    ├── design/gradus-design.md
    ├── adr/
    ├── plan/phase-0-plan.md
    └── threat-model.md
```

---

## 3. Implementation Waves

- **Wave 0A:** Repository governance, `.walden/constitution.md`, `.walden/environment.md`, ADRs 0001–0004, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `.gitignore`, `.pre-commit-hooks.yaml`, `.golangci.yml`, `.github/workflows/ci.yml`.
- **Wave 0B:** Go module initialization (`go mod init github.com/Milosauro/gradus-ai`), version package, minimal `gradusctl version --json` command, `skills/embed.go` placeholder.
- **Wave 1:** Deterministic CLI Kernel (Walden features F1–F5):
  - F1: `core-schema-init` (`spec/v1`, parser, `init`, `explain`)
  - F2: `resolve-and-lock` (git fetcher, normalized lock, §10 resolution, `resolve`, `status`)
  - F3: `agent-compilers` (`compile` for Claude Code, Antigravity, AGENTS.md — covers E1)
  - F4: `verify-check-evidence` (`verify`, `check`, `evidence`, `hooks` — covers E2, E8)
  - F5: `runtime-guard` (`guard` PreToolUse + anti-tamper — covers E3)
  - *Milestone:* Stage 1 Dogfooding on `gradus-ai`.
- **Wave 2:** Lifecycle State & MVP Skills (Walden features F6–F7):
  - F6: `cli-lifecycle-state` (`conflicts`, `lessons`, `reconcile`)
  - F7: `mvp-skills` (the 6 Agent Skills developed with skill-TDD in `evals/skills/` — covers E4–E7)
  - *Milestone:* Stage 2 Dogfooding on `gradus-ai`.
- **Wave 3:** Comparative Evaluation & Release (Walden feature F8):
  - F8: `phase0-eval-release` (eval harness E9, end-to-end verification E10, `.goreleaser.yaml`, cosign, SBOM, tag `v0.1.0`).
