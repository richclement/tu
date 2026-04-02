# Document Patterns

Use these patterns when `tu` shows markdown-heavy hotspots or oversized repo guidance.

## Root Guidance Docs

Treat `AGENTS.md`, `CLAUDE.md`, and similar root guidance files as maps.

Keep:

- durable repo-wide rules
- routing guidance for where deeper knowledge lives
- task-specific entry points
- a small amount of non-duplicated operational policy

Move out:

- long release procedures
- implementation detail for one subsystem
- duplicated architecture explanations
- repeated instructions that already live in a canonical leaf doc

Rule of thumb: root guidance should tell an agent where to go next, not force it to load the entire repo's operating manual.

## Index Pattern For Architecture, Design, And Implementation

Each doc family should have an index file that answers:

- what this doc family covers
- who owns it
- how current it is
- which leaf doc to read for which question
- which docs are canonical versus historical

The index should reduce reading, not add another layer of prose.

Rule: prefer a visible top-of-doc metadata block unless the repo already has a frontmatter-based documentation system.

### Minimal index fields

Use a lightweight metadata block near the top by default. Switch to frontmatter only when the repo already standardizes on frontmatter for tooling, validation, or doc generation.

```md
# Architecture Index

Owner: Platform team
Status: Active
Last updated: 2026-04-01
Audience: Engineers and agents changing core services
Source of truth: This index routes to the canonical architecture docs listed below.
```

Frontmatter is acceptable and agents will understand it quickly, but do not introduce it casually into a repo that otherwise uses plain Markdown. Prefer one metadata style per doc family so trust signals do not drift.

### Per-entry guidance

For each linked doc, include:

- purpose
- owner when different
- freshness or status if relevant
- when to read it
- when not to read it

Example:

```md
- [Scanner pipeline](scanner-pipeline.md)
  - Use for directory walk, ignore handling, and worker-pool design.
  - Skip if you only need CLI flag semantics.
```

## Progressive Disclosure Pattern

For large markdown files:

1. Put the scope and metadata at the top.
2. Add a heading map or table of contents.
3. Split into leaf docs when sections answer different questions.
4. Keep reference depth shallow: index -> leaf.

Avoid:

- one document that mixes product intent, architecture, implementation details, and release checklists
- link mazes that require opening several intermediary docs before a leaf
- duplicate summaries that drift from the canonical source

## Split Criteria For Markdown

Split by the question someone is trying to answer.

Good splits:

- design goals vs implementation plan
- architecture overview vs subsystem references
- operator runbook vs contributor guide
- API contract vs migration notes

Bad splits:

- arbitrary page count
- arbitrary heading count
- "part 1 / part 2" with no responsibility boundary

## Trust Signals

Large markdown without trust signals is expensive for agents to use. Add:

- owner
- status
- last updated
- intended audience
- source-of-truth language

These signals help the agent decide whether to trust the doc and whether to continue reading.

## Common Smells

- giant root `AGENTS.md` or `CLAUDE.md`
- architecture doc with no index and no owner
- implementation doc that embeds stale rollout instructions
- several medium-size docs with overlapping instructions and no canonical source
