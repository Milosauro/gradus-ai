---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-26T12:18:22Z
last_modified: 2026-11-26T12:18:22Z
approved_fingerprint: sha256:2e4c24723af385a0f5f8456a96169bbc5d4a22607377387bdb435732ec2171ed
source_requirements_approved_at: 2026-11-26T12:15:36Z
source_requirements_fingerprint: sha256:78e0c852d44c8eae3d6f01e02d3ac339b84c5bbd31886ac3b89ab11cc0b5d1fe
---

# Feature Design

## Architecture

This release applies four targeted, low-coupling fixes across existing packages without introducing new packages or external dependencies:

1. **Shell Hook-Bypass Regex (`internal/guard/antitamper.go`):**
   - Extend `hookBypassEnvRe` with multiline mode `(?m)`, optional `export\s+` alongside `env\s+` in command/statement prefix position, and trailing boundary `(?:\s|[;&|]|$)` so standalone assignments separated by `;`, `&&`, `||`, `|`, or newlines (e.g., `export HUSKY=0; git commit`, `export SKIP=gradus`, `set -a; HUSKY=0; git commit`) are caught while non-assignment arguments (`echo SKIP=foo`, `grep LEFTHOOK=0`, `git commit -m "document HUSKY=0"`) remain unflagged.

2. **Git-Bounded Workspace Resolution (`internal/guard/payload.go`):**
   - In `resolveWorkspaceRoot(fallback, cwd string, workspacePaths []string) string`, before inspecting `parent := filepath.Dir(c)` for `gradus.yaml`, check `os.Stat(filepath.Join(c, ".git"))`. When `.git` exists (directory for standalone repos or file for git submodules/worktrees), stop climbing and do not inspect `parent`. Regular subdirectories without `.git` (such as `.agents/`) continue resolving their parent `gradus.yaml`, and deeper subdirectories rely on `findGradusRoot` in `internal/cli/guard.go`, which already stops at `.git`.

3. **Non-Blocking Hook Status on Malformed Config (`internal/cli/status.go`, `README.md`):**
   - In `newStatusCmd`, when `buildStatusReport(dir, userDir)` returns an error under `hookMode` or `antigravityHook`, emit the compact diagnostic status (`gradus: lock=invalid conflicts=0 drift=0 lessons=0`, wrapped in Antigravity `injectSteps` JSON when `antigravityHook` is set) and return `nil` (`exit 0`, satisfying D-82).
   - Non-hook `gradusctl status` and `gradusctl guard` remain fail-closed (`exit 2`). Document in `README.md` under Security boundaries that a malformed `gradus.yaml` fails closed in `gradusctl guard` and must be repaired manually.

4. **MCP Level Diagnostics & Design Doc Closure (`internal/mcp/server.go`, `docs/design/gradus-design.md`, `CHANGELOG.md`):**
   - In `Server.toolProposeRule`, split the level validation check: reject `strings.Contains(level, "..")` with `"invalid level %q: path traversal is not allowed"` and `strings.ContainsAny(level, "/\\")` with `"invalid level identifier %q: path separators are not allowed"`.
   - In `docs/design/gradus-design.md`, mark §24.3 as `**[Validated]**` referencing `wallet-tracker` and mark `O-1` in §27 as closed. Add `## [0.4.2]` to `CHANGELOG.md`.

## Options Considered

- **Anti-tamper shell analysis:** Extending the anchored Go `regexp` (`hookBypassEnvRe`) vs. adding a full POSIX shell parser dependency (`mvdan.cc/sh`). Kept zero-dependency regex matching per `NFR1` and Gradus's minimal-dependency principle; the anchored statement-prefix pattern covers `export`, `env`, and standalone assignments cleanly.
- **Malformed `gradus.yaml` in `gradusctl guard`:** Allowing agents to edit `gradus.yaml` when parsing fails vs. keeping `guard` fail-closed (`exit 2`) and documenting manual repair. Kept fail-closed per Constraint `C1`, Hard Rule #4, and Threat Model T2 (`gradus.yaml` is a protected governance file that must not be auto-edited by an agent to recover from or induce a parse error).

## Simplicity And Elegance Review

Every fix modifies only the exact function responsible for the defect (`hookBypassEnvRe`, `resolveWorkspaceRoot`, `newStatusCmd`, `toolProposeRule`), reusing existing helpers and error types with zero new packages, flags, or dependencies.

## Failure Modes And Tradeoffs

- **Regex shell approximation:** A commit message containing an embedded delimiter immediately followed by a bypass assignment (e.g., `git commit -m "note; HUSKY=0"`) may conservatively trigger an `ask` prompt. This is an accepted fail-safe tradeoff over shipping a full shell AST parser.
- **Malformed `gradus.yaml` fail-closed:** Developers who introduce a YAML syntax error in `gradus.yaml` must fix the file manually outside the agent loop; `status --hook` still exits `0` with `lock=invalid` so session startup is never blocked.

## Verification Plan

- **Unit & race tests (`internal/guard`, `internal/mcp`, `internal/cli`):**
  - `TestAntiTamperExportAndStandaloneV042` in `internal/guard/guard_test.go`: asserts `R1.AC1` (`export HUSKY=0; git commit -m x`, `export HUSKY=0 && git commit -m x`, `export SKIP=gradus`, `set -a; HUSKY=0; git commit`) and `R1.AC2` (`echo SKIP=foo`, `grep LEFTHOOK=0 README.md`, `git commit -m "document HUSKY=0"`).
  - `TestNestedGitBoundaryAndInvalidConfigV042` in `internal/cli/v042_hardening_test.go`: asserts `R2.AC1` (nested `.git` repo inside Gradus parent does not inherit parent rules), `R2.AC2` (`.agents/` and `src/pkg` subdirectories still inherit parent rules), `R3.AC1` (`status --hook` and `status --antigravity-hook` exit `0` with `lock=invalid` on malformed `gradus.yaml`), and `R3.AC2` (non-hook `status` and `guard` exit `2` on malformed `gradus.yaml`).
  - `TestMCPProposeRuleLevelDiagnosticsV042` in `internal/mcp/server_test.go`: asserts `R4.AC1` (`../etc` vs `unknown/level` distinct error messages).
  - `TestDocsAndDesignClosureV042` in `internal/cli/v042_hardening_test.go`: asserts `R3.AC3` (`README.md` manual repair note) and `R4.AC2` (`docs/design/gradus-design.md` §24.3 validated + `O-1` closed, `CHANGELOG.md` `v0.4.2`).
- **Build & repository checks:** `CGO_ENABLED=0 go build`, `go mod tidy -diff`, `go test -race -count=1 ./...`, `gradusctl verify --json`, `gradusctl check --json`, `walden validate --all --json`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `hookBypassEnvRe` in `internal/guard/antitamper.go`; verified by `TestAntiTamperExportAndStandaloneV042` (`R1.AC1`, `R1.AC2`). |
| `R2` | `resolveWorkspaceRoot` in `internal/guard/payload.go` + `findGradusRoot` in `internal/cli/guard.go`; verified by `TestNestedGitBoundaryAndInvalidConfigV042` (`R2.AC1`, `R2.AC2`). |
| `R3` | `newStatusCmd` in `internal/cli/status.go`, `newGuardCmd` in `internal/cli/guard.go`, and `README.md`; verified by `TestNestedGitBoundaryAndInvalidConfigV042` and `TestDocsAndDesignClosureV042` (`R3.AC1`, `R3.AC2`, `R3.AC3`). |
| `R4` | `Server.toolProposeRule` in `internal/mcp/server.go`, `docs/design/gradus-design.md`, and `CHANGELOG.md`; verified by `TestMCPProposeRuleLevelDiagnosticsV042` and `TestDocsAndDesignClosureV042` (`R4.AC1`, `R4.AC2`). |
| `NFR1` | Static `CGO_ENABLED=0` build, `go mod tidy -diff`, and `go test -race -count=1 ./...`. |
| `NFR2` | Canonical English across all modified code, tests, specs, and docs. |
