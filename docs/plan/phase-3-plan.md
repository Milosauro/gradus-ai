# Gradus — Phase 3 ("Organization") Implementation Plan

| Field | Value |
|---|---|
| Scope | Phase 3 (`v0.4.0`) |
| Status | In progress |
| Design Reference | [`docs/design/gradus-design.md`](../design/gradus-design.md) (Design 1.1, §20, §21, §24.1, §25) |

## Goal

Scale Gradus from business-unit governance (Phase 2) to organization-wide governance without breaking prior phases: profile-driven conditional overlay activation, compliance framework mapping and audit exports, model capability-tier policies, enterprise managed settings and chat skill bundles, an MCP server exposing the resolved pyramid, process maturity assessment, and optional commit-process attribution trailers.

## Feature Breakdown (Walden `F23`–`F28`)

| Feature | Walden Spec | Scope |
|---|---|---|
| **F23** | `conditional-overlays-by-profile` | Evaluate `activation_rules` in `level.yaml` and `overlay.yaml` against the repository `profile` (`data`, `exposure`, `domain`, `client`, extensible keys) during `resolve`, enforcing `allowed_overlays`, `strength` ordering, and clear provenance/diagnostics. |
| **F24** | `framework-mapping-and-audit-export` | Support `controls:` metadata on rules (`X-02`), framework coverage reporting, and structured audit export (`gradusctl audit export`) combining resolved rules, provenance, evidence records, and active exceptions. |
| **F25** | `model-policy-and-tiers` | Activate the `models:` configuration key (§21): capability tiers (`tier-high`, `tier-standard`, `tier-light`), operation risk classes, per-agent model mapping, compiler `model` emission where supported (e.g. Claude Code skills), `model_tier` evidence recording, and `verify` tier checks. |
| **F26** | `managed-settings-and-chat-bundles` | Enterprise managed settings generation for supported coding agents and portable ZIP skill bundle export for chat agents (§20). |
| **F27** | `mcp-server` | `gradusctl serve --mcp` (§20): local/stdio MCP server exposing `resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, and `get_toolchain` over the resolved pyramid. |
| **F28** | `maturity-assessment-and-commit-trailers` | `gradus-assess` skill for process maturity assessment (strictly processes, never individuals, `X-13`/P16) and optional git commit attribution trailers (`Gradus-Lock-Hash`, `Gradus-Agent`, `Gradus-Model-Tier`, `Gradus-Review-ID`, `X-05`/`D-65`). |

## Exit Verification

- Each feature (`F23`–`F28`) is specified, approved, implemented, and verified via Walden (`walden verify` + `walden validate --all --json`).
- Static `CGO_ENABLED=0` binary build (`PRJ-GO-001`), `go mod tidy -diff`, full `go test -race ./...`, and `gradusctl verify` / `check` / `test` pass on `gradus-ai`.
- `spec/v1/schema/*.json`, `spec/v1/SPEC.md`, `docs/design/gradus-design.md`, and `docs/compatibility-matrix.md` stay synchronized with every new `gradus/v1` capability.
