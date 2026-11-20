---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T09:29:51Z
last_modified: 2026-11-20T09:29:51Z
approved_fingerprint: sha256:9b1b61ebc3c7284d75f9c7af6320a192f493ce6c7f72c2d945fee9b25380ba14
source_requirements_approved_at: 2026-11-20T09:28:07Z
source_requirements_fingerprint: sha256:d50c71049f40a7a8059bf4c0793319f121edf2c5e5564b5217c8be7fbb181021
---

# Feature Design

## Architecture

Feature **F3 (`agent-compilers`)** introduces `internal/compile` and `internal/cli/compile.go`:

1. **`internal/compile/managed.go` (Idempotent Rendering & Managed Sections):**
   - `RenderManagedBlock(viewHash string, body string) string`: wraps `body` with `<!-- gradus:begin view_hash=<hash> -->` and `<!-- gradus:end -->`.
   - `ApplyManagedSection(existing string, block string) string`:
     - If `existing` is empty, returns `block + "\n"`.
     - If both `<!-- gradus:begin` and `<!-- gradus:end -->` markers exist, replaces strictly between them (inclusive), preserving all surrounding human prose byte-for-byte.
     - Otherwise appends `\n` + `block + "\n"` at the end.
   - `WriteIfChanged(path string, content []byte) (changed bool, err error)`: compares on-disk bytes with `content`; skips `os.WriteFile` if identical so repeated `compile` runs preserve timestamps and produce zero diffs (`R4.AC1`).
   - `PruneStaleGradusFiles(dir string, prefix string, keep map[string]bool) ([]string, error)`: removes orphaned `gradus-*.md` files (e.g., when Antigravity chunk count shrinks from 3 to 1) without touching non-`gradus-` user files.
   - `RenderRuleMarkdown(r resolve.ResolvedRule) string`: formats rule ID, title, severity, level, provenance, text, and rejected rationalizations; shared across targets.
   - **Explicit precedence header (§6.3):** every compiled instruction block includes a short header stating the pyramid level order (from highest authority to lowest) and that lower levels can only restrict, never relax, upper levels.

2. **`internal/compile/agentsmd.go` (`AGENTS.md` Target):**
   - Filters `view.Rules` to non-`user`, non-waived active rules (ADR 0002).
   - Builds the managed block and writes `AGENTS.md` via `ApplyManagedSection` + `WriteIfChanged`.

3. **`internal/compile/claudecode.go` (`claude-code` Target):**
   - Global non-`user` rules -> `CLAUDE.md` managed section.
   - Scoped non-`user` rules (`scope != ["**/*"]`) -> `.claude/rules/gradus-<id>.md` with `paths:` YAML frontmatter.
   - `user`-level rules -> `CLAUDE.local.md` (git-ignored).
   - `.claude/settings.json`: reads existing JSON (if present) into `map[string]any`, merges static `PreToolUse` (`gradusctl guard`) and `SessionStart` (`gradusctl status --brief`) hook dispatchers (ADR 0003) plus `permissions.deny`/`permissions.ask` entries for project `enforcement: permission` rules without removing unrelated user keys, and writes deterministic indented JSON via `WriteIfChanged`.
   - Extracts embedded skills from `skills.FS` into `.claude/skills/gradus-*/SKILL.md` (skipping `gradus-placeholder` unless explicitly requested).

4. **`internal/compile/antigravity.go` (`antigravity` Target):**
   - Renders `AGENTS.md` managed section (shared with `agents-md` target).
   - Separates project/team rules (`gradus-<level>-<nnn>.md` / `gradus-scoped-<nnn>.md`) from `user`-level rules (`gradus-user-<nnn>.md`, git-ignored).
   - Groups by `scope`: global `["**/*"]` -> `trigger: always_on`; scoped -> `trigger: glob` with `globs: [...]`.
   - **Size-limit splitter (`R3.AC2`, E1):** chunks rules into sequential files (`001.md`, `002.md`, ...) whenever adding the next rule would exceed `11,500` characters, guaranteeing every file stays strictly under `12,000` characters; prunes stale `gradus-*.md` files via `PruneStaleGradusFiles`.
   - `.agents/hooks.json`: reads existing JSON (if present) and merges static `PreToolUse` (`gradusctl guard`) and `SessionStart` (`gradusctl status --brief`) dispatchers (ADR 0003) without embedding any user-level rule patterns.
   - Extracts embedded skills from `skills.FS` into `.agents/skills/gradus-*/SKILL.md` (skipping `gradus-placeholder`).

5. **`internal/compile/compile.go` & `internal/cli/compile.go`:**
   - `Compile(rootDir string, view resolve.ResolvedView, targets []string, skillsFS fs.FS) (CompileResult, error)`: plans all target files in memory (so `gradusctl verify` in F4 can reuse the exact same expected-file map for drift detection!), then writes changed files and prunes stale `gradus-*` files.
   - `gradusctl compile [--dir] [--json]`: runs `resolve.Resolve`; exits `2` on error, refuses to write and exits `1` if `len(view.Conflicts) > 0`, otherwise runs `compile.Compile` and exits `0`.

## Options Considered

- **In-memory file plan (`PlanCompile` + `ApplyPlan`) vs direct disk writes:** Chose producing an in-memory `map[string][]byte` plan first and applying it via `WriteIfChanged`, because F4 (`gradusctl verify` / `reconcile`) can directly compare on-disk files against `PlanCompile` to detect drift with zero duplicated rendering logic.
- **JSON settings merge:** Chose `map[string]any` structural merge with idempotent Gradus marker/command deduplication over overwriting `.claude/settings.json` / `.agents/hooks.json`, preserving user hooks and unrelated settings.

## Simplicity And Elegance Review

- All three targets share `RenderRuleMarkdown`, `WriteIfChanged`, `PruneStaleGradusFiles`, and `extractSkills`.
- `PlanCompile` is pure and deterministic (no clock timestamps, sorted map/slice iteration), making idempotency tests trivial.

## Failure Modes And Tradeoffs

- **Single rule exceeding 11,500 characters:** If a single rule's rendered Markdown alone exceeds 12,000 characters, `antigravity.go` returns a clear error (exit code `2`) rather than silently truncating policy text.
- **Malformed existing `.claude/settings.json` or `.agents/hooks.json`:** Returns an explicit JSON parse error (exit code `2`) rather than overwriting or corrupting user configuration.

## Verification Plan

- **`internal/compile` tests:**
  - `TestManagedSectionsAndIdempotency`: preserves prose around markers, skips disk write on identical second run.
  - `TestClaudeCodeCompiler`: verifies `CLAUDE.md`, `.claude/rules/gradus-*.md`, `CLAUDE.local.md`, `.claude/settings.json` merge, and `.claude/skills/` extraction.
  - `TestAntigravityCompilerAndSplit`: verifies `trigger: always_on`/`glob`, `gradus-user-*.md` separation, `> 25,000`-char rule set splitting strictly `< 12,000` chars per file, orphan cleanup, and `.agents/hooks.json` static dispatcher.
- **`internal/cli` tests:**
  - `TestCompileCommand`: end-to-end `compile --json` idempotency (2nd run `changed_files` empty), conflict block (`exit 1`, files untouched), and error handling (`exit 2`), plus `CGO_ENABLED=0` static build check.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `internal/compile/managed.go` & `agentsmd.go` (markers, prose preservation, ADR 0002 rendering) |
| `R2` | `R2.AC1`–`R2.AC3`: `internal/compile/claudecode.go` (`CLAUDE.md`, `.claude/rules/`, `CLAUDE.local.md`, `.claude/settings.json`, `.claude/skills/`) |
| `R3` | `R3.AC1`–`R3.AC3`: `internal/compile/antigravity.go` (`AGENTS.md`, `.agents/rules/` `<11.5k` split, `gradus-user-*.md`, `.agents/hooks.json`, `.agents/skills/`) |
| `R4` | `R4.AC1`–`R4.AC3`: `internal/cli/compile.go` (`gradusctl compile`, idempotent `WriteIfChanged`, exit `0`/`1`/`2`) |
| `NFR1` | Deterministic sorting + `WriteIfChanged` zero-diff second run |
| `NFR2` | `skills.FS` `//go:embed` extraction with zero Node/npm runtime dependency |
| `NFR3` | Exit codes `0` (clean), `1` (blocking conflicts), `2` (config/lock/JSON merge errors) |
