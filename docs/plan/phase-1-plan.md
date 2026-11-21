# Gradus — Phase 1 Master Implementation Plan ("Team")

| Field | Value |
|---|---|
| Project | **Gradus** (`github.com/Milosauro/gradus-ai`) |
| Design baseline | `docs/design/gradus-design.md` (§6.6, §13, §16, §21, §23, §24.1, §25) |
| Previous milestone | Phase 0 MVP (`v0.1.0`–`v0.1.4`, all 8 Walden features `F1`–`F8` certified) |
| Target milestone | **Phase 1 — Team (`v0.2.0`)** |
| SDD Workflow | Walden (`walden v0.14.0+`) |
| Status | Complete (`F9`–`F14` implemented and verified; ready for `v0.2.0` release) |

---

## 1. Objectives & Design Alignment

Phase 1 extends Gradus from a single-developer kit into a **team-governed pyramid** while preserving every Phase 0 contract:

1. **Additive-only schema (`gradus/v1`):** Reuse existing parsed structs in `internal/schema/types.go` (`Charter`, `ExceptionPolicy`, `OverlayRef`, `OverlayConfig`, `ExamplePair`, `Rule.Status`, `Rule.ReviewBy`) and keep unknown YAML/JSON keys non-fatal (§9.2).
2. **All 5 Phase 1 CLI commands from §16 included:** `gradusctl exception`, `gradusctl test`, `gradusctl simulate`, `gradusctl doctor`, and `gradusctl handbook`.
3. **Field-testing lessons from Phase 0 (`wallet-tracker` on Antigravity & Claude Code) integrated:**
   - Configurable anti-tamper policy (`ask` vs `deny`) so teams running Claude Code in `auto` mode can enforce hard blocks on governance files.
   - `gradusctl doctor` to validate agent hook files (`.claude/settings.json`, `.agents/hooks.json`), `gradusctl` PATH availability, lock state, and permission modes.
   - Hygiene subcommands (`evidence prune`, `lessons prune`) and clear visibility of effective `user`-level rules in `resolve` / `status`.

---

## 2. Phase 1 Feature Breakdown (Walden Features `F9`–`F14`)

### Wave 4 — Team Governance Core (`F9`–`F11`)

- **`F9` `charters-and-lifecycle` (§6.6, X-07, D-28, D-36):**
  - Enforce `level.yaml` charter rules using the existing `schema.Charter` fields (`purpose`, `includes`, `excludes`, `examples_in`, `examples_out`, `allowed_severities`, `review_cadence`):
    - Deterministic enforcement of `allowed_severities` during `gradusctl resolve` and `gradusctl verify` (exit `1` when a rule exceeds its level's allowed severities).
    - Scope/topic classification guidance in `gradusctl explain`, `gradus-author`, `gradus-onboard`, and `gradus-review` using `includes` / `excludes`.
  - Rule lifecycle & per-level context budget (X-07):
    - Validate `status` (`active`, `draft`, `deprecated`, `retired`), `review_by`, and optional `expires` dates in `gradusctl verify` (filtering `draft`/`retired` rules from compiled output and flagging overdue `review_by` dates).
    - Optional per-level context budget (`context_budget` in `level.yaml`: `max_rules`, `max_chars`) verified deterministically during `resolve` / `verify`.

- **`F10` `overlays-and-exceptions` (§7, §13, §16 `exception`, X-08):**
  - Explicit overlay activation (`activation: explicit` in `gradus.yaml` `overlays:`):
    - Enforce `constraints.allowed_overlays` from upper levels, verify overlay lock hashes in `gradus.lock`, and enforce the P2 restriction invariant according to overlay `strength`.
  - CLI workflow `gradusctl exception` (`request | approve | list | expire | break-glass`) managing `.gradus/exceptions/*.yaml`:
    - Support `kind: exception` and `kind: break-glass` (hotfix-style emergency bypass).
    - Enforce `level.yaml` `exceptions` policy (`max_duration`, `approvers`, `break_glass: true|false` — disallowing break-glass on levels where `break_glass: false` or `fail-closed`).
    - Require mandatory `reason`, short `expires` duration, owner notification metadata, and mandatory `post_review_due` date for `kind: break-glass`, failing closed in `gradusctl verify` when `expires` or `post_review_due` has passed.

- **`F11` `doctor-and-guard-hardening` (§16 `doctor`, Field-Testing Feedback):**
  - New `gradusctl doctor [--json]` command (§16):
    - Diagnose `gradus.yaml` / `gradus.lock` integrity, verify `gradusctl` binary resolution on `PATH`, validate syntax and schema of `.claude/settings.json` and `.agents/hooks.json` (catching malformed hook entries such as unknown hook types), and report active Claude Code / Antigravity hook readiness with actionable fix instructions.
  - Configurable `anti_tamper` policy (`ask` default vs `deny` strict) so repositories using Claude Code `auto` mode can hard-block agent edits or shell mutations targeting governance and hook files.
  - Hygiene commands (`gradusctl evidence prune`, `gradusctl lessons prune`) and explicit `user`-level rule summary in `gradusctl status` / `resolve`.

*(Run `walden consolidate` after `F11`.)*

### Wave 5 — Testing, CI & Team Adoption (`F12`–`F14`)

- **`F12` `rule-tests-and-impact` (§9.4, §16 `test` & `simulate`, §23, X-03):**
  - `gradusctl test [--rule <id>] [--json]`:
    - Execute deterministic rule tests directly from the `### Compliant` and `### Non-compliant` code blocks already parsed into `schema.Rule.Examples` (`ExamplePair`) in rule Markdown files (`rules/*.md`).
    - Verify that `check` rules (built-in pattern and external checkers) and `hook` rules (`guard` patterns) pass all `Compliant` examples and flag all `Non-compliant` examples (exit `0` if all pass, exit `1` on test failure, exit `2` on missing checker/error).
  - `gradusctl simulate [--since <ref|N>] [--json]`:
    - Apply the current (or candidate) rule set across recent git commits/diffs via `os/exec` `git` and report which rules would have triggered, on which commits/files, and the resulting impact before rolling out a rule to a shared `team` level.

- **`F13` `github-action-and-ci-agents` (§12, §21, §24.1, X-09):**
  - Reusable composite GitHub Action (`.github/actions/gradus/action.yml`, with 40-char SHA-pinned steps per `PRJ-CI-001`) executing `gradusctl verify` and `gradusctl check --base <ref>`, emitting evidence artifacts, and writing a GitHub Step Summary.
  - Background and CI agent profiles (§21): support `--agent-profile <path|id>` (or `profile.agent` in headless/CI mode) so background/CI agents replace the personal `user` level with a team-defined agent profile (e.g., draft-PR-only guardrails).

- **`F14` `handbook-and-templates` (§16 `handbook`, X-11, X-12, D-38):**
  - `gradusctl handbook [--output <path>] [--json]`:
    - Deterministically generate a human onboarding handbook in Markdown from the resolved pyramid (levels, charters, owners/approvers, active rules grouped by level/scope/enforcement, active overlays, and exception/break-glass instructions).
  - Non-prescriptive starter templates under `examples/` (`personal`, `team`, and sector overlays) verified by automated tests.

*(Run `walden consolidate` after `F14` and certify `v0.2.0` release.)*

---

## 3. Phase 1 Exit Criteria (`P1-E1`–`P1-E6`)

| # | Criterion |
|---|---|
| `P1-E1` | A shared `team` level enforces `charter.allowed_severities`, rule lifecycle (`status`, `review_by`), and `context_budget` limits via `gradusctl verify`. |
| `P1-E2` | Explicit overlays and `gradusctl exception` (including `kind: break-glass` with `post_review_due` and `max_duration`) resolve deterministically and fail closed when expired or disallowed. |
| `P1-E3` | `gradusctl doctor` diagnoses corrupted hook configs (`.claude/settings.json`, `.agents/hooks.json`) and `anti_tamper: deny` hard-blocks governance file edits even in Claude Code `auto` mode. |
| `P1-E4` | `gradusctl test` validates rules against their own `### Compliant` / `### Non-compliant` Markdown examples; `gradusctl simulate --since <ref>` reports impact across recent git commits. |
| `P1-E5` | The reusable GitHub Action and CI agent profile (`--agent-profile`) run `verify` + `check` in CI and store evidence artifacts. |
| `P1-E6` | `gradusctl handbook` generates a complete onboarding handbook from the resolved pyramid, and all `examples/` templates pass `gradusctl verify` and `gradusctl test`. |
