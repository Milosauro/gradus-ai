# ADR 0001: Level Ordering and CI Overrides

- **Status:** Decided
- **Date:** 2026-11-19
- **Deciders:** Alessandro Enrico Milani, Core Architecture
- **Context:** `gradus-design.md` §6.3, §9.2, §10, §24.2

## Context and Problem Statement

In `gradus-design.md` §24.2, the minimal personal setup illustrates `user` listed before `project` so project rules act as the more specific level. However, §9.2 specifies that home-directory paths (`~/.gradus/...`) are excluded during CI runs to prevent environmental leakage, while §10 step 8 stipulates that an `overrides` targeting an unknown ID is an immediate error. If a committed repository or team rule attempts to override a user-level rule, local execution succeeds but CI resolution fails because the target rule does not exist in the CI view. Furthermore, transitioning from Phase 0 to Phase 1 (`team` -> `project` -> `user`) inverts the relative precedence if `user` were positioned above `project`.

## Decision Drivers

- Resolution must be deterministic and reproducible between developer workstations and CI runners.
- Committed repository levels must not depend on or specialize uncommitted personal workstations state.
- Pyramid definitions must scale smoothly from personal single-developer usage to multi-tier enterprise configurations without requiring an inversion of level semantics.

## Considered Options

1. **Allow `user` above `project` in personal setups, with silent ignore of missing override targets in CI.**
2. **Prohibit committed levels from overriding `user` rules; enforce `user` as the most specific level.**

## Decision Outcome

Chosen option: **Option 2**.

1. The `user` level is formally defined as the most specific level in pyramids that participate in CI or shared repositories. In local single-developer repositories where no team level exists, `project` represents repository-bound rules and `user` represents personal workstation preferences.
2. An override declared in a committed level (`org`, `team`, `project`) targeting a rule defined solely in the uncommitted `user` level (`~/.gradus/levels/user`) is structurally prohibited and rejected during `gradusctl verify`.
3. In CI environments where `~` paths are skipped, resolution succeeds without missing-target errors because committed levels only reference IDs within committed sources.

## Consequences

- **Positive:** CI runs never fail due to missing local user rules.
- **Positive:** Ordering remains stable (`org` -> `team` -> `project` -> `user`) across all phases.
- **Negative:** A developer cannot commit a project rule that purports to specialize an uncommitted personal habit (which is architecturally desirable).
