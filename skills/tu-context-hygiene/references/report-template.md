# Report Template

Use this structure when presenting findings to the developer. Keep the posture advisory first.

## Default Stance

- Report findings before suggesting edits.
- Ask before proposing concrete file changes.
- Keep each recommendation tied to evidence from `tu` and the progressive inspection pass.

## Template

```md
## Top Findings

- `<path>` (`<tokens>` tokens, `<class>`, `<severity>`): `<one-line finding>`
- `<path>` (`<tokens>` tokens, `<class>`, `<severity>`): `<one-line finding>`

### `<path>`

**Why This File Is Hard For Agents**

- `<size, trust, or navigability problem>`
- `<ownership, freshness, or cohesion problem>`

**What To Inspect Next**

- `<specific section, heading, function, or subtree to inspect>`
- `<specific follow-up question or comparison>`

**Recommended Refactor Pattern**

- `<map-plus-leaf-docs, index-plus-leaf-docs, seam-based extraction, test split, or exclude/deprioritize>`
- `<why that pattern fits this file>`

**Questions For The Developer**

- `<ownership or canonical-source question>`
- `<whether this file is intentionally reference-oriented or just accumulated>`

**Suggested Acceptance Criteria**

- `<what would make this file agent-readable after refactor>`
- `<what metadata, index, or seam should exist afterward>`
```

## Good Final-Line Summary

Close with a repo-level summary such as:

- "The main issue is not just raw token size; it is that the largest files mix several responsibilities and do not help the agent decide what to read next."
- "The root guidance is healthy, but the design docs need an index so agents can inspect them purposefully."
- "Most of the apparent hotspots are fixtures and generated artifacts, so the next pass should exclude them before recommending code refactors."

## Acceptance Criteria Patterns

### Root guidance docs

- Root doc stays within the target budget for its class.
- Root doc routes to canonical leaf docs instead of duplicating them.
- Durable repo-wide rules remain easy to find.

### Indexed doc families

- An index exists for the doc family.
- The index includes owner, status, freshness, and routing guidance.
- Leaf docs hold the detailed material.

### Code files

- The resulting files each have one stable responsibility.
- The exported interface remains obvious.
- Routine edits no longer require touching one oversized shared file.
