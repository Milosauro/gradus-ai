# Gradus — Design Document

| Field | Value |
|---|---|
| Project | **Gradus** — layered, governed rules for AI coding agents |
| Document version | 1.2 |
| Date | 25 November 2026 |
| Status | Living design specification. See `README.md` and `CHANGELOG.md` for current release and implementation status. |
| Repository | https://github.com/Milosauro/gradus-ai |
| License | Apache-2.0 |
| Spec version described | `gradus/v1` |
| Language | English (canonical). A personal Italian translation exists and is non-normative. |

## How to read this document

### Normative precedence

When specification sources appear to overlap, precedence is resolved in this order (D-75):

1. **`spec/v1/schema/*.json`** — normative for file formats, field names, and structural validation.
2. **`docs/design/gradus-design.md` and accepted ADRs (`docs/adr/`)** — normative for semantics (level precedence, invariants, enforcement behavior, exit codes, and governance workflows).
3. **`spec/v1/SPEC.md`** — descriptive guide summarizing the schemas and conventions.

Every section carries one of these labels:

| Label | Meaning |
|---|---|
| **Decided** | Agreed and stable. Changes require a new entry in the decision log. |
| **Specified** | Concrete format or behavior, precise enough to implement. May still be refined during Phase 0. |
| **Proposed** | Recommended direction awaiting confirmation. |
| **Deferred** | Designed at a high level and intentionally scheduled after the MVP. |
| **Open** | Not yet decided. Listed in §27. |

"MVP" means the scope of Phase 0 (§24). "The kit" and "Gradus" are used interchangeably.

---

## 1. Problem **[Decided]**

Teams adopting AI coding agents (Claude Code, Antigravity, Codex, GitHub Copilot, OpenCode, Cursor and others) quickly end up with every developer running a personal setup: instruction files, ad-hoc rules, skills installed at will. The same repository is changed by agents following different and often contradictory rules, and company policies on data, security, certification and conventions are easy to miss.

Existing tools cover parts of the problem. None combines, in one coherent product:

- a hierarchy of rules whose levels are **defined by the organization**, not fixed by a tool;
- **governance** of those levels: who owns them, who approves changes, where they come from, whether they are locked;
- **conflict handling** with a clear invariant;
- **traceability** of each rule to its origin and **evidence** of what was checked;
- a **learning loop** that turns repeated corrections into rules at the right level, with human approval;
- **tool independence** and coexistence with the workflows teams already use.

The problem is also organizational. Gradus makes divergence visible and enforces decisions once they are made; it does not replace the discussion between people about which conventions to adopt. It should make that discussion easier.

## 2. Goals and non-goals **[Decided]**

**Goals**

1. Let a single developer layer rules for personal projects with minimal setup.
2. Let the same setup grow to team, business unit and organization without changing tools.
3. Enforce deterministically whatever can be checked mechanically, and guide the agent reliably for everything else.
4. Produce verifiable evidence of compliance.
5. Work with as many coding agents as possible.

**Non-goals**

| Not a goal | Why |
|---|---|
| A spec-driven development framework | Teams keep their SDD tool (Walden, spec-kit, others); Gradus binds to it (§19). |
| A rule-file synchronizer | Synchronizing formats is a means, not the product. |
| A runtime gateway for agent actions in production systems | Different problem space. |
| A certification tool | Gradus helps produce evidence; it does not certify compliance. |
| A tool to monitor or evaluate people | Explicitly excluded (P16). |
| A universal organizational template | Every organization has its own structure (P15). |

## 3. Positioning **[Decided]**

### 3.1 What is common and what is distinctive

Layered rules compiled into agent instruction files already exist in simple forms (§3.2). Gradus's value is therefore **the governance around the levels**, not the levels themselves:

- configurable levels with lock state, versioned remote sources, owners, approvers and charters;
- a pyramid definition that is itself governed and layered (§7);
- overlays with conditional applicability;
- abstract obligations bound to different tools per team;
- a learning loop with promotion to the right level, and drift reconciliation;
- verifiable evidence and progressive adoption from one developer to a whole company.

### 3.2 Prior art (analysis of November 2026)

No project overlaps with the combination above. Gradus absorbs ideas, not dependencies.

| Project | Nature | Overlap | Ideas absorbed |
|---|---|---|---|
| Claude Code memory and managed settings | Native hierarchy (managed policy, project, user), hooks, permissions | Native mechanisms | Used as compilation and enforcement targets |
| Cursor rules | Team, project and user rules; enforced team rules | Native mechanisms | Top-down precedence is closer to Gradus semantics |
| stele | Repository rule engine: rules shown before writes, checked after, verified in CI; lockfile; checker plugins; published eval | High on enforcement engine; low on governance | Write-time enforcement; no silent disabling; missing checker = error, never "no violations"; unknown keys as notes; eval against a CLAUDE.md baseline; checker plugin protocol |
| agentlaw | Law-first kit per project; memory and session restore via MCP | Medium (hierarchy by artifact type) | Lesson write discipline; promotion proposals judged by the agent and approved by a human. Counter-example: avoid heavy session start-up procedures |
| org-agentic-toolkit, daimyo, UCP | Organization/project/user rules compiled or served with inheritance | High on the basic pyramid | Multiple inheritance for technology scopes (→ technology overlays); MUST rules accumulate |
| Hivelore | Lessons turned into deterministic guards for git hooks and CI | Medium | Repeated lesson → deterministic guard when possible |
| Qodo (commercial) | Consolidated rule list across agents; inactive rules deactivated | Medium | Flag rules that are never triggered (optional, no data about people) |
| spec-kit | SDD with a constitution and an analyze step | Low (provider) | Constitution as governing document; lesson from a bug where the implement step did not load the constitution → policy must be loaded wherever output is produced |
| Walden | SDD delivery kernel: deterministic CLI plus skills, gated workflow | Low (provider) | Skill for judgment, CLI for enforcement; CLI owns state; fingerprints; humans approve; blocking decision markers; constitution vs lessons; proportional lanes |
| Superpowers (TDD, writing-skills) | Skills with explicit rationalizations; TDD applied to skills | Low | Rules state the excuses they reject; skills are developed against test scenarios |
| claude-md-management, claude-reflect, hookify, code-review | Rule maintenance, correction capture, frustration-to-hook, compliance review | Medium on single pieces | Diff-based proposals; multilingual correction detection; deterministic hooks from repeated frustration; redundant compliance reviewers with confidence threshold |

## 4. Design principles **[Decided]**

| # | Principle |
|---|---|
| P1 | **Judgment to the model, enforcement to determinism.** Whatever can be checked mechanically is checked by the CLI, hooks, permissions or CI, not left to the prompt. |
| P2 | **Restriction invariant.** A lower level may only restrict or specialize the levels above it, never relax them, unless a rule is explicitly declared overridable. |
| P3 | **Policy is loaded wherever output is produced**, not only where output is checked. Review is a safety net, not the primary mechanism. |
| P4 | **The agent proposes; the competent human approves.** No self-approval, not even when asked in chat. |
| P5 | **Single source of truth.** Agent instruction files are generated outputs; direct edits are detected as drift. |
| P6 | **Workflow rules follow the code, not the person.** In a repository or path, the owner's workflow applies; the user level follows the person only for preferences that do not touch obligations. |
| P7 | **Traceability.** Every rule has an ID, a level, an owner and a source; every finding cites them. |
| P8 | **Proportionality.** The amount of ceremony scales with the impact of a change. |
| P9 | **Those who organize the work decide.** Gradus provides mechanisms, not a mandatory structure. |
| P10 | **Tool and workflow independence.** Abstract obligations, concrete bindings and adapters. |
| P11 | **Minimal context.** The agent loads the resolved view for its scope, never the raw pyramid. |
| P12 | **Progressive adoption.** Fully useful to one developer; grows to team, business unit and organization without changing tools. |
| P13 | **Minimal mandatory core; everything else configurable.** Only levels, rules, resolution and compilation are required. |
| P14 | **Do not reinvent; reuse.** Integrate established linters, scanners, policy engines, hook managers, dependency updaters and CI. |
| P15 | **Templates, not prescriptions.** Examples are starting points to adapt. |
| P16 | **No people monitoring.** Gradus governs rules and code; it does not measure the activity or performance of individuals. |

## 5. Concepts **[Specified]**

| Term | Definition |
|---|---|
| **Pyramid** | The ordered set of levels that applies to a scope, from most general to most specific. |
| **Level** | One step of the pyramid (e.g. organization, team, project, user). Has a source, owners, approvers, a charter and a lock state. |
| **Charter** | Statement of what belongs to a level and what does not, with examples. |
| **Rule** | A single instruction with metadata (ID, severity, enforcement, scope, source). One Markdown file per rule. |
| **Severity** | `MUST`, `SHOULD` or `MAY`. |
| **Enforcement** | How a rule is applied: `prompt`, `review`, `check`, `hook`, `permission`. |
| **Overlay** | A cross-cutting rule set (regulatory, client, technology) activated for some scopes, with a declared strength. |
| **Applicability profile** | Facts a project declares about itself (e.g. data handled), used to activate overlays. |
| **Obligation** | An abstract requirement (e.g. "an approved spec before code") declared by a level. |
| **Binding** | The concrete way a lower level satisfies an obligation (e.g. with Walden). |
| **Adapter** | Component connecting Gradus to an agent tool or workflow tool. |
| **Toolchain** | Skills, tools, MCP servers or plugins declared required, recommended or forbidden by a level. |
| **Lesson** | A recorded observation (correction, repeated mistake, stated preference) with evidence. Not a rule. |
| **Promotion** | Turning a lesson into a rule at a chosen level, through approval. |
| **Resolved view** | The merged, conflict-checked set of rules for a scope. The only thing agents load. |
| **Lockfile** | `gradus.lock`: resolved versions and hashes of every level and overlay. |
| **Evidence** | Machine-readable record of a check or review, tied to commit and lock. |
| **Exception** | A time-limited, approved waiver of a rule for a scope. |
| **Drift** | Any divergence between what was resolved and approved and what is on disk. |

## 6. The pyramid model

### 6.1 Levels **[Decided]**

The number and names of levels are defined in configuration. The default template has four levels; organizations adapt it.

| Default level | Typical content |
|---|---|
| `org` | Data handling, certifications, security, mandatory practices, allowed tools and vendors |
| `team` | Conventions, naming, adopted workflows |
| `project` | Repository- or module-specific rules |
| `user` | Personal preferences and habits |

Deep pyramids (more than six levels) are allowed, but `gradusctl doctor` flags them: they usually indicate an organizational issue rather than a technical one.

### 6.2 Level properties **[Specified]**

Locking is a property of **every** level, not only the top one. A common regulated setup locks every level except `user`.

| Property | Meaning |
|---|---|
| `id` | Level identifier. Order in the list is the order in the pyramid. |
| `source` | Where the level comes from: local path or git repository (MVP); HTTPS URL or API (later). |
| `locked` | If true, the level cannot be edited in the working repository; changes go through its source and its review process. Locked does not mean immutable. |
| `editable_rules` | Rule IDs of a locked level that may be specialized locally. |
| `owners`, `approvers` | Who owns the level and who approves changes. |
| `offline` | Behavior when the source is unreachable: `cache` (with `max_age`) or `fail-closed`. |
| `min_version` | Minimum version a consumer may pin (remote levels). |
| `signature` | `required` or `optional` verification of release signatures (later). |
| `charter` | See §6.6. |
| `scope` | Paths or repositories where the level applies. |
| `exceptions` | Exception policy for the level (§13). |

### 6.3 Precedence and the restriction invariant **[Decided]**

- Rules of all applicable levels are merged.
- A lower-level rule may **specialize** a higher one by declaring `overrides: <ID>`. A specialization must not lower severity, widen scope, or weaken enforcement, unless the higher rule declares `overridable: true` (optionally with constraints).
- Any other contradiction is a **conflict**. Conflicts are never resolved silently: they become blocking markers until an owner with authority decides (§10).
- This semantics belongs to Gradus and is independent of each tool's native precedence (Claude Code, for example, lets more specific files win except for managed policy). Compiled outputs state the precedence explicitly, and deterministic rules are enforced through hooks and checks.
- The `user` level, even when it is the only editable one, is subject to the invariant.

### 6.4 Overlays **[Decided; conditional activation Specified]**

Some rules are cross-cutting: data protection where personal data is processed, payment security for payment flows, a client's rules for that client's projects, conventions for a technology stack. Forcing them into a level causes duplication. Gradus therefore models **pyramid + overlays**.

- Each overlay has an owner, a source, a version and a **strength**: the level it acts as (`strength: org` means it carries the authority of the `org` level). In the resolution order it is inserted right after that level.
- Between overlays that conflict, the more restrictive rule wins when the conflict is structural (e.g. `MUST` vs `SHOULD` on the same overridden ID); otherwise the conflict is blocking.
- Technology overlays (e.g. `stack-python-fastapi`) are ordinary overlays.

**Conditional activation.** Overlays need not be imposed everywhere: a small service that handles no personal data should not receive data-protection rules.

1. A project declares an **applicability profile** with a minimal vocabulary: `data` (`none`, `internal`, `personal`, `sensitive`), `exposure` (`internal`, `public`), `domain` (free list, e.g. `payments`), `client` (free list). Organizations may extend the vocabulary.
2. A higher level declares **activation rules**: which overlays are mandatory for which profile values.
3. Individual overlay rules may carry their own `applies_when` conditions.

The profile is itself governed: it is approved by the project owners, and skills flag inconsistencies (e.g. `data: none` declared while the code handles email addresses). Flexibility must not become a loophole.

### 6.5 Scopes and "rules follow the code" **[Decided]**

Rules with `scope` globs load only where relevant. Workflow obligations are attached to the repository or path, not to the person: a developer from a team using spec-kit who contributes to a repository bound to Walden follows Walden there. In monorepos, bindings are path-scoped.

### 6.6 Charters **[Decided; format Specified]**

Every level has a charter that says what belongs there and what does not. Without it, rules drift to the wrong level (naming conventions in the company policy, regulatory requirements in personal preferences). Charters are used by `gradus-author` to guide writing and redirect content, by `gradus-onboard` to classify discovered rules, and by `gradus-review` to flag misplaced rules.

```yaml
charter:
  purpose: "Constraints that apply to the whole organization: law, contracts, certifications, security."
  includes:
    - "Data handling and classification"
    - "Certification requirements"
    - "Allowed and forbidden tools, models and vendors"
  excludes:
    - "Naming conventions -> team"
    - "Technical choices of one repository -> project"
    - "Personal style preferences -> user"
  examples_in: ["No real personal data in test fixtures"]
  examples_out: ["Use camelCase for variables"]
  allowed_severities: [MUST, SHOULD]
  review_cadence: "6 months"
```

Gradus ships example charters for the default levels; organizations adapt them.

---

## 7. Governance of the pyramid definition **[Decided; details Specified]**

The pyramid definition is critical code: if it is removed or edited by mistake, every guarantee disappears. Where it lives and how rigid it is remains the choice of whoever organizes the work (P9).

### 7.1 Modes

| Mode | Where the definition lives | Protection | Fits |
|---|---|---|---|
| A. Dedicated locked repository | A governance repository referenced and pinned by the lockfile | Source repository review and permissions; hashes; signatures (later) | Regulated organizations, many teams |
| B. Versioned file in the project | `gradus.yaml` in the working repository | CODEOWNERS, branch protection, mandatory CI check, agent edit block | Personal projects, single teams |
| C. Hybrid | Base structure in a dedicated repository, local extensions in the project | A and B combined, with constraints declared from above | Most mid-size organizations |

### 7.2 Layered structure with inherited constraints

The structure of the pyramid is itself layered: each owner shapes their subtree within constraints set by the level above. Examples of constraints:

- "the `team` level must exist and be locked";
- "projects may add at most one intermediate level";
- "only these overlays are allowed";
- "the project level may be a local file, but CODEOWNERS is mandatory".

`gradusctl verify` checks every local definition against inherited constraints. A violation makes the pyramid invalid.

### 7.3 Protection against accidental changes

1. `verify` runs in git hooks and CI and fails if the definition changes without required approvals or breaks inherited constraints.
2. Hashes in `gradus.lock` make every change visible in review.
3. Agent hooks require confirmation before an agent edits `gradus.yaml`, `gradus.lock`, `.gradus/` or the hook configuration that runs Gradus (no silent disabling).
4. `gradusctl doctor` explains in plain language what is broken and how to fix it.

## 8. Sources, versions and integrity **[Decided; MVP subset Specified]**

The model is that of a package manager.

| Aspect | Design | MVP |
|---|---|---|
| Source types | Local path; git (tag or commit); HTTPS URL; internal API | Local path and git |
| Credentials | Never in configuration files; existing git credentials, environment variables or vault | Yes |
| Lockfile | `gradus.lock` with resolved version and hash for every level and overlay | Yes |
| Integrity | A locked level whose local copy differs from the lock hash fails verification | Yes |
| Signatures | Signed git tags or detached signatures (e.g. cosign, minisign), per level | Later |
| Minimum version | `min_version` set by the level owner | Later |
| Offline | Per level: `cache` with `max_age`, or `fail-closed` | `cache` and `fail-closed` |
| Performance | One resolution pass with cache; no fetch per session when the lock is valid | Yes |
| Evidence binding | Every evidence record carries the lock hash, so evidence produced under an older pyramid is recognizable as stale | Yes |

**Approvals for remote levels.** In the MVP and Phase 1, changes to a remote level are approved through pull or merge requests on the source repository. An approval flow through an API remains in the design and is deferred (§25).

**Managed settings.** For rules that must be impossible to bypass, the CLI can generate fragments of managed settings or equivalent policy files for IT to distribute (deferred).

---

## 9. Specification `gradus/v1` (draft) **[Specified]**

This section defines the formats implemented in Phase 0. Fields marked *(later)* are reserved now so that later phases do not break compatibility.

### 9.1 File layout

```
<repository>/
├── gradus.yaml                 # pyramid definition (committed)
├── gradus.lock                 # resolved versions and hashes (committed)
├── .gradus/
│   ├── levels/
│   │   └── project/            # one directory per local level
│   │       ├── level.yaml      # level metadata and charter
│   │       └── rules/          # one Markdown file per rule
│   │           └── PRJ-NAM-001.md
│   ├── exceptions/             # approved exceptions (committed)
│   ├── findings/               # dismissed or confirmed semantic findings (committed)
│   ├── cache/                  # fetched remote levels (git-ignored)
│   └── evidence/               # local evidence sink (git-ignored by default)
├── AGENTS.md                   # generated
├── CLAUDE.md                   # generated (or a managed section within it)
├── .claude/rules/gradus-*.md   # generated
├── .agents/rules/gradus-*.md   # generated (Antigravity)
└── .agents/hooks.json          # generated or merged (Antigravity)

~/.gradus/
├── config.yaml                 # user settings
├── levels/user/                # the user level (level.yaml + rules/)
└── lessons/<project-key>/      # lessons, local and never committed automatically
```

Generated files for project-wide content are committed, so agents that run without a local Gradus installation (for example background or CI agents) still receive the instructions. Generated files that contain user-level content are git-ignored.

### 9.2 `gradus.yaml`

```yaml
schema: gradus/v1

extends: null                   # or a git reference to a base definition (mode C)

levels:
  - id: team
    source:
      git: git@github.com:acme/ai-rules-team.git
      ref: v1.2.0
    locked: true
    offline: { mode: cache, max_age: 7d }
    approvers: [team-leads]
    editable_rules: [TEAM-NAM-012]

  - id: project
    source: { path: .gradus/levels/project }
    approvers: [repo-codeowners]

  - id: user
    source: { path: ~/.gradus/levels/user }
    approvers: [self]

overlays:                       # optional
  - id: stack-python-fastapi
    source: { path: .gradus/overlays/stack-python-fastapi }
    strength: team
    activation: always          # or: profile (see activation_rules in a higher level)

profile:                        # optional applicability profile
  data: internal
  exposure: internal
  domain: []
  client: []

targets: [claude-code, antigravity, agents-md]

hooks:
  manager: auto                 # auto | pre-commit | lefthook | husky | native | none

evidence:
  sinks:
    - type: local
  # see §12 for all sink types
```

Validation rules:

- Level IDs are unique and match `^[a-z][a-z0-9-]*$`.
- The order of `levels` is the order of the pyramid, from most general to most specific.
- Unknown keys produce a note, not an error, so that a newer configuration does not break an older CLI.
- Paths starting with `~` refer to the user's home directory and are never resolved inside CI unless explicitly configured.

### 9.3 `level.yaml`

Every level directory, local or remote, contains a `level.yaml`.

```yaml
schema: gradus/v1
id: team
version: 1.2.0                  # remote levels: matches the git tag
owners: [platform-team]
approvers: [team-leads]
charter: { ... }                # see §6.6
constraints:                    # constraints imposed on levels below (optional)
  require_levels: [project]
  max_extra_levels: 1
  allowed_overlays: [stack-*, gdpr]
activation_rules: []            # overlay activation by profile (optional)
exceptions:
  max_duration: 30d
  approvers: [team-leads]
  break_glass: true
```

### 9.4 Rule files

One Markdown file per rule, named after the rule ID, with YAML frontmatter.

~~~markdown
---
id: ORG-SEC-004
title: No real personal data in tests
severity: MUST
enforcement: check
scope: ["**/*"]
source: "Internal data policy v3, section 4.2"
checker:
  name: gitleaks
  config: .gitleaks.toml
tags: [data, testing]
version: 3
---
Test fixtures, logs and examples must never contain real personal data.

## Why
Repositories are copied, cached and shared far more widely than production databases.

## Rationalizations
- "It is only a local test" → the rule applies to local tests.
- "The data is already public" → classification does not depend on availability.
- "I will anonymize it later" → real data must never enter the repository.

## Examples
### Non-compliant
```json
{ "email": "mario.rossi@realcompany.example" }
```
### Compliant
```json
{ "email": "user-001@example.test" }
```
~~~

Frontmatter fields:

| Field | Required | Values and meaning |
|---|---|---|
| `id` | yes | Unique opaque identifier in the resolved view; pattern `^[A-Z][A-Z0-9]*(-[A-Z0-9]+)+$`. `<LEVEL>-<AREA>-<NNN>` is a naming convention only — a rule's effective `level` is strictly derived from its containing level container (ADR 0004). |
| `title` | yes | Short human-readable title. |
| `severity` | yes | `MUST`, `SHOULD`, `MAY`. |
| `enforcement` | no (default `prompt`) | `prompt` (instruction only), `review` (checked by `gradus-review`), `check` (deterministic check by the CLI or an external checker, in git hooks and CI), `hook` (blocked or confirmed in the agent session), `permission` (compiled into agent permissions). |
| `scope` | no (default `["**/*"]`) | Path globs. |
| `overrides` | no | ID of a higher-level rule that this rule specializes. |
| `overridable` | no (default `false`) | Whether lower levels may relax this rule; may be an object with constraints. |
| `applies_when` | no | Conditions on the applicability profile. |
| `source` | recommended | Origin: policy, regulation, decision record. |
| `controls` | no | External control IDs (framework mapping, optional). |
| `owner` | no | Defaults to the level owner. |
| `status` | no (default `active`) | `draft`, `active`, `deprecated`, `retired`. Lifecycle is optional. |
| `review_by` | no | Date for the next review (optional lifecycle). |
| `checker` | for `check` | External engine name and configuration. |
| `guard` | for `hook` | Deterministic match for agent actions, e.g. `{ action: command, pattern: "^git commit", decision: ask }`. Decisions: `deny`, `ask`. |
| `tags` | no | Free labels; also used to find semantic conflict candidates. |
| `version` | no | Integer incremented on change. |

The level is derived from the file location and never written in the rule. Body sections: the first paragraph is the rule statement (mandatory); `## Why`, `## Rationalizations` and `## Examples` are optional. Examples under `### Compliant` and `### Non-compliant` are used as rule tests (§23).

### 9.5 Overlays

An overlay directory has the same structure as a level (`overlay.yaml` plus `rules/`), with `strength` and optional `activation` conditions in `overlay.yaml`.

### 9.6 `gradus.lock`

```yaml
schema: gradus/v1
definition_hash: sha256:...      # hash of gradus.yaml and inherited definitions
entries:
  - kind: level
    id: team
    source: git@github.com:acme/ai-rules-team.git
    ref: v1.2.0
    commit: 4f2a9c1...
    hash: sha256:...             # hash of the normalized level content
    signature: none              # none | verified (later)
  - kind: level
    id: project
    source: .gradus/levels/project
    hash: sha256:...
view_hash: sha256:...            # hash of the resolved project view
```

The `user` level is never written to the lockfile; it is resolved locally.

### 9.7 Resolved view

`gradusctl resolve --json` emits the resolved view; compilation renders it for each target.

```json
{
  "schema": "gradus/v1",
  "view_hash": "sha256:...",
  "lock_hash": "sha256:...",
  "scope": { "repository": ".", "path": null, "includes_user_level": true },
  "levels": [ { "id": "team", "version": "1.2.0", "hash": "sha256:..." } ],
  "overlays": [ { "id": "stack-python-fastapi", "strength": "team", "active": true } ],
  "profile": { "data": "internal" },
  "rules": [
    {
      "id": "TEAM-SEC-004",
      "title": "No real personal data in tests",
      "level": "team",
      "severity": "MUST",
      "enforcement": "check",
      "scope": ["**/*"],
      "text": "Test fixtures, logs and examples must never contain real personal data.",
      "rationalizations": ["..."],
      "overrides": null,
      "provenance": { "source": "team@v1.2.0", "path": "rules/TEAM-SEC-004.md", "hash": "sha256:..." }
    }
  ],
  "conflicts": [],
  "exceptions": [],
  "budget": { "characters": 8450, "targets": { "antigravity": { "limit": 12000 } } }
}
```

`view_hash` is computed deterministically over shared/project levels only, excluding the personal `user` level and `lock_hash` so `gradus.lock` and compiled project markers remain invariant across developers (D-80).

### 9.8 Lesson record

Stored as JSON under `~/.gradus/lessons/<project-key>/` (normative schema: `spec/v1/schema/lesson.schema.json`).

```json
{
  "schema": "gradus/v1",
  "id": "L-2026-11-19-001",
  "created": "2026-11-19T10:12:00Z",
  "kind": "correction",
  "statement": "Do not commit without an explicit request.",
  "evidence": [
    { "session": "local", "excerpt": "please, don't commit on your own again", "at": "2026-11-19T10:11:40Z" }
  ],
  "occurrences": 2,
  "suggested_level": "user",
  "suggested_enforcement": "hook",
  "status": "proposed"
}
```

Excerpts are redacted (§22) and kept local.

### 9.9 Evidence record

```json
{
  "schema": "gradus/v1",
  "id": "ev-20261119-101500-3f9a",
  "kind": "check",
  "timestamp": "2026-11-19T10:15:00Z",
  "commit": "a1b2c3d",
  "base": "origin/main",
  "lock_hash": "sha256:...",
  "view_hash": "sha256:...",
  "producer": { "tool": "gradusctl", "version": "0.1.0", "skill": null, "model_tier": null },
  "findings": [
    { "rule_id": "ORG-SEC-004", "severity": "MUST", "path": "tests/fixtures/users.json", "line": 3,
      "message": "Real e-mail address in fixture", "confidence": 1.0, "status": "open" }
  ],
  "verdict": "fail"
}
```

`kind` is one of `verify`, `check`, `review`, `semantic-conflicts`, `exception-use`. `verdict` is `pass`, `fail` or `error`. Records contain no information about the activity of individuals beyond what is already in git history.

### 9.10 Exception record

```yaml
schema: gradus/v1
id: EX-2026-014
rule_id: TEAM-TST-003
scope: ["services/legacy/**"]
reason: "Legacy module without test harness; migration planned in Q1."
approved_by: [team-leads]
created: 2026-11-19
expires: 2026-12-19
kind: exception                 # exception | break-glass
post_review_due: null           # mandatory date for break-glass
```

### 9.11 Active Phase 2 and reserved Phase 3 fields

`obligations`, `bindings`, and `toolchain` are active in `gradus/v1` (§19). `models` and `adapters` remain reserved top-level keys for Phase 3 (§20–§21) and emit informational notes when encountered.

---

## 10. Resolution algorithm **[Specified]**

`gradusctl resolve` performs these steps:

1. **Load the definition.** Read `gradus.yaml` and any `extends`; validate the schema; collect inherited constraints.
2. **Resolve sources.** For each level and overlay: use the cache if it matches the lock; otherwise fetch, hash and compare. Apply `offline` policy when unreachable. A mismatch on a locked level is an error.
3. **Check constraints.** Verify the local definition against inherited constraints (§7.2).
4. **Determine applicability.** Read the profile; evaluate activation rules from higher levels; mark overlays active or inactive.
5. **Order.** Build the sequence: levels in declared order, each active overlay inserted right after the level named by its `strength`.
6. **Load rules.** Parse rule files; validate frontmatter; drop `retired` rules; keep `draft` rules out of the view unless requested.
7. **Apply exceptions.** Active, unexpired exceptions mark rules as waived for their scope. Expired exceptions are reported.
8. **Merge and detect structural conflicts.** Union of all rules, then:
   - duplicate IDs without `overrides` → conflict;
   - `overrides` pointing to an unknown ID → error;
   - a specialization that lowers severity, widens scope or weakens enforcement on a non-overridable rule → conflict;
   - an override of a rule in a locked level not listed in `editable_rules` → conflict;
   - contradictory deterministic guards or permissions (e.g. lower `allow` vs higher `deny` on the same action) → conflict;
   - a lower level disabling a mandatory overlay → conflict.
   Enforcement strength order: `permission` = `hook` > `check` > `review` > `prompt`.
9. **Include confirmed semantic conflicts** recorded in `.gradus/findings/` as blocking markers (§10.1).
10. **Budget.** Measure the rendered size per target; warn above the level budget; fail when a hard target limit would be exceeded (e.g. the Antigravity rule size limit), unless the compiler can split the content.
11. **Emit** the resolved view with its hash; update `view_hash` in the lock when run with `--write`.

Any conflict makes `verify` fail until it is resolved by changing a rule, declaring an override as allowed by the owner, or approving an exception.

### 10.1 Semantic conflict check with a model **[Specified as advisory]**

Contradictions between rules written in natural language cannot be detected deterministically. Gradus adds an **advisory** check:

1. `gradusctl conflicts --candidates` lists candidate pairs: rules from different levels or overlays with overlapping scope and shared tags or similar terms, plus every new or changed rule paired with the existing ones. Pairs already judged (same content hashes) are skipped.
2. The skill `gradus-conflicts` asks the model to classify each pair: `contradiction`, `duplicate`, `redundant`, `compatible`, with a short justification and confidence.
3. Results are written as evidence of kind `semantic-conflicts` with status `suspected`.
4. A human decides: **confirm** (becomes a blocking conflict in `.gradus/findings/`), **dismiss** (recorded with the content hashes so it is not asked again), or **fix** the rules.

It runs when rules change (authoring, lock updates, changelog reviews) or on demand, never at every session start. A high-capability model is recommended. It never resolves anything on its own and never blocks without human confirmation.

---

## 11. Enforcement **[Decided; MVP targets Specified]**

### 11.1 Three levels of guarantee

| Level | Guarantees | Where |
|---|---|---|
| 1. Instructions | Resolved view, skills, rejected rationalizations | Every agent (AGENTS.md, SKILL.md and native formats) |
| 2. In-session enforcement | Hooks, permissions, command blocks, write-time checks | Agents that support them; declared degraded mode elsewhere |
| 3. Universal enforcement | `verify`, `check` in git hooks and CI; evidence tied to commits | Independent of the agent |

Level 3 is the guarantee that matters most: whatever agent produced the code, nothing reaches the main branch without the same checks.

### 11.2 Enforcement map

| Rule kind | Enforcement |
|---|---|
| `MUST`, deterministically checkable on agent actions | `hook` or `permission` + `check` + prompt |
| `MUST`, deterministically checkable on code | `check` in git hooks and CI + prompt |
| `MUST`, not deterministic | prompt with rationalizations + mandatory `review` |
| `SHOULD` | prompt + `review` with warning severity |
| `MAY` | prompt only |

Guiding example: "do not commit without an explicit request" is deterministic on agent actions, so it becomes a hook that asks or denies `git commit`, not just a sentence in an instruction file.

### 11.3 MVP targets

**Claude Code**

| Gradus element | Target |
|---|---|
| Project view | `CLAUDE.md` (managed section delimited by markers) and `.claude/rules/gradus-*.md`, using path frontmatter for scoped rules |
| User view | Git-ignored local instruction file (`CLAUDE.local.md` or an equivalent git-ignored import, depending on the Claude Code version) |
| Skills | `.claude/skills/gradus-*/SKILL.md` |
| `hook` rules | `PreToolUse` hooks in `.claude/settings.json` calling `gradusctl guard` |
| `permission` rules | `permissions.deny` / `ask` in `.claude/settings.json` |
| Session start | `SessionStart` hook calling `gradusctl status --brief` (drift warning) |

**Antigravity**

| Gradus element | Target |
|---|---|
| Project view | `AGENTS.md` managed section, and `.agents/rules/gradus-*.md` files with mandatory trigger frontmatter (`always_on` for global rules, `glob` for scoped rules) |
| Size limit | Each rule file stays below the Antigravity rule size limit (12,000 characters); the compiler splits by level and topic |
| User view | `.agents/rules/gradus-user-*.md` (git-ignored) |
| Skills | `.agents/skills/gradus-*/SKILL.md` |
| `hook` rules | `.agents/hooks.json`, `PreToolUse` on `run_command` and file-writing tools, returning `deny` or `ask` |
| `permission` rules | Documented for the CLI user settings; the workspace-level guarantee relies on hooks |
| Session start | `SessionStart` hook calling `gradusctl status --brief` |

**AGENTS.md** covers level 1 for every other agent that reads it.

Generated content is written only inside delimited managed sections or in files owned by Gradus; content written by people is never overwritten.

### 11.4 Git hooks: integration with existing managers **[Decided]**

Gradus does not ship its own hook framework (P14).

| Manager | Characteristics | Integration |
|---|---|---|
| pre-commit | Widely used, language-agnostic, `.pre-commit-config.yaml`, hooks distributed from repositories | The Gradus repository publishes a `.pre-commit-hooks.yaml`; users add one entry |
| lefthook | Single binary, fast, parallel, `lefthook.yml` | `gradusctl hooks` prints or inserts the snippet |
| husky | Common in JavaScript projects, scripts in `.husky/` | `gradusctl hooks` prints or inserts the snippet |
| Native git hooks | No dependency; `core.hooksPath` | Fallback: `.githooks/` plus `core.hooksPath`, only with explicit consent |

`hooks.manager: auto` detects the manager in use. The pre-commit stage runs `gradusctl verify` and `gradusctl check --staged`. Model-based review (`gradus-review`) is run by the agent before committing, not by git hooks, because it needs a model.

### 11.5 CI

Any CI can run `gradusctl verify` and `gradusctl check --base <ref>`. Exit codes: `0` pass, `1` violations, `2` error. An error is never reported as a pass: a missing or failing checker produces exit code `2`. A ready-made GitHub Action is planned for Phase 1; other CI systems use the same commands.

## 12. Evidence **[Specified]**

Where evidence lives is configurable, because pipelines differ between teams.

| Sink | Behavior | Typical use |
|---|---|---|
| `none` | Nothing is stored | Personal projects |
| `local` | JSON records in `.gradus/evidence/` (git-ignored), with retention (default 30 days) | Developer machines |
| `ci-artifact` | Records written to a directory the CI uploads as an artifact | Teams with structured pipelines |
| `repo` | Records committed under a configured path | Small teams wanting history in git (opt-in) |
| `command` | Each record passed on stdin to a configured command (e.g. upload to object storage or an internal service) | Organizations with their own archive |

Several sinks can be active at once, and sinks can differ between local runs and CI (`evidence.ci.sinks`). Records follow §9.9. Evidence never includes data about individual activity beyond git metadata.

## 13. Exceptions and break-glass **[Specified]**

- Every exception has an explicit expiry date, a reason, a scope and approvers (§9.10).
- Each level declares `max_duration` and its approvers. There is **no global default**: a level that allows exceptions must declare its own maximum. Example templates use 30 days only as an illustration.
- Expired exceptions are reported by `verify` and stop waiving the rule.
- **Break-glass**, in hotfix style: during an incident a rule can be bypassed with a mandatory reason, a short duration, notification to the level owners and a mandatory post-review date. Levels may disable break-glass (`break_glass: false`), for example for `fail-closed` levels.

## 14. Learning loop **[Specified]**

1. **Capture.** `gradus-learn` detects explicit corrections, repeated requests, reverted changes and signs of frustration (in any language) and records **lessons**. It records little and well: "do not save" is a valid outcome.
2. **Classify.** Each lesson gets a suggested level (using charters) and a suggested enforcement. A lesson that is deterministic and `MUST`-like suggests a `hook` or `check`.
3. **Promote.** A promotion is proposed after **2 occurrences** or on explicit confirmation (thresholds configurable). User-level promotions are applied after the user confirms; promotions to other levels become proposals to the level approvers (pull request on the level source).
4. **Never automatic.** A harsh tone does not always mean a missing rule; every promotion needs human approval.

Lessons are local and never committed automatically. Only the promoted rule, written by `gradus-author`, enters a level.

---

## 15. Architecture **[Decided]**

Gradus separates judgment from enforcement.

| Non-deterministic: skills (or humans) | Deterministic: `gradusctl` |
|---|---|
| Analyze context and propose an initial pyramid | Resolve levels, overlays and exceptions; produce the view |
| Write and edit rule text | Validate schemas, inherited constraints, structural conflicts |
| Classify the right level for a rule | Manage lockfile, hashes, versions (signatures later) |
| Review code against non-mechanical rules | Run deterministic checks and external checkers |
| Detect corrections and frustration | Record evidence, exceptions and decisions |
| Judge candidate semantic conflicts | Emit verdicts for git hooks and CI; guard agent actions |

**The CLI owns all state changes**: lock updates, approvals, exceptions, evidence, finding status. Skills edit content and call the CLI; they never write those fields directly.

**Implementation language: Go.** Chosen for fast development by a single maintainer, straightforward cross-compilation for macOS, Linux and Windows, single static binaries, fast start-up inside agent hooks, and an approachable codebase for contributors. Comparable tools (stele, Walden) use the same approach.

**CLI requirements**: single executable for macOS, Linux and Windows; no runtime dependencies; stable and documented JSON output; exit codes `0`/`1`/`2`; fast enough to run inside agent hooks (target: well under one second for `guard` and `status`).

**Implementation notes**: git sources are handled by invoking the installed `git` command rather than a git library, so the user's existing credentials and configuration are reused (§8) and no native dependencies are needed; libraries are chosen among actively maintained ones.

## 16. CLI `gradusctl` **[Specified for MVP; later commands Proposed]**

| Command | Purpose | Phase |
|---|---|---|
| `init` | Create `gradus.yaml`, the project level and git-ignore entries; optionally from `gradus-onboard` output | MVP |
| `resolve` | Fetch and verify sources, update the lock, emit the resolved view (`--json`, `--write`) | MVP |
| `compile` | Generate target files for the configured agents | MVP |
| `verify` | Validate definition, constraints, integrity, conflicts, generated files up to date, expired exceptions | MVP |
| `check` | Run deterministic checks and external checkers (`--staged`, `--base <ref>`) | MVP |
| `guard` | Evaluate an agent action against `hook` rules; called by agent hooks; returns allow/ask/deny | MVP |
| `status` | Short summary: lock state, drift, conflicts, pending lessons | MVP |
| `explain <id>` | Origin, level, overrides, history and rationale of a rule | MVP |
| `conflicts` | List semantic conflict candidates; record decisions | MVP |
| `lessons` | List, show, dismiss lessons; record promotion decisions | MVP |
| `reconcile` | List drift items and apply the chosen action | MVP |
| `hooks` | Detect the git hook manager and print or insert configuration | MVP |
| `evidence` | Write records to configured sinks; list local records | MVP |
| `exception` | Request, approve, list, expire exceptions; break-glass | Phase 1 |
| `test` | Run rule tests from rule examples | Phase 1 |
| `simulate` | Apply a changed rule set to recent commits and report impact | Phase 1 |
| `doctor` | Diagnose configuration and explain fixes | Phase 1 |
| `handbook` | Generate the human onboarding handbook | Phase 1 |
| `changelog` | Readable diff between two versions of a level | Phase 2 |
| `toolchain`, `adapter` | Toolchain verification; adapter management | Phase 2 |
| `serve --mcp` | Expose the pyramid through MCP | Phase 3 |

## 17. Skills **[Specified for MVP]**

Skills follow the Agent Skills standard (name, description and standard fields only in the core); agent-specific fields are added by the compiler where supported. Names carry the `gradus-` prefix.

| Skill | Phase | Inputs | Outputs | Must not |
|---|---|---|---|---|
| `gradus-onboard` | MVP | Repository (rule files, skills, hooks, linters, CI, CODEOWNERS), git history, files supplied by the user (exported conversations, internal docs, decision records, policies), short interview | Context dossier; divergence report; draft pyramid with rules classified by level, each citing evidence and confidence; open questions | Apply anything without approval; send data anywhere except the model in use; keep secrets or personal data unredacted |
| `gradus-author` | MVP (basic) | User intent, documents, lessons to promote; charters | Rule files with correct metadata, rationalizations and examples; redirection when content belongs to another level; proposals for locked levels | Approve; write to locked levels directly |
| `gradus-review` | MVP | Staged changes, a branch, or existing code; resolved view | Findings citing rule ID, level, severity and source, with confidence; evidence record via CLI | Report low-confidence findings as violations; modify code unless asked |
| `gradus-learn` | MVP | Current session signals; optionally supplied conversation history | Lessons; promotion proposals with suggested level and enforcement | Promote without approval; store unredacted excerpts |
| `gradus-reconcile` | MVP | Drift items from `gradusctl reconcile` | Explanation of each drift and options allowed by the user's permissions: restore, move to the right editable level, propose upstream, request an exception, postpone (never for `fail-closed`) | Execute actions itself; the CLI executes |
| `gradus-conflicts` | MVP | Candidate pairs from `gradusctl conflicts` | Classification with justification and confidence | Block or resolve on its own |
| `gradus-link` | Phase 2 | Installed tools and skills | Compatibility and security audit, adapter manifest, binding proposal | Link a tool that violates the pyramid without an approved decision |
| `gradus-assess` | Phase 3 | Repository and configuration | Maturity assessment of processes (never of people) | Evaluate individuals |

Loading the pyramid at session start is not a skill: it is done by the compiled instruction files plus a `SessionStart` hook that warns about drift.

## 18. Main flows **[Specified]**

**First adoption.** Install the CLI and skills → `gradus-onboard` analyzes the repository and supplied files → dossier, divergences and draft pyramid are discussed by the people involved → `gradusctl init`, `resolve`, `compile` → hooks configured with the existing manager → CI step added.

**Daily work.** The agent loads the compiled view → hooks guard deterministic actions → before committing, the agent runs `gradus-review` on staged changes → git hooks run `verify` and `check` → CI runs the same commands and stores evidence.

**Changing a rule.** Proposed by a person or by `gradus-author`/`gradus-learn` → level classified via charters → editable level: applied after approval; locked level: pull request on its source → after release, a lock update pull request in consuming projects contains the readable changelog (Phase 2) → evidence produced under the previous lock is recognizable as stale.

**Repeated frustration.** The user asks twice not to commit autonomously → `gradus-learn` records the lesson and proposes a `hook` rule at the user level (or at the team level if the user says so) → after approval, `gradus-author` writes the rule and `compile` installs the guard.

**Drift.** Someone edits a generated file or a locked level → `verify` or the `SessionStart` hook reports it → `gradus-reconcile` explains the options → the CLI applies the chosen one.

## 19. Obligations, bindings, adapters and toolchain **[Decided; Specified]**

- **Obligations** are declared by a level in abstract form, optionally with `allowed_providers`.
- **Bindings** are declared by a lower level and say how the obligation is satisfied in a scope (Walden, spec-kit, an internal process).
- **Adapters** inject the resolved view into the provider's files inside a managed `<!-- gradus:begin/end -->` section (e.g. `.walden/constitution.md`, `.specify/memory/constitution.md`), verify the provider's evidence (e.g. Walden's release check verdict), check required versions, and detect drift inside the managed section while preserving custom user content outside the markers (D-83). Official adapters are maintained by the project; custom ones are declarative manifests generated by `gradus-link`; community adapters can be contributed back.
- **Toolchain**: a level may declare skills, tools, MCP servers and plugins as `required`, `recommended` or `forbidden`. Lower levels may add, never remove a `required` item or enable a `forbidden` one. Where possible the declaration is translated into native mechanisms (managed settings, organization-provisioned skills); elsewhere it is checked by the CLI.

```yaml
obligations:
  - id: OBL-SDD-01
    rule: "New behavior requires an approved specification before code."
    allowed_providers: [walden, spec-kit]
bindings:
  - obligation: OBL-SDD-01
    provider: walden
    requires: { cli: ">=1.4" }
    scope: ["services/payments/**"]
toolchain:
  required:
    - { type: skill, id: gradus-review }
  forbidden:
    - { type: mcp, id: "*-public-pastebin" }
```

## 20. Other agents, chat agents and MCP **[Decided; Specified]**

- **Other coding agents** (Phase 2): Codex, GitHub Copilot, and OpenCode targets compile `AGENTS.md`, native instructions, and skills. A compatibility matrix (`docs/compatibility-matrix.md`) tracks agent × feature support.
- **Chat agents & enterprise distribution** (Phase 3, D-86): `gradusctl compile --chat-bundle <path.zip>` packages shared rules (excluding `user` rules) and embedded skills into a byte-reproducible ZIP for chat products; `gradusctl compile --managed-settings <path>` emits enterprise `managed-settings.json` for machine-wide rollout.
- **MCP server** (Phase 3, `gradusctl serve --mcp`, D-87): zero-dependency JSON-RPC 2.0 stdio server exposing five strictly non-mutating, read-only annotated tools (`resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, `get_toolchain`).
- **Background and CI agents** (Specified, D-81): compiled project files are committed; `--agent-profile <dir>` (or `GRADUS_AGENT_PROFILE`) replaces the personal `user` level with a team-defined profile (e.g. "open draft pull requests only, never merge"); their pull requests pass the same CI checks; subagents inherit the resolved view.

## 21. Model policy **[Decided; Specified]**

Risk addressed: using a less capable model to save quota on operations that affect others.

- Policy uses **capability tiers** (e.g. `tier-high`, `tier-standard`, `tier-light`) mapped to concrete models per agent by the level that declares them, and **risk classes** of operations (e.g. review and authoring of shared levels → `tier-high`).
- Where the agent supports it, the compiler sets the model on skills or subagents (in Claude Code, the `model` frontmatter field; this is an extension, not part of the open standard, so it is generated per agent).
- Everywhere, the tier used is recorded in evidence; CI may reject evidence produced below the required tier. Where the model in use cannot be verified, the rule remains a strong recommendation and the compatibility matrix says so.

## 22. Security and privacy **[Decided]**

- **Prompt injection.** Only sources declared in `gradus.yaml` contribute rules. Everything else (repository files, issues, comments, documents, tool output, third-party skills) is data. The resolved view records provenance for every rule. In CI, changes from forks or untrusted pull requests cannot alter the pyramid used to evaluate them. The eval suite includes injection scenarios.
- **Onboarding inputs.** Secrets and credentials are always redacted; personal data is flagged and redacted by default; nothing is sent anywhere except the model already in use. The analysis itself must respect the organization's data rules.
- **Lessons and evidence** stay local by default and contain no data about individual activity (P16).
- **Code on developer machines.** Hooks, the CLI and adapters are documented transparently; security teams must be able to audit what they run. Third-party skills are audited by `gradus-link` before linking.
- **Supply chain.** Signed releases, checksums, pinned CI actions, minimal dependencies, SBOM.
- **Credentials** never appear in configuration files.

## 23. Quality and evaluation **[Decided]**

- **Skill development follows TDD for skills**: run realistic scenarios without the skill, record failures and rationalizations verbatim, write the minimal skill that fixes them, rerun until they pass; scenarios stay as a regression suite.
- **Rule tests**: compliant and non-compliant examples in rule files are run against `check` rules and used as review test cases (`gradusctl test`, Phase 1).
- **Published evaluation**: the repository publishes scenarios, method and results, rerunnable with one command, comparing agents with Gradus against a plain instruction-file baseline, per agent.
- **Dogfooding**: the Gradus repository is governed by Gradus.

## 24. Roadmap, MVP and Phase 0 exit criteria

### 24.1 Adoption phases **[Decided]**

| Phase | Context | Main capabilities |
|---|---|---|
| 0 | Personal repositories of the author | Core format, `gradusctl` MVP commands, local and git sources, lockfile, compilation for Claude Code, Antigravity and AGENTS.md, agent hooks, git hook integration, evidence sinks, skills `onboard`, `author` (basic), `review`, `learn`, `reconcile`, `conflicts` |
| 1 | Team | Shared team level, charters enforced in review, overlays with explicit activation, exceptions and break-glass, rule tests and impact simulation, GitHub Action, background and CI agents, onboarding handbook |
| 2 | Business unit | Locked remote levels with signatures and `min_version`, obligations, bindings, adapters (Walden, spec-kit), toolchain, lock update pull requests with readable changelog, `gradus-link`, Codex, Copilot and OpenCode targets |
| 3 | Organization | Conditional overlay activation by profile, framework mapping and audit export, managed settings generation, model policy, MCP server and chat distribution, maturity assessment, commit attribution |

Each phase adds capabilities without redoing earlier work: the personal pyramid of Phase 0 becomes the user level later.

### 24.2 Minimal personal setup **[Specified]**

```yaml
# gradus.yaml — personal use
schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
  - id: user
    source: { path: ~/.gradus/levels/user }
targets: [claude-code, antigravity, agents-md]
```

Per ADR 0001, `user` is always the most specific (lowest) level in the pyramid so that committed repository levels (`org`, `team`, `project`) never depend on or override rules that exist only in a developer's personal `~/.gradus/levels/user` directory, and CI can cleanly omit `user` without breaking override chains.

### 24.3 Phase 0 exit criteria **[Proposed — to be agreed with the author]**

Phase 0 is validated on a **real project of the author**, with real rules, in both Claude Code and Antigravity.

**Test setup**

- Three levels: a simulated `team` level in a separate local git repository (locked, to exercise git sources and the lockfile), the `project` level and the `user` level.
- At least ten real rules, including: two `hook` rules (e.g. no commit without explicit request; no edits to a protected path); two `check` rules using existing tools (e.g. secret scanning, a forbidden pattern); three `prompt`/`review` rules (e.g. naming, structure, tests); one path-scoped rule; one valid override; one invalid override (expected conflict).

**Exit criteria**

| # | Criterion |
|---|---|
| E1 | `resolve` and `compile` generate correct outputs for Claude Code, Antigravity and AGENTS.md; a second run produces no diff; Antigravity rule files respect the size limit. |
| E2 | `verify` detects a tampered locked level, outdated generated files and the invalid override, with the documented exit codes. |
| E3 | In both agents, an attempt to commit without an explicit request is blocked or requires confirmation through the agent hook. |
| E4 | `gradus-review` on staged changes reports a seeded violation citing the rule ID, and reports nothing on a clean change. |
| E5 | After two repeated corrections, `gradus-learn` has recorded a lesson and proposes a promotion at a sensible level; nothing is applied without approval. |
| E6 | A manual edit to a generated file is detected; `gradus-reconcile` offers the options and the CLI applies "restore" and "move to project level" correctly. |
| E7 | `gradus-conflicts` flags a seeded contradiction between two natural-language rules; a dismissed pair is not asked again. |
| E8 | Git hook integration works with at least one manager (pre-commit or lefthook); a CI run executes `verify` and `check` and stores evidence as an artifact. |
| E9 | An evaluation of at least ten scenarios runs in both agents with and without Gradus; compliance with Gradus is at least equal to the baseline on every scenario and higher on the deterministic ones; results are committed. |
| E10 | Following only the README, a fresh repository is set up for personal use end to end. |

## 25. Further capabilities **[Decided outcomes]**

| ID | Capability | Outcome | Phase |
|---|---|---|---|
| X-01 | Open specification (`spec/`) beside the usable kit | Accepted | 0 (draft), stabilized later |
| X-02 | Mapping rules to recognized frameworks (`controls`) with coverage report and audit export | Accepted, optional | 3 |
| X-03 | Rule tests and impact simulation on recent history | Accepted | 1 |
| X-04 | Reuse of existing checkers (linters, Semgrep/Opengrep, secret scanners, OPA, license checks) through a checker plugin protocol | Accepted | 0 |
| X-05 | Commit attribution through git trailers (agent, model tier, lock hash, review ID), describing the process, not people | Accepted, optional | 3 |
| X-06 | Metrics on activity | **Excluded** (P16) | — |
| X-07 | Rule lifecycle (status, review dates, expiry) and per-level context budget | Accepted, optional | 1 |
| X-08 | Break-glass procedure, hotfix style | Accepted, optional | 1 |
| X-09 | Background and CI agents | Accepted, optional | 1 |
| X-10 | Prompt injection defense | Accepted | 0 |
| X-11 | Example templates (personal, team, a few sectors), never prescriptive | Accepted as examples | 1 |
| X-12 | Onboarding handbook generated from the pyramid | Accepted | 1 |
| X-13 | Maturity model and assessment (processes, never people) | Accepted, optional | 3 |
| X-14 | Readable changelog delivered in lock update pull requests | Accepted | 2 |
| X-15 | Dogfooding and project governance (`SECURITY.md`, threat model, ADRs, signed releases, SBOM, EN documentation with IT translation) | Accepted | 0 onward |
| — | Approvals through an API for remote levels | Kept in design | Deferred |

## 26. Repository, naming and release **[Decided]**

| Item | Value |
|---|---|
| Project name | Gradus (Latin for "step") |
| Repository | `github.com/Milosauro/gradus-ai`; moving it to an organization is to be decided before the first installable release, because install paths contain the owner |
| Organization name | `gradus-ai` reserved as a preference (not yet created) |
| CLI and packages | `gradusctl` (bare `gradus` is taken on npm, PyPI and crates.io, and by a JVM build tool) |
| Skill prefix | `gradus-` |
| Domain | To be chosen (`gradus.dev` is in use) |
| Trademark | No evident conflict in developer software; a full search on EUIPO/TMview and WIPO (classes 9 and 42) is pending |
| License | Apache-2.0 |
| Repository structure | Monorepo: `spec/`, `skills/`, `cmd/gradusctl/`, `internal/`, `adapters/`, `templates/`, `examples/`, `evals/`, `docs/` |
| Documentation | English canonical; Italian translations non-normative |

## 27. Open items

| # | Item |
|---|---|
| O-1 | Phase 0 exit criteria (§24.3): confirm with the author, choose the test project. |
| O-2 | Trademark search on EUIPO/TMview and WIPO; domain choice; organization creation and repository move (release topics, not design blockers). |
| O-3 | Exact budget values per level and per target, to be measured on real rules during Phase 0. |

## 28. Decision log

IDs are stable across document versions; superseded decisions are marked.

| ID | Decision | Status |
|---|---|---|
| D-01 | Gradus is a governance layer, not an SDD framework or a rule synchronizer | Decided |
| D-02 | Skills for judgment, CLI for enforcement | Decided |
| D-03 | Restriction invariant with explicit per-rule overrides | Decided |
| D-04 | Levels defined in configuration; number and names free | Decided |
| D-05 | Locking is a property of every level | Decided |
| D-06 | Pyramid definition treated as critical code; modes A/B/C chosen by whoever organizes the work | Decided |
| D-07 | Layered pyramid structure with inherited constraints | Decided |
| D-08 | Overlays beside levels | Decided |
| D-09 | Remote sources with lockfile, hashes, optional signatures, `min_version`, per-level offline policy | Decided |
| D-10 | Conflicts marked and blocking, never resolved silently by the model | Decided |
| D-11 | Lessons separate from rules, with explicit promotion | Decided |
| D-12 | Abstract obligations separate from bindings and adapters | Decided |
| D-13 | Workflow rules follow the code, not the person | Decided |
| D-14 | Pyramid is the single source of truth; generated outputs with drift detection | Decided |
| D-15 | `gradus-onboard` analyzes repository, git history and user-supplied files | Decided |
| D-16 | `gradus-link` with mandatory compatibility and security audit | Decided |
| D-17 | Policy loaded wherever output is produced | Decided |
| D-18 | Rules state rejected rationalizations; skills developed against eval scenarios | Decided |
| D-19 | The agent never approves | Decided |
| D-20 | Walden and spec-kit supported as providers through adapters, not as foundations | Decided |
| D-21 | Multi-agent compatibility on open standards with per-agent extensions generated | Decided |
| D-22 | Three levels of guarantee; universal enforcement in git hooks and CI | Decided |
| D-23 | Distribution to chat agents via skills and an MCP server | Decided, deferred |
| D-24 | Toolchain declarable per level | Decided, deferred |
| D-25 | Model policy by capability tiers and risk classes, with tier recorded in evidence | Decided in principle, deferred |
| D-26 | `gradus-reconcile` with options depending on the user's permissions | Decided |
| D-27 | Conditional overlay activation through a governed profile | Decided |
| D-28 | `gradus-author` guided by level charters | Decided |
| D-29 | Charter for every level | Decided |
| D-30 | Open specification beside a directly usable repository | Decided |
| D-31 | Minimal mandatory core; personal use is first-class | Decided |
| D-32 | Framework mapping is optional | Decided |
| D-33 | Rule tests and impact simulation | Decided |
| D-34 | Reuse of existing checkers and tools | Decided |
| D-35 | Prompt injection defense in the rule chain | Decided |
| D-36 | Lifecycle, budget, break-glass, assessment are optional | Decided |
| D-37 | Templates are non-prescriptive examples | Decided |
| D-38 | Onboarding handbook generated from the pyramid | Decided |
| D-39 | Changelog delivered through lock update pull requests | Decided |
| D-40 | No metrics on people's activity | Decided |
| D-41 | Adoption path: personal → team → business unit → organization | Decided |
| D-42 | Commit attribution optional | Decided |
| D-43 | Background and CI agents optional, with agent profile | Decided |
| D-44 | Phase 0 targets: Claude Code and Antigravity, plus AGENTS.md | Decided |
| D-45 | Name neutral with respect to tools and trademarks | Decided |
| D-46 | Implementation in a dedicated project by the author; this document is the design reference | Decided |
| D-47 | No blocking overlap with prior art | Decided |
| D-48 | Absorb ideas from prior art, not dependencies | Decided |
| D-49 | Communicated value is governance around levels | Decided |
| D-50 | Project name: Gradus | Decided |
| D-51 | Identifiers: `gradusctl` for CLI and packages; `gradus-` skill prefix; `gradus-ai` organization name preferred | Decided |
| D-52 | Repository at `Milosauro/gradus-ai`; possible move to an organization before the first installable release | Decided |
| D-53 | License Apache-2.0 | Decided |
| D-54 | File names: `gradus.yaml`, `gradus.lock`, `.gradus/`, `level.yaml`, schema `gradus/v1` | Decided |
| D-55 | One Markdown file per rule with YAML frontmatter; level derived from location | Decided |
| D-56 | Overlay strength inserts the overlay after the named level; between overlays the more restrictive structural rule wins, otherwise blocking conflict | Decided |
| D-57 | Lesson promotion proposed after 2 occurrences or explicit confirmation (configurable) | Decided |
| D-58 | Remote level approvals through pull requests; API approvals kept in design, deferred | Decided |
| D-59 | Onboarding redaction defaults: secrets always, personal data flagged and redacted | Decided |
| D-60 | Agents after the MVP: Codex, Copilot, OpenCode | Decided |
| D-61 | `editable_rules` by rule ID; overrides restrict unless the rule is overridable | Decided |
| D-62 | Profile vocabulary: `data`, `exposure`, `domain`, `client`, extensible | Decided |
| D-63 | Model tiers and MCP server deferred | Decided |
| D-64 | Exceptions: explicit expiry; maximum duration declared by each level; no global default | Decided |
| D-65 | Commit trailers deferred to Phase 3 | Decided |
| D-66 | Technology overlays are ordinary overlays | Decided |
| D-67 | Antigravity enforcement via `.agents/hooks.json` and rule files with trigger frontmatter, within the rule size limit | Decided |
| D-68 | Own generators for MVP targets, no rulesync dependency | Decided |
| D-69 | CLI implemented in Go; git operations through the installed `git` command (§15) | Decided |
| D-70 | Advisory semantic conflict check by a model, human-confirmed, cached by content hash | Decided |
| D-71 | Configurable evidence sinks: none, local, ci-artifact, repo, command | Decided |
| D-72 | Integration with existing git hook managers; native hooks only as fallback with consent | Decided |
| D-73 | Exit codes 0/1/2; an error is never a pass | Decided |
| D-74 | Design document in English (canonical) with a non-normative Italian translation | Decided |
| D-75 | Normative precedence: `spec/v1/schema/*.json` for formats, `gradus-design.md` + ADRs for semantics, `spec/v1/SPEC.md` as descriptive guide | Decided |
| D-76 | Anti-tamper supports `ask` (default) and `deny` modes, protecting governance files, git hook configs (`lefthook*.yml`, `.pre-commit-config.yaml`, `.husky/**`), CI workflows, agent configs (`opencode.json`, `.codex/config.toml`), and hook-bypass flags/env vars (`--no-verify`, `core.hooksPath`, `HUSKY=0`, `LEFTHOOK=0`, `SKIP=...`) | Decided |
| D-77 | Human-only governance commands (`exception approve`, `exception break-glass`, `conflicts dismiss`, `lessons update`, `hooks install`) are gated by anti-tamper | Decided |
| D-78 | Exception `approvers` are checked locally against level policy and `max_duration` is validated at request time; PR review + `CODEOWNERS` on `.gradus/exceptions/` provides the authoritative identity boundary | Decided |
| D-79 | `gradusctl resolve --update` permits intentional locked-ref upgrades on `source.ref` changes while still failing (exit `2`) on content tampering under an unchanged `ref` | Decided |
| D-80 | `view_hash` excludes the personal `user` level and `lock_hash` so `gradus.lock` and compiled project markers stay invariant across developers | Decided |
| D-81 | `--agent-profile` / `GRADUS_AGENT_PROFILE` substitutes a team-defined profile for the personal `user` level in background and CI agent runs | Decided |
| D-82 | `gradusctl status --hook` (`SessionStart`) is non-blocking (always exits `0`) and emits a fast summary banner of lock, conflict, drift, and lesson counts | Decided |
| D-83 | `gradusctl adapter verify` inspects only the managed `<!-- gradus:begin/end -->` block in provider files, preserving custom user sections outside the markers | Decided |
| D-84 | Conditional overlays (`activation: profile`, `level.yaml` `activation_rules`, `overlay.yaml` `when`) evaluate extensible `profile` attributes deterministically; `required: true` emits a blocking `overlay` conflict when absent or `disabled` | Decided |
| D-85 | Compliance `controls:` inherit monotonically across rule overrides (`gradusctl audit coverage\|export`); `models:` (`tier-high > tier-standard > tier-light`) enforces monotonic tightening and `verify --check-model-tier` checks self-reported `model_tier` in local evidence | Decided |
| D-86 | Enterprise `managed-settings.json` (`--managed-settings`) and byte-reproducible chat ZIP bundles (`--chat-bundle`, excluding `user` rules) are opt-in compile outputs; `guard` and `status --hook` walk upward to `gradus.yaml` and silently allow/exit `0` outside Gradus repositories | Decided |
| D-87 | `gradusctl serve --mcp` runs a zero-dependency JSON-RPC 2.0 stdio server exposing five read-only/non-mutating tools (`resolve_rules`, `explain_rule`, `check_text`, `propose_rule`, `get_toolchain`), validating `propose_rule` levels against the resolved pyramid | Decided |
| D-88 | `gradusctl assess` and `gradus-assess` evaluate five process dimensions (`L1`–`L5`) and never measure or rank individuals (P16) | Decided |
| D-89 | Optional `Gradus-*` commit attribution trailers (`gradusctl trailers generate\|verify`) verify `Gradus-Lock-Hash` against the commit's own `gradus.lock` (`git show <ref>:gradus.lock`) whenever present; `resolve --write` refuses to update `gradus.lock` when blocking conflicts exist | Decided |

## 29. Glossary (English – Italian)

| English | Italiano |
|---|---|
| level | gradino, livello |
| pyramid | piramide |
| charter | statuto |
| rule | regola |
| overlay | overlay (profilo trasversale) |
| applicability profile | profilo di applicabilità |
| obligation | obbligo |
| binding | binding (collegamento a uno strumento) |
| adapter | adapter |
| toolchain | toolchain (strumenti dichiarati) |
| lesson | lesson (osservazione registrata) |
| promotion | promozione |
| resolved view | vista risolta |
| lockfile | lockfile |
| evidence | evidenza |
| exception | eccezione |
| break-glass | procedura di emergenza |
| drift | drift (divergenza) |
| restriction invariant | invariante di restrizione |

## 30. References

- Agent Skills standard: https://agentskills.io
- AGENTS.md: https://agents.md
- Model Context Protocol: https://modelcontextprotocol.io
- Claude Code documentation (memory, settings, hooks, skills): https://code.claude.com/docs
- Antigravity documentation (rules, skills, hooks, permissions): https://antigravity.google/docs
- Cursor rules: https://cursor.com/docs/rules
- Walden: https://github.com/andrearaponi/walden
- spec-kit: https://github.com/github/spec-kit
- Superpowers: https://github.com/obra/superpowers
- Anthropic plugins (code-review, hookify, claude-md-management): https://github.com/anthropics/claude-plugins-official
- claude-reflect: https://github.com/BayramAnnakov/claude-reflect
- stele: https://github.com/lictor-ai/stele
- agentlaw: https://pypi.org/project/agentlaw/
- org-agentic-toolkit: https://pypi.org/project/org-agentic-toolkit/
- daimyo: https://pypi.org/project/daimyo/
- UCP spec: https://github.com/corepack-ai/ucp-spec
- pre-commit: https://pre-commit.com · lefthook: https://github.com/evilmartians/lefthook · husky: https://typicode.github.io/husky
