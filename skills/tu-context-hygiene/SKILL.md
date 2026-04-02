---
name: tu-context-hygiene
description: Interpret `tu` scan results to identify context-heavy files, inspect them progressively, and recommend indexing or refactor patterns that improve agent readability. Use when working with `tu`, token budgets, oversized `AGENTS.md` or `CLAUDE.md`, large architecture/design/implementation docs, or large code and test files that should be split or made easier for an agent to navigate.
---

# TU Context Hygiene

## Overview

Use this skill to turn `tu` output into an agent-readable remediation plan. Find the largest files first, classify them before reading deeply, inspect only the parts that matter, and recommend structural changes that reduce context waste without defaulting to edits.

Keep the default posture advisory. Report findings first, ask before proposing concrete edits, and never read a large candidate file in full on the first pass unless the developer explicitly asks.

## Quick Start

1. Run a shallow scan to identify root-level hotspots.
2. Run a repo-wide JSON scan when you need to rank, classify, or compare multiple files.
3. Group the largest files by class before opening them.
4. Inspect large files progressively, not monolithically.
5. Produce the standard report contract from `references/report-template.md`.

See `references/scan-workflow.md` for command recipes.

## Operating Rules

- Prefer `tu` from `PATH`. If you are working inside the `tu` source tree itself, `go run ./cmd/tu ...` is an acceptable fallback.
- Stay report-only by default. Ask before proposing edits or file moves.
- Do not load a large file fully on first pass. Read the map before the territory.
- Classify generated, vendored, fixture-heavy, or build-output files early so you do not spend time "fixing" low-value noise.
- Use repo ranking plus file-class budgets. Do not treat one global token ceiling as universally correct.
- Recommend splits by responsibility or question-to-answer, not by arbitrary line count or section count.

## Workflow

### 1. Scan First

Run the two-pass workflow from `references/scan-workflow.md`:

- Shallow/root triage: `tu . --depth 2 --threshold 300`
- Repo-wide machine-readable triage: `tu . --format json --threshold 800`

Rescan a subtree when one area is clearly hot. Use scan-time excludes when fixtures, generated output, or vendored content dominate the ranking.

### 2. Classify Before Reading

Sort large files into one of these classes:

- root guidance docs
- indexed project docs
- leaf markdown docs
- code files
- test files
- generated, vendor, or fixture files

Use `references/file-class-rubric.md` for budgets, warning levels, and trust checks.

### 3. Inspect Progressively

For markdown:

- read the title, metadata block, first screen, and heading list first
- search for `Owner`, `Status`, `Last updated`, `Source of truth`, and `Audience`
- open only the sections that correspond to the actual question

For code:

- read imports, top-level types, top-level functions or methods, and section boundaries first
- search for large literals, giant switches, mixed responsibilities, or unrelated entrypoints
- open the specific ranges that appear to carry the most responsibility

Read `references/doc-patterns.md` for markdown restructuring and `references/code-patterns.md` for code and test-file seams.

### 4. Score The File

Judge each large file on:

- token size relative to its file class
- navigability
- ownership
- freshness
- canonicality
- cohesion
- blast radius

Use the reusable rubric in `references/file-class-rubric.md`. Large size alone is not sufficient; a file can still be acceptable if it is intentionally reference-oriented and easy to trust.

### 5. Recommend The Right Pattern

Use the smallest restructuring pattern that fixes the real problem:

- Root guidance docs: keep them as a map, not the whole corpus.
- Architecture/design/implementation docs: introduce or tighten an index that explains what each leaf doc is for.
- Leaf markdown docs: split by question someone is trying to answer.
- Code files: extract stable responsibilities, not cosmetic fragments.
- Test files: split by behavior area, fixture family, or subsystem.

## Reporting Contract

Always produce the same headings in the final report:

- `Top Findings`
- `Why This File Is Hard For Agents`
- `What To Inspect Next`
- `Recommended Refactor Pattern`
- `Questions For The Developer`
- `Suggested Acceptance Criteria`

Use `references/report-template.md` as the exact shape.

## Calibration

Check your recommendations against these baseline outcomes:

- A small `AGENTS.md` plus a 5k-token design doc should usually produce "keep the root doc lean, split or index the design doc."
- A 2k-token `AGENTS.md` full of duplicated guidance should usually produce "rewrite as a map plus leaf docs."
- A repo dominated by fixtures or generated output should usually produce "exclude or deprioritize the noise before recommending refactors."
- A 4k-token code file that mixes parsing, validation, and formatting should usually produce a seam-based extraction recommendation.
- A large doc with clear owner, status, freshness, and index metadata may be acceptable even when it is above the default target.

## References

- `references/scan-workflow.md`: command recipes, thresholds, and rescan strategy
- `references/file-class-rubric.md`: budgets, trust checks, and severity rules
- `references/doc-patterns.md`: indexing, metadata, and progressive disclosure patterns
- `references/code-patterns.md`: code and test-file split patterns
- `references/report-template.md`: standard developer-facing output contract
