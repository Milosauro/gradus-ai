---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T11:04:10Z
last_modified: 2026-11-20T11:04:10Z
approved_fingerprint: sha256:efdac13fe2edbc12872284d62d723965fb24f9014f76fc541468247e53e347b0
source_requirements_approved_at: 2026-11-20T11:00:12Z
source_requirements_fingerprint: sha256:9e7207caba6ad3c1e4c76a31e10c78a6efe997c064b7c3eaa81cf68344808ee6
---

# Design Document

## Architecture

Feature **F5 (`runtime-guard`)** introduces the `internal/guard` package and the `gradusctl guard` Cobra subcommand (`internal/cli/guard.go`), wired to the static `PreToolUse` hook dispatchers already emitted by F3 (`internal/compile/claudecode.go` and `internal/compile/antigravity.go`).

### Component Overview

1. **`internal/guard/payload.go` — Multi-Agent `PreToolUse` Payload Normalizer**
   - Defines `HookPayload` (tolerant JSON structure accepting `tool_name` / `tool` and `tool_input` / `input` maps) and `NormalizedAction`:
     - `ToolName string` (raw tool name)
     - `Action string` (`"command"` | `"write"` | `"other"`)
     - `Target string` (shell command string for `"command"`, or clean repo-relative slash path for `"write"`)
   - Maps agent tool names:
     - **Command tools:** `Bash`, `run_command` → `Action: "command"`; extracts `tool_input["command"]` or `tool_input["CommandLine"]`.
     - **Write/Edit tools:** `Edit`, `Write`, `MultiEdit`, `write_to_file`, `replace_file_content`, `multi_replace_file_content`, `write_file`, `edit_file` → `Action: "write"`; extracts `file_path`, `path`, `TargetFile`, `Path`.
     - Normalizes file paths via `NormalizeTargetPath(rootDir, rawPath)`: converts `\` to `/`, resolves workspace-rooted absolute paths to clean repo-relative slash paths, trims leading `./`, so anti-tamper and glob matching work identically on macOS, Linux, and Windows.
   - Rejects empty input, invalid JSON, or payloads missing a tool name with an error (mapped to exit code `2`).

2. **`internal/guard/antitamper.go` — Built-In Anti-Tamper Protection (Threat Model T2)**
   - `IsProtectedGovernancePath(relPath string) bool` checks a normalized slash path against the built-in protected set using `doublestar.Match`:
     - `gradus.yaml`
     - `gradus.lock`
     - `.gradus/**`
     - `.walden/**`
     - `.agents/hooks.json`
     - `.claude/settings.json`
   - When `Action == "write"` and `IsProtectedGovernancePath(Target)` is true, injects a built-in floor decision of `ask` with `AntiTamper: true` and an explicit reason explaining that governance files require human confirmation.

3. **`internal/guard/evaluator.go` — Guard Evaluation Engine & Precedence**
   - `Decision` constants: `DecisionAllow = "allow"`, `DecisionAsk = "ask"`, `DecisionDeny = "deny"`.
   - `EvaluationResult`:
     ```go
     type EvaluationResult struct {
         Decision           string              `json:"decision"`
         Reason             string              `json:"reason"`
         MatchedRules       []MatchedRule       `json:"matched_rules"`
         AntiTamper         bool                `json:"anti_tamper"`
         HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput"`
     }
     ```
     where `HookSpecificOutput` provides `hookEventName: "PreToolUse"`, `permissionDecision`, and `permissionDecisionReason` for direct agent-runtime consumption.
   - `Evaluate(rootDir string, action NormalizedAction, rules []resolve.ResolvedRule) (EvaluationResult, error)`:
     - Starts at `allow`.
     - Applies built-in anti-tamper floor (`ask`) for protected governance paths (`write` actions).
     - Iterates all rules where `!rule.Waived && rule.Enforcement == "hook" && rule.Guard != nil`:
       - Checks action compatibility (`command`/`bash` matches `command`; `write`/`edit` matches `write`).
       - For `write` actions, checks rule `scope` (using `doublestar.Match`, treating empty or `**/*` as global).
       - Compiles and matches `guard.pattern`:
         - For `command` actions: `regexp.Compile(rule.Guard.Pattern)` against `action.Target` (invalid regex returns a technical error → exit `2`).
         - For `write` actions: matches via `doublestar.Match` first; if the pattern is not a glob or fails glob match, falls back to `regexp.Compile` (or compiles regex when pattern has regex metacharacters) so both glob patterns (`secrets/**`) and regex patterns work predictably.
       - Escalates decision strictly according to `rank(deny)=2 > rank(ask)=1 > rank(allow)=0`, appending every matched rule to `MatchedRules`.
   - `LoadLocalGuardRules(rootDir, userLevelDir string) ([]resolve.ResolvedRule, error)`:
     - Calls `resolve.Resolve(rootDir, resolve.Options{OfflineOnly: true})` so zero network calls occur and locked-level tampering fails closed.
     - If the resolved view does not already include the `user` level, checks `userLevelDir` (defaulting to `~/.gradus/levels/user`, overridable for tests) and loads any `rules/*.md` files via `schema.LoadRulesDir(userDir, "user")` so personal `enforcement: hook` rules (ADR 0003) are always enforced locally without mutating shared repository files.

4. **`internal/resolve/resolve.go` & `internal/source/git.go` — Offline-Only Fast Path**
   - Adds `OfflineOnly bool` to `resolve.Options` (default `false` so existing callers are unaffected).
   - When `OfflineOnly` is `true`, remote git sources read strictly from the local cache directory without invoking `git fetch` or `git clone`, guaranteeing `< 100ms` latency and zero network I/O in `gradusctl guard`.

5. **`internal/cli/guard.go` — `gradusctl guard` Command**
   - Registered on `NewRootCmd` in `internal/cli/root.go`.
   - Flags: `--user-dir` (optional override for `~/.gradus/levels/user`, useful for testing and custom environments).
   - Reads `cmd.InOrStdin()`, calls `guard.ParsePayload`, `guard.LoadLocalGuardRules`, and `guard.Evaluate`.
   - Writes the JSON `EvaluationResult` to `cmd.OutOrStdout()`.
   - Exit codes (D-73):
     - `allow` or `ask` → `0`
     - `deny` → `1` (via `ExitError{Code: 1, Silent: true}` since JSON verdict was already printed)
     - Technical error (malformed JSON, invalid regex, missing/invalid `gradus.yaml`, lock tamper) → `2` (via `ExitError{Code: 2}`).

## Options Considered

1. **Cached Serialized View File vs. Direct Local Offline Resolution**
   - *Option A:* Require a separate pre-compiled `.gradus/cache/view.json` written by `gradusctl resolve` and fail if missing.
   - *Option B (Chosen):* Run `resolve.Resolve(rootDir, resolve.Options{OfflineOnly: true})` directly on local files and cached git directories, plus load `~/.gradus/levels/user` if present.
   - *Rationale:* Local pyramid resolution in Go takes `< 5ms` for typical repositories, avoids stale-cache bugs when a developer edits a personal rule in `~/.gradus/levels/user`, and automatically enforces lock integrity verification on every guard invocation.

2. **Agent-Specific Output Flags vs. Dual-Compatible JSON Envelope**
   - *Option A:* Require `--agent claude` or `--agent antigravity` flags in `.claude/settings.json` and `.agents/hooks.json`.
   - *Option B (Chosen):* Emit a single compact JSON object containing both Gradus top-level fields (`decision`, `reason`, `matched_rules`, `anti_tamper`) and `hookSpecificOutput` (`hookEventName`, `permissionDecision`, `permissionDecisionReason`).
   - *Rationale:* Matches the static `gradusctl guard` command already emitted by F3 (`claudecode.go` and `antigravity.go`) with zero configuration flags required.

## Simplicity And Elegance Review

- **Zero Duplication:** Reuses `internal/resolve`, `internal/schema`, and `doublestar/v4` directly; adds only one boolean field (`OfflineOnly`) to `resolve.Options`.
- **Clean Separation:** `internal/guard` is pure Go with no Cobra dependency (`payload.go`, `antitamper.go`, `evaluator.go`), making unit testing fast and deterministic across macOS, Linux, and Windows.
- **Cross-Platform Path Safety:** Normalizing Windows backslashes and workspace-rooted absolute paths in one helper (`NormalizeTargetPath`) eliminates OS-specific anti-tamper bypasses.

## Failure Modes And Tradeoffs

- **Malformed or Empty `stdin`:** Fails closed with exit code `2` and stderr diagnostic; never defaults to `allow`.
- **Invalid Guard Regex in a Rule:** Fails closed with exit code `2` naming the offending rule ID so broken security rules cannot silently stop matching.
- **Unprimed Remote Git Cache in `OfflineOnly` Mode:** If a `fail-closed` remote level has never been fetched into the local cache, `OfflineOnly` resolution fails closed with exit code `2`, prompting the user to run `gradusctl resolve`.
- **Tradeoff (No Evidence Disk Writes in Hot Path):** `gradusctl guard` intentionally does not write `.gradus/evidence/` JSON files on every tool call to stay well below the `< 100ms` latency budget and avoid disk churn.

## Verification Plan

1. **Unit Tests (`internal/guard/guard_test.go`):**
   - `TestParsePayload`: verifies Claude Code (`Bash`, `Edit`, `Write`, `MultiEdit`) and Antigravity (`run_command`, `write_to_file`, `replace_file_content`, `multi_replace_file_content`) payload normalization, Windows/absolute path normalization, and error handling on empty/malformed JSON.
   - `TestAntiTamperProtection`: verifies built-in `ask` floor on all 6 protected governance paths (`gradus.yaml`, `gradus.lock`, `.gradus/**`, `.walden/**`, `.agents/hooks.json`, `.claude/settings.json`), non-protected paths returning `allow`, and escalation to `deny` when an explicit `deny` rule also matches.
   - `TestGuardEvaluatePrecedenceAndUserLevel`: verifies `command` and `write` rule matching, waived rule skipping, `deny > ask > allow` precedence with multiple matched rules, personal `~/.gradus/levels/user` hook rule loading without git-tracked config changes, invalid regex error reporting, and `< 100ms` evaluation latency.
2. **CLI Integration Tests (`internal/cli/guard_test.go`):**
   - `TestGuardCommand`: runs `gradusctl guard` end-to-end via `NewRootCmd` with piped `stdin` for `allow` (exit `0`), `ask` via user-level hook and anti-tamper (exit `0`), `deny` (exit `1`), and malformed JSON (exit `2`), verifying JSON output fields (`decision`, `matched_rules`, `anti_tamper`, `hookSpecificOutput`).

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC3`: `internal/guard/payload.go` (`ParsePayload` Claude Code & Antigravity normalization, fail-closed exit `2`) |
| `R2` | `R2.AC1`–`R2.AC2`: `internal/guard/antitamper.go` + `internal/guard/evaluator.go` (`IsProtectedGovernancePath` `ask` floor + `deny` escalation) |
| `R3` | `R3.AC1`–`R3.AC3`: `internal/guard/evaluator.go` + `internal/resolve/resolve.go` (`LoadLocalGuardRules`, `OfflineOnly`, user level, `deny > ask > allow`) |
| `R4` | `R4.AC1`–`R4.AC2`: `internal/cli/guard.go` (JSON envelope + `hookSpecificOutput`, exit codes `0`/`1`/`2`) |
| `NFR1` | `resolve.Options{OfflineOnly: true}` + in-memory evaluation `< 100ms` |
| `NFR2` | `internal/guard/payload.go` (`NormalizeTargetPath` cross-platform slash/root normalization) |
| `NFR3` | Fail-closed error propagation in `internal/guard` and `internal/cli/guard.go` |
