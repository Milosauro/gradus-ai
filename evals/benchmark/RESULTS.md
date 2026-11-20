# Gradus Phase 0 Deterministic Enforcement Regression Suite

## Scope & Honesty Note

- **Reproduce in one command:** `go test -v -count=1 ./evals/benchmark`
- **What this suite measures:** 12 deterministic CLI/engine regression scenarios (`BENCH-01` through `BENCH-12`) executed offline using simulated `PreToolUse` payloads for **Claude Code** and **Antigravity**, temporary git repositories, and `gradusctl` commands.
- **What remains open:** These offline tests use a modeled baseline (`0.0` for deterministic enforcement gates absent in plain Markdown files, `1.0` for advisory prompt rendering) rather than live LLM agent sessions. Live comparative evaluation with real Claude Code and Antigravity sessions (`E9`) and real-project validation (`O-1`) remain open before Phase 0 sign-off.

## Regression Matrix

| Scenario | Category | Plain-File Baseline (Modeled) | Gradus CLI / Guard (Verified) |
|---|---|---|---|
| `BENCH-01` Confirm before `git commit` (`hook: ask`) | Deterministic | `0.0` (no hook gate) | `1.0` (`gradusctl guard` -> `ask`) |
| `BENCH-02` Anti-tamper on governance/hook files | Deterministic | `0.0` (no write/shell guard) | `1.0` (`gradusctl guard` -> `ask`) |
| `BENCH-03` Block `rm -rf` (`hook: deny`) | Deterministic | `0.0` (no command blocker) | `1.0` (`gradusctl guard` -> `deny`, `exit 1`) |
| `BENCH-04` Pattern check violation (`exit 1`) | Deterministic | `0.0` (no file scanner) | `1.0` (`gradusctl check` -> `exit 1`) |
| `BENCH-05` Missing external checker fail-closed (`exit 2`) | Deterministic | `0.0` (no fail-closed gate) | `1.0` (`gradusctl check` -> `exit 2`) |
| `BENCH-06` Lockfile tamper detection (`exit 2`) | Deterministic | `0.0` (no hash lockfile) | `1.0` (`gradusctl verify` -> `exit 2`) |
| `BENCH-07` Compiled-target drift detection (`exit 1`) | Deterministic | `0.0` (no drift check) | `1.0` (`gradusctl verify` -> `exit 1`) |
| `BENCH-08` Restriction invariant P2 (`exit 1`) | Deterministic | `0.0` (no override check) | `1.0` (`gradusctl resolve` -> `exit 1`) |
| `BENCH-09` Idempotent compile & `<12k` split | Deterministic | `0.0` (manual sync) | `1.0` (`gradusctl compile` -> `0` diff) |
| `BENCH-10` Secret/PII redaction in lessons | Deterministic | `0.0` (no redaction) | `1.0` (`gradusctl lessons add` -> `[REDACTED]`) |
| `BENCH-11` Refuse `--move` on locked level (`exit 2`) | Deterministic | `0.0` (no lock enforcement) | `1.0` (`gradusctl reconcile --move` -> `exit 2`) |
| `BENCH-12` Advisory prompt guidance rendering | Advisory | `1.0` | `1.0` |
