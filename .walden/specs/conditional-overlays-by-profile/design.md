---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T16:42:52Z
last_modified: 2026-11-24T16:42:52Z
approved_fingerprint: sha256:7bdb937e7f2d5f424b26f6039fab38b989c4e205054bb6e8f7f53f611513cf1d
source_requirements_approved_at: 2026-11-24T16:42:52Z
source_requirements_fingerprint: sha256:2934348402470723fe717d0ab14da2dc2dbd46da10a65d23b8259f3f67c48c82
---

# Design: Conditional Overlay Activation by Profile (`F23`)

## Architecture

1. **Schema Layer (`internal/schema`):**
   - Extend `Profile` in `internal/schema/types.go` with `Extra map[string]any` (`yaml:",inline"` and custom `MarshalJSON`/`UnmarshalJSON` or helper `Lookup(key string) ([]string, bool)`) so both standard (`data`, `exposure`, `domain`, `client`) and arbitrary custom profile keys are preserved and queryable uniformly.
   - Replace `ActivationRules []any` in `LevelConfig` with typed `ActivationRules []ActivationRule` where `ActivationRule` has `Overlay string`, `When map[string]any`, and `Required bool`.
   - Add `When map[string]any` to `OverlayConfig` (`overlay.yaml`).
   - Update `parse_config.go` (`levelFieldSpec`, `overlayFieldSpec`, `pyramidFieldSpec`) so `activation_rules`, `when`, and extensible `profile` keys produce zero unknown-key warnings, and validate `activation_rules` in `validate_config.go`.
2. **Resolution Layer (`internal/resolve/overlays.go` & `internal/resolve/resolve.go`):**
   - Add `internal/resolve/overlays.go` with deterministic predicate evaluation (`MatchProfile(profile *schema.Profile, when map[string]any) bool`):
     - Empty `when` never matches.
     - Every key in `when` must match `profile` (logical AND).
     - For each key, normalize both the `profile` value(s) and `when` expected value(s) to trimmed lowercase strings (supporting string, bool, int/float, and slices of scalars); match succeeds if any `profile` value equals any expected value (logical OR).
   - During `resolve.Resolve`:
     - Collect all `activation_rules` from resolved levels in pyramid order.
     - For each declared overlay in `cfg.Overlays`, determine `active` and `reason`:
       - `""`, `"always"`, `"explicit"` → `active = true`, `reason = "explicit"` (or `"always"`).
       - `"disabled"` → `active = false`, `reason = "disabled"`.
       - `"profile"` → check matching `level.yaml` `activation_rules` (in pyramid order), then `overlay.yaml` `when:`; if matched, `active = true` with `reason = "profile match (level:<id>)"` or `"profile match (overlay.yaml)"`; otherwise `active = false`, `reason = "profile mismatch"`.
     - After evaluating declared overlays, verify every matching `activation_rule` with `Required: true`: if the target overlay is not declared or not active (e.g. `disabled`), emit a blocking `Conflict` (`Kind: "overlay"`).
   - Add `Reason string` (`json:"reason,omitempty"`) to `OverlaySummary` in `internal/resolve/types.go`.
3. **JSON Schemas (`spec/v1/schema/*.json`):**
   - Update `level.schema.json` (`activation_rules`), `overlay.schema.json` (`when`), `gradus.schema.json` (`profile.additionalProperties: true`), and `resolved-view.schema.json` (`overlays[].reason`).

## Options Considered

- **Option A (Chosen):** Declarative key-value `when` maps (AND across keys, OR across list values, case-insensitive scalar normalization) in `level.yaml` `activation_rules` and `overlay.yaml` `when`. Simple, auditable, zero expression-language injection surface, directly matches `gradus-design.md` §9.3 & D-62.
- **Option B (Rejected):** Embed CEL or a scripting expression evaluator. Adds heavy dependencies and violates Gradus's minimal-dependency / static-binary philosophy.

## Simplicity And Elegance Review

- A single `Profile.Lookup(key)` + `MatchProfile(profile, when)` helper handles both standard and extensible keys in ~60 lines of pure Go with zero external dependencies.

## Failure Modes And Tradeoffs

- **Missing `profile` in `gradus.yaml`:** When `profile` is nil, `MatchProfile` returns `false`, so `activation: profile` overlays stay inactive (`reason: "profile mismatch"`) and `required: true` rules do not false-trigger.
- **Disabled mandatory overlay:** If a repo explicitly sets `activation: disabled` on an overlay required by an upper level's matching `activation_rule`, the overlay remains inactive and `resolve` emits a blocking `overlay` conflict so `verify` fails with exit `1`.

## Verification Plan

- Unit tests in `internal/schema/parse_config_test.go` (`TestProfileAndActivationRuleParsing`).
- Unit tests in `internal/resolve/overlays_test.go` (`TestConditionalOverlayActivation`).
- CLI & schema tests in `internal/cli/overlays_profile_test.go` (`TestConditionalOverlaysCLI`) and `go test ./spec/v1`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/schema/types.go`, `parse_config.go`, `validate_config.go` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/resolve/overlays.go`, `internal/resolve/resolve.go`, `internal/resolve/types.go` (`R2.AC1`, `R2.AC2`) |
| `R3` | `spec/v1/schema/*.json`, `internal/cli/overlays_profile_test.go` (`R3.AC1`) |
| `NFR1` | Static `CGO_ENABLED=0` build and `git` via `os/exec` |
| `NFR2` | Canonical English across all repository files |
| `C1` | Backward-compatible handling of `always`, `explicit`, `disabled`, and omitted `activation` |
| `C2` | Full Go test suite regression pass |
