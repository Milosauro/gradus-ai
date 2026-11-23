---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T16:02:49Z
last_modified: 2026-11-23T16:02:49Z
approved_fingerprint: sha256:d3a8639242e781345644f35eec98ba639927d800353ce002ed60720aa01f60ba
source_requirements_approved_at: 2026-11-23T15:52:48Z
source_requirements_fingerprint: sha256:f59362cce6b34db26330285cbe9d6b98d960cf6022ee2da3115618e1faf6be1e
---

# Feature Design

## Architecture

This feature adds three new compiler targets (`codex`, `copilot`, `opencode`) to `internal/compile` and documents all six targets in `docs/compatibility-matrix.md`:

1. **Codex Compiler (`internal/compile/codex.go`):**
   - `PlanCodex(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error)`:
     - Reads optional `AGENTS.md`, generates managed section via `PlanAgentsMD(existing, view)` (non-user active rules only).
     - Extracts embedded Gradus skills into `.agents/skills/gradus-*/SKILL.md` via `PlanEmbeddedSkills`.

2. **GitHub Copilot Compiler (`internal/compile/copilot.go`):**
   - `PlanCopilot(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error)`:
     - Global non-user active rules rendered into `.github/copilot-instructions.md` inside `<!-- gradus:begin view_hash=... --> ... <!-- gradus:end -->` via `RenderManagedBlock` + `ApplyManagedSection` (preserving surrounding human prose). When scoped project rules exist, appends a concise pointer to `.github/instructions/gradus-*.instructions.md`.
     - Non-global (path-scoped) active project rules emitted to `.github/instructions/gradus-<id>.instructions.md` with YAML frontmatter:
       ```yaml
       ---
       applyTo: "<glob1>,<glob2>"
       ---
       ```
     - Embedded Gradus skills extracted into `.github/skills/gradus-*/SKILL.md` via `PlanEmbeddedSkills`.

3. **OpenCode Compiler (`internal/compile/opencode.go`):**
   - `PlanOpenCode(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error)`:
     - Reads optional `AGENTS.md`, generates managed section via `PlanAgentsMD(existing, view)`.
     - Extracts embedded Gradus skills into `.opencode/skills/gradus-*/SKILL.md` via `PlanEmbeddedSkills`.

4. **Dispatcher & Pruning (`internal/compile/compile.go`):**
   - Add `"codex"`, `"copilot"`, and `"opencode"` cases in `PlanCompile` (preserving unknown-target error -> exit `2`).
   - In `ApplyCompile`, track `copilotKeep` for `.github/instructions/gradus-*.instructions.md` and prune stale `gradus-*.instructions.md` files via `PruneStaleGradusFiles` when `"copilot"` is targeted (leaving human-written `*.instructions.md` untouched).
   - Because `verify`, `status`, `doctor`, and `reconcile` all delegate to `compile.PlanCompile`, drift detection works automatically for all three new targets (`C2`).

5. **Compatibility Matrix (`docs/compatibility-matrix.md`):**
   - Tabular agent × feature matrix covering `claude-code`, `antigravity`, `agents-md`, `codex`, `copilot`, and `opencode` across instructions, scoped rules, embedded skills, `PreToolUse` runtime guard, `SessionStart` hook, and permissions.

## Options Considered

- **Chosen:** Separate focused compiler files (`codex.go`, `copilot.go`, `opencode.go`) reusing `PlanAgentsMD`, `RenderManagedBlock`, `ApplyManagedSection`, `PlanEmbeddedSkills`, and `PruneStaleGradusFiles`.
  - Mirrors `claudecode.go` and `antigravity.go`, keeps each target's conventions isolated, and guarantees automatic integration with `verify`/`status`/`doctor`/`reconcile`.
- **Alternative:** Alias `codex` and `opencode` directly to `agents-md` without extracting skills.
  - Rejected: Codex (`.agents/skills/`) and OpenCode (`.opencode/skills/`) both support Agent Skills natively; extracting skills unlocks `gradus-review`, `gradus-author`, `gradus-link`, etc. out of the box.

## Simplicity And Elegance Review

- Zero new dependencies; `PlanCodex` and `PlanOpenCode` are ~25 lines each reusing `PlanAgentsMD` and `PlanEmbeddedSkills`.
- `PruneStaleGradusFiles` already matches prefix `gradus-` and `.md` suffix, so it prunes stale `.github/instructions/gradus-*.instructions.md` files directly without modifying `managed.go`.

## Failure Modes And Tradeoffs

- **Shared `AGENTS.md` across multiple targets (e.g. `targets: [antigravity, codex, opencode]`):** All three render `AGENTS.md` through the exact same deterministic `PlanAgentsMD` function, so `PlanCompile` produces byte-identical content regardless of target order (`NFR1`, `NFR2`).
- **No native `PreToolUse` hooks for Codex/Copilot/OpenCode:** Documented clearly in `docs/compatibility-matrix.md`; deterministic enforcement for those agents relies on git hooks (`gradusctl hooks`) and CI (`gradusctl verify` / `gradusctl check`).

## Verification Plan

- Unit tests in `internal/compile/extra_targets_test.go` covering `PlanCodex`, `PlanCopilot`, `PlanOpenCode`, human-prose preservation, `applyTo` frontmatter, stale `.github/instructions/gradus-*.instructions.md` pruning, and unknown target rejection (`R1.AC1`, `R2.AC1`, `R2.AC2`, `R3.AC1`, `R4.AC3`, `NFR1`, `NFR2`).
- CLI end-to-end test in `internal/cli/extra_targets_test.go` covering `gradusctl compile` idempotency (`Changed: 0` on second run) and `gradusctl verify` drift detection on `.github/copilot-instructions.md` (`R4.AC1`, `R4.AC2`, `C2`), plus verifying `docs/compatibility-matrix.md` lists all six targets (`R4.AC4`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/compile/codex.go`, `internal/compile/compile.go` (`R1.AC1`) |
| `R2` | `internal/compile/copilot.go`, `internal/compile/compile.go` (`R2.AC1`, `R2.AC2`) |
| `R3` | `internal/compile/opencode.go`, `internal/compile/compile.go` (`R3.AC1`) |
| `R4` | `internal/compile/compile.go`, `internal/cli/extra_targets_test.go`, `docs/compatibility-matrix.md` (`R4.AC1`–`R4.AC4`) |
| `NFR1` | Deterministic sorting and `WriteIfChanged` idempotency tested across all 3 new targets |
| `NFR2` | Existing `claude-code`, `antigravity`, and `agents-md` compiler tests unchanged and passing |
| `C1` | Reuses `PlanAgentsMD`, `RenderManagedBlock`, `ApplyManagedSection`, `PlanEmbeddedSkills`, `PruneStaleGradusFiles` |
| `C2` | Wired through `compile.PlanCompile` consumed by `verify`, `status`, `doctor`, and `reconcile` |
