---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T09:01:17Z
last_modified: 2026-11-25T09:01:17Z
approved_fingerprint: sha256:381cbbcd46938f17c39cc376ffa6d1b0edfb72f7e63d08c3b695e5ba4ac23bd5
source_requirements_approved_at: 2026-11-25T09:01:17Z
source_requirements_fingerprint: sha256:73cbc5775fde1679979c3077f2990539b77da2a71eb03d71e56b0f85a3a96f5b
---

# Design: Managed Settings & Chat Skill Bundles (`F26`)

## Architecture

1. **Managed Settings Generator (`internal/compile/managed_settings.go`):**
   - `RenderClaudeManagedSettings(view resolve.ResolvedView, antiTamperMode string) ([]byte, error)`:
     - Builds a clean enterprise Claude Code `managed-settings.json` object:
       - `hooks`: `PreToolUse` (`Bash|Edit|Write|MultiEdit` → `gradusctl guard`) and `SessionStart` (`gradusctl status --brief --hook`).
       - `permissions`: `deny` and `ask` slices compiled from active non-waived `permission` rules (reusing `formatClaudePermission` from `claudecode.go`).
       - `deniedMcpServers`: `[{"serverName": "<id>"}]` for each `view.Toolchain.Forbidden` item with `type == "mcp"` (sorted deterministically).
       - `disableBypassPermissionsMode: "disable"` when `antiTamperMode == "deny"`.
2. **Chat Skill Bundle Packager (`internal/compile/chat_bundle.go`):**
   - `WriteChatBundleZIP(outPath string, view resolve.ResolvedView, skillsFS fs.FS) (ChatBundleResult, error)`:
     - Uses stdlib `archive/zip` with sorted entry paths and fixed `Modified: time.Unix(0, 0).UTC()` headers so repeated builds over the same resolved view are byte-for-byte identical.
     - Includes:
       - `manifest.json`: `schema`, `view_hash`, `lock_hash`, `levels` (excluding `user`), `rule_count`, `skills`.
       - `RULES.md`: compiled precedence header + all active non-waived non-`user` rules.
       - `skills/<name>/SKILL.md`: all embedded skills from `skillsFS`.
3. **CLI Integration (`internal/cli/compile.go`):**
   - Add optional flags `--managed-settings <path>` and `--chat-bundle <path.zip>` to `gradusctl compile`.
   - Keep `ApplyCompile` and `gradusctl verify` drift detection unchanged so enterprise export artifacts do not create false drift in developer repositories.

## Options Considered

- **Option A (Chosen):** Opt-in `--managed-settings` and `--chat-bundle` flags on `gradusctl compile`. Keeps developer workflows (`compile`/`verify`) clean while letting platform/CI pipelines emit enterprise artifacts from the same resolved view.
- **Option B (Rejected):** Emit `managed-settings.json` and `chat-bundle.zip` unconditionally into the repo root on every `compile`. Pollutes developer repos with binary ZIPs and MDM-only configs.

## Simplicity And Elegance Review

- Pure Go stdlib (`archive/zip`, `encoding/json`) with zero external dependencies and deterministic output.

## Failure Modes And Tradeoffs

- **User-level leakage in chat bundles:** `WriteChatBundleZIP` explicitly strips `user`-level rules and summaries so organization-provisioned ZIPs never leak a maintainer's personal `~/.gradus/levels/user` habits.

## Verification Plan

- Unit tests in `internal/compile/enterprise_test.go` (`TestManagedSettingsGeneration`, `TestChatBundleZIPReproducibility`).
- End-to-end CLI tests in `internal/cli/enterprise_test.go` (`TestManagedSettingsAndChatBundleCLI`).
- `go mod tidy -diff` and `CGO_ENABLED=0 go build`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/compile/managed_settings.go`, `internal/cli/compile.go` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/compile/chat_bundle.go`, `internal/cli/compile.go` (`R2.AC1`, `R2.AC2`) |
| `NFR1` | Stdlib `archive/zip`, `go mod tidy -diff`, static `CGO_ENABLED=0` build |
| `NFR2` | Canonical English across all repository files |
| `C1` | Opt-in flags leaving default `compile`/`verify` unchanged |
| `C2` | Full Go test suite regression pass |
