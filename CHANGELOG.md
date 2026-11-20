# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-11-20

### Added
- **Core Schema & CLI (`F1`):** `spec/v1` JSON Schemas, forward-compatible YAML/frontmatter parser with ADR 0004 container-based level derivation, `gradusctl init`, `gradusctl explain`, and `gradusctl version [--json]`.
- **Pyramid Resolution & Lockfile (`F2`):** Local and `os/exec` Git source resolution, offline cache policies, normalized cross-platform SHA-256 `gradus.lock`, P2 restriction invariant (with ADR 0001), `gradusctl resolve`, and `gradusctl status [--brief]`.
- **Agent Compilers (`F3`):** Deterministic `gradusctl compile` generating `AGENTS.md`, `CLAUDE.md`, `.claude/rules/`, `.claude/settings.json`, `.agents/rules/` (`<11,500`-char chunking), `.agents/hooks.json`, and embedded skill extraction.
- **Verification, Checks, Evidence & Git Hooks (`F4`):** `gradusctl verify`, `gradusctl check [--staged | --base]`, 5 evidence sinks (`none`, `local`, `ci-artifact`, `repo`, `command`), and `gradusctl hooks` (`pre-commit`, `lefthook`, `husky`, `native`).
- **Runtime Guard (`F5`):** `<100ms` offline `PreToolUse` hook evaluator (`gradusctl guard`) for Claude Code and Antigravity with built-in anti-tamper protection (Threat Model T2) and `deny > ask > allow` precedence.
- **Lifecycle State (`F6`):** `gradusctl conflicts`, `gradusctl lessons` with automatic secret/PII redaction (Threat Model T4, P16), and `gradusctl reconcile` (`--restore`, `--move`).
- **Canonical MVP Skills & Evals (`F7`, `F8`):** Six embedded Agent Skills (`gradus-onboard`, `gradus-author`, `gradus-review`, `gradus-learn`, `gradus-reconcile`, `gradus-conflicts`), `evals/skills/` Skill-TDD suite, and `evals/benchmark/` comparative evaluation (`E1`–`E10`).
- **Supply-Chain Hardening (`T3`):** `.goreleaser.yaml` (`CGO_ENABLED=0` static binaries, `checksums.txt`, `syft` SBOM, keyless `cosign` signing) and SHA-pinned `.github/workflows/release.yml` with build provenance attestations.
