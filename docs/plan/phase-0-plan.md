# Gradus — Phase 0 Master Implementation Plan

| Field | Value |
|---|---|
| Project | **Gradus** (`github.com/Milosauro/gradus-ai`) |
| Design baseline | `docs/design/gradus-design.md` (v1.0.2, schema `gradus/v1` draft) |
| SDD Workflow | Walden (`walden v0.14.0+`) |
| Target Milestone | **Phase 0 MVP (`v0.1.0`)** — Exit criteria `E1`–`E10` |
| Status | Active (Wave 0 complete; ready for Wave 1) |

---

## 1. Golden Rules for Developers

1. **Language:** Every file in this repository (Go code, comments, identifiers, CLI output, error messages, documentation, `.walden/*`, ADRs, tests, commit messages, PRs) MUST be written in **English**.
2. **Approval Gates:** Never commit, push, or run `walden review approve` without explicit human confirmation in chat. Never start implementing a Walden feature until `requirements`, `design`, and `tasks` are all `approved` and execution is explicitly requested.
3. **State Ownership:** Walden state (frontmatter, timestamps, fingerprints, checkboxes, evidence ledgers) is modified exclusively by the `walden` CLI. Gradus state (`gradus.lock`, `.gradus/evidence/`, `.gradus/exceptions/`, `.gradus/findings/`, `~/.gradus/lessons/`) is modified exclusively by `gradusctl`.
4. **Exit Codes (D-73):** Commands strictly adhere to `0` (pass), `1` (rule violation / drift / conflict), `2` (technical error / invalid schema / missing checker / lock tamper / unreachable `fail-closed` source). **An error is NEVER reported as a pass.**
5. **Forward Compatibility (§9.2):** Unknown YAML/JSON keys emit non-fatal diagnostic notes, never fatal errors.
6. **Go Build & Git (D-69):** Builds use `CGO_ENABLED=0` (static binary). Git operations shell out to the system `git` CLI via `os/exec` (no `go-git` or `libgit2`).
7. **Approved Dependencies:** `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/bmatcuk/doublestar/v4` (required because stdlib `filepath.Match` does not support `**/*`), and `github.com/santhosh-tekuri/jsonschema/v6` (optional for JSON Schema tests).
8. **Privacy (P16):** Zero personal activity metrics anywhere in code, schemas, lessons, or evidence.
9. **Performance:** `gradusctl guard` and `gradusctl status --brief` must complete in `< 100ms` using local cached state without network calls.
10. **Branching & Commits:** Trunk-based (`main` protected), short-lived branches (`feat/<feature>`, `fix/<slug>`, `docs/<slug>`, `ci/<slug>`, `chore/<slug>`), on-demand `release/vX.Y` branches (upstream-first cherry-pick), Conventional Commits in English.

---

## 2. Walden Spec-Driven Workflow (Per-Feature Loop)

For every feature `F1`–`F8`:

1. **Requirements:** Run `walden feature init <feature> --json`. Write EARS acceptance criteria (`R1.AC1`, etc.) with an `Acceptance check:` line under each AC. Run `walden validate <feature> --json` and `walden review open <feature> --phase requirements --json`. Wait for explicit approval, then run `walden review approve <feature> --phase requirements --json`.
2. **Design:** Author `design.md` with the 6 required H2 headings (`Architecture`, `Options Considered`, `Simplicity And Elegance Review`, `Failure Modes And Tradeoffs`, `Verification Plan`, `Requirement Coverage` with backticked IDs). Validate, open review, wait for explicit approval, approve via CLI.
3. **Tasks:** Author `tasks.md` using 2-space metadata, 4-space `- command:`, 6-space `expect_output:` / `covers:`. For Go tests, always use `-v -count=1 -run ^TestExactName$` with `expect_output: "--- PASS: TestExactName"` so zero-match runs cannot pass vacuously. Validate (`--all`), open review, wait for explicit approval, approve via CLI.
4. **Execution (TDD):** Run `walden task start <feature> <id> --json`. Write failing behavioral test first, implement minimal code, complete via `walden task complete <feature> <id> --json`.
5. **Consolidation:** Run `walden consolidate` every 2–3 approved features to keep `.walden/contracts.md` coherent.

---

## 3. Monorepo Architecture

```text
gradus-ai/
├── go.mod                          # module github.com/Milosauro/gradus-ai
├── go.sum
├── cmd/gradusctl/main.go           # minimal CLI entrypoint calling internal/cli
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
│   └── state/                      # findings, lessons (redacted), drift reconciliation
├── skills/
│   ├── embed.go                    # //go:embed gradus-*/*.md
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

## 4. Technical Contracts & Algorithms

### 4.1 Normalized Cross-Platform Hashing (`internal/lock/hash.go`)
1. Walk all regular files in the level/overlay directory (`level.yaml`, `rules/*.md`), skipping `.git/` and temporary hidden files.
2. Convert relative paths to forward slashes (`filepath.ToSlash`) and sort ASCII-lexicographically.
3. For each file, normalize `\r\n` to `\n`, then feed `<relPath>\n<byteLen>\n<normalizedBytes>\n` into `sha256`.
4. Format result as `"sha256:" + hex`.

### 4.2 Managed Sections (`internal/compile/managed.go`)
Shared files (`AGENTS.md`, `CLAUDE.md`) use exact delimiters:
```markdown
<!-- gradus:begin view_hash=sha256:... -->
... generated content ...
<!-- gradus:end -->
```
- Missing file → create with managed block.
- Existing markers → replace only between markers (inclusive), preserving user prose outside.
- Existing file without markers → append managed block at the bottom.
- Drift detection compares on-disk bytes against in-memory expected output.

### 4.3 Antigravity Rule Splitting (`internal/compile/antigravity.go`)
- Separate project/team rules (`.agents/rules/gradus-*.md`) from user rules (`.agents/rules/gradus-user-*.md`, git-ignored).
- Group by `scope`: global `["**/*"]` → `trigger: always_on`; scoped → `trigger: glob` with `globs: [...]`.
- Sort by pyramid level order, then rule ID.
- Chunk into `001.md`, `002.md`, ... whenever adding the next rule would exceed **11,500 characters** (500-char safety margin under Antigravity's 12,000-char limit).

### 4.4 Static Guard Dispatcher (`internal/guard/guard.go`, ADR 0003)
- `.agents/hooks.json` and `.claude/settings.json` register only a static `PreToolUse` command calling `gradusctl guard`.
- `gradusctl guard` reads the tool payload from `stdin`, evaluates the local resolved view (including `~/.gradus/levels/user`) in `< 100ms`, enforces built-in anti-tamper confirmation (`ask`) on governance files (`gradus.yaml`, `gradus.lock`, `.gradus/**`, `.walden/**`, hook configs), and resolves `deny > ask > allow`.

---

## 5. Implementation Waves

- **Wave 0A & 0B (Done on `chore/wave-0-bootstrap`):** Walden bootstrap, constitution, ADRs 0001–0004, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, CI workflow (SHA-pinned Actions, OS matrix), Go module & `gradusctl version [--json]`.
- **Wave 1 — Deterministic CLI Kernel (Walden Features F1–F5):**
  - **F1 `core-schema-init`:** `spec/v1/SPEC.md` + JSON Schemas, `internal/schema` parser/validator, `gradusctl init`, `gradusctl explain`.
  - **F2 `resolve-and-lock`:** `internal/source` (local + git `os/exec` + offline policy), `internal/lock`, `internal/resolve` (§10, P2 restriction invariant, structural conflicts), `gradusctl resolve [--json] [--write]`, `gradusctl status [--brief]`. *(Consolidate after F2.)*
  - **F3 `agent-compilers`:** `gradusctl compile` for Claude Code, Antigravity (`<11.5k` split + trigger frontmatter), `AGENTS.md`, and embedded skill extraction. *(Covers E1.)*
  - **F4 `verify-check-evidence`:** `gradusctl verify`, `gradusctl check [--staged | --base]`, `gradusctl evidence` (5 sinks), `gradusctl hooks` (`pre-commit`/`lefthook`/`husky`/`native`). *(Covers E2, E8.)*
  - **F5 `runtime-guard`:** `gradusctl guard` (`<100ms`) + anti-tamper protection. *(Covers E3. Consolidate after F5 & activate Stage 1 Dogfooding on `gradus-ai`.)*
- **Wave 2 — Lifecycle State & MVP Skills (Walden Features F6–F7):**
  - **F6 `cli-lifecycle-state`:** `gradusctl conflicts`, `gradusctl lessons` (secret/PII redaction), `gradusctl reconcile`.
  - **F7 `mvp-skills`:** All 6 Agent Skills (`onboard`, `author`, `review`, `learn`, `reconcile`, `conflicts`) developed with skill-TDD in `evals/skills/` and embedded via `skills/embed.go`. *(Covers E4–E7. Consolidate after F7 & activate Stage 2 Dogfooding.)*
- **Wave 3 — Evaluation, Release Pipeline & `v0.1.0` (Walden Feature F8):**
  - **F8 `phase0-eval-release`:** `>= 10` scenario comparative benchmark in `evals/benchmark/` (E9), real-project E1–E10 validation, `.goreleaser.yaml` + `.github/workflows/release.yml` (`cosign`, `syft` SBOM, provenance), Decision Gate O-2 (owner), branch `release/v0.1`, `walden release check --strict`, tag `v0.1.0`.
