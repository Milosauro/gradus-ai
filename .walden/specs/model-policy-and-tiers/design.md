---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T08:35:09Z
last_modified: 2026-11-25T08:35:09Z
approved_fingerprint: sha256:9d9c350cc5b07c82636c96c2dfb62e4550ffce2513511683a27a8005da33a210
source_requirements_approved_at: 2026-11-25T08:35:09Z
source_requirements_fingerprint: sha256:1bd160438b9aac6f1198d240ee833d359d33825066d7c0193785cc8d608ee089
---

# Design: Model Policy & Capability Tiers (`F25`)

## Architecture

1. **Schema Layer (`internal/schema`):**
   - Add `ModelPolicy` struct to `internal/schema/types.go`:
     - `Tiers map[string]map[string]string` (`tier-high|tier-standard|tier-light` → `agent` → concrete model name)
     - `Operations map[string]string` (operation name e.g. `review`, `author`, `conflicts`, `learn`, `onboard`, `reconcile`, `link`, `assess` → required tier)
   - Add `Models *ModelPolicy` to `PyramidConfig` and `LevelConfig`.
   - Remove `"models"` from `ReservedTopLevelKeys` in `internal/schema/unknown.go` (leaving `"adapters"` reserved for Phase 3), add `"models"` to `pyramidFieldSpec` and `levelFieldSpec` in `parse_config.go`, and validate tier names (`tier-high`, `tier-standard`, `tier-light`) in `validate_config.go`.
2. **Resolution Layer (`internal/resolve/models.go`, `resolve.go`, `types.go`):**
   - Define tier ranking: `tier-light` (1) < `tier-standard` (2) < `tier-high` (3).
   - `ResolveModelPolicy(steps []PyramidStep, projectModels *schema.ModelPolicy) (*ResolvedModelPolicy, []Conflict)`:
     - Merges `tiers` (agent model mappings) and `operations` across levels in pyramid order (`org` → `team` → `project` → `user`).
     - If a lower level sets an operation's tier strictly lower than an upper level's tier, emits a blocking `Conflict` (`Kind: "model_policy"`) and retains the stricter tier.
   - Expose `Models *ResolvedModelPolicy` (`json:"models,omitempty"`) on `ResolvedView`, plus helper `TierRank(tier string) int` and `ModelForSkill(target, skillName string) string`.
3. **Claude Code Compiler (`internal/compile/claudecode.go`):**
   - After `PlanEmbeddedSkills`, if `view.Models != nil`, inject `model: <concrete-model>` into the YAML frontmatter of each `.claude/skills/gradus-<op>/SKILL.md` whose operation resolves to a `claude-code` model.
4. **Evidence & Verify CLI (`internal/cli/evidence.go`, `internal/cli/verify.go`):**
   - Add `--skill` and `--model-tier` flags to `gradusctl evidence write`, validating `--model-tier` if non-empty and setting `rec.Producer.Skill` / `rec.Producer.ModelTier`.
   - Add `--check-model-tier` flag to `gradusctl verify`: loads local `.gradus/evidence/*.json`, checks the latest record for each governed operation (`review` ↔ `kind == "review"`, `conflicts` ↔ `kind == "semantic-conflicts"`, or matching `producer.skill`), and fails with exit `1` if its `model_tier` is missing or below the required tier.
5. **JSON Schemas (`spec/v1/schema/{gradus,level,resolved-view}.schema.json`):**
   - Add `models` object schema to all three files.

## Options Considered

- **Option A (Chosen):** Opt-in `--check-model-tier` flag on `gradusctl verify` (plus automatic monotonic conflict checking in `resolve`/`verify` whenever `models:` is declared). Keeps existing repos and CI green until teams enable evidence-tier enforcement.
- **Option B (Rejected):** Always fail `verify` if no local evidence exists for every operation in `models.operations`. Would break `verify` on fresh checkouts where no review evidence has been generated yet.

## Simplicity And Elegance Review

- Reuses `evidence.Producer.Skill` and `evidence.Producer.ModelTier` (already present in `internal/evidence/sink.go` and `spec/v1/schema/evidence.schema.json` since Phase 0!).

## Failure Modes And Tradeoffs

- **Unverifiable runtime model:** Per Design §21, where an agent harness cannot cryptographically attest the model used, `model_tier` in evidence is self-reported and documented as a process check complemented by `claude-code` compiled `model:` frontmatter.

## Verification Plan

- Unit tests in `internal/resolve/models_test.go` (`TestModelPolicySchemaAndResolution`).
- Compiler tests in `internal/compile/claudecode_test.go` (`TestClaudeCodeSkillModelInjection`).
- CLI tests in `internal/cli/models_test.go` (`TestModelPolicyEvidenceAndVerifyCLI`) and `go test ./spec/v1`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `internal/schema/{types,unknown,parse_config,validate_config}.go`, `internal/resolve/models.go` (`R1.AC1`, `R1.AC2`) |
| `R2` | `internal/compile/claudecode.go` (`R2.AC1`) |
| `R3` | `internal/cli/evidence.go`, `internal/cli/verify.go` (`R3.AC1`, `R3.AC2`) |
| `NFR1` | Static `CGO_ENABLED=0` binary build and `git` via `os/exec` |
| `NFR2` | Canonical English across all repository files |
| `C1` | Backward-compatible behavior when `models:` is absent |
| `C2` | Full Go test suite regression pass |
