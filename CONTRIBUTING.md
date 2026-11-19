# Contributing to Gradus

Thank you for your interest in contributing to **Gradus**!

## 1. Ground Rules

- **Language:** All repository content — code, comments, documentation, commit messages, issues, and pull requests — **MUST be in English**.
- **Governance:** Gradus is governed by its design specification (`docs/design/gradus-design.md`) and follows Spec-Driven Development via **Walden** (`walden v0.14.0+`).
- **Determinism:** Core enforcement resides in `gradusctl` (Go, `CGO_ENABLED=0`).

## 2. Branching and Workflow

- We follow a **Trunk-Based Development** model:
  - `main` is our integration trunk and must always pass all tests and validations.
  - Create short-lived branches off `main`:
    - `feat/<feature-name>` for new features or Walden specs.
    - `fix/<slug>` for bugfixes.
    - `docs/<slug>` for documentation updates.
    - `spec/<slug>` for schema and specification work.
    - `ci/<slug>` for pipeline changes.
  - Release branches (`release/vX.Y`) are created on demand for minor version freezes. Bugfixes land on `main` first and are cherry-picked to the active release branch.

## 3. Commit Message Convention

We follow **Conventional Commits** in English:

```text
<type>(<scope>): <short summary>

[optional body]
```

Allowed types:
- `feat`: A new feature or capability.
- `fix`: A bug fix.
- `docs`: Documentation changes only.
- `spec`: Changes to specification or schema files.
- `test`: Adding or updating tests.
- `ci`: Changes to CI/CD workflows and configuration.
- `refactor`: Code change that neither fixes a bug nor adds a feature.
- `chore`: Maintenance tasks, dependency updates.

## 4. Walden Spec-Driven Workflow

For any new capability or behavioral change:
1. Initialize a feature: `walden feature init <feature-name>`
2. Author `requirements.md` using EARS criteria.
3. Review and approve requirements via `walden review approve <feature-name> --phase requirements`.
4. Author `design.md` covering the 6 mandatory architectural sections.
5. Review and approve design.
6. Author `tasks.md` with non-vacuous structured verification commands.
7. Execute tasks using TDD, verifying with `walden task complete <feature-name> <task-id>`.

## 5. Pull Request Checklist

Before submitting a PR, ensure:
- [ ] `CGO_ENABLED=0 go test -race -count=1 ./...` passes.
- [ ] `go vet ./...` and `golangci-lint run ./...` pass with zero warnings.
- [ ] `walden validate --all --json` returns exit code 0.
- [ ] All files are written in canonical English.
- [ ] No git-tracked files expose personal configuration or secrets.
