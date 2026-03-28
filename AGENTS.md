# AGENTS.md

## Release Workflow

- This repo uses an `Unreleased` changelog workflow.
- Keep all not-yet-published changes under `## [Unreleased]` in `CHANGELOG.md`.
- Do not create a new versioned changelog section on feature branches just because code changed.
- When a real release is cut, move the contents of `Unreleased` into a dated versioned section for that release.

## Version Scheme

- This repo uses a three-digit semantic version scheme: `MAJOR.MINOR.PATCH`.
- Do not invent a fourth digit such as `MICRO`.
- Git tags should use the same three-digit version with a `v` prefix, for example `v0.1.2`.
- `VERSION` should stay on the current unreleased target version until that release is actually published.

## Practical Rules For Agents

- If `0.1.2` is the next unreleased release, keep accumulating unreleased changes under `Unreleased` and keep `VERSION` at `0.1.2`.
- Do not bump `VERSION` from `0.1.2` to `0.1.3` unless the project is intentionally starting work on the next release after `0.1.2` has shipped.
- Before changing `VERSION` or `CHANGELOG.md`, check existing git tags so the files match actual published releases.
- Prefer consistency with published tags over inventing new local version numbers.
