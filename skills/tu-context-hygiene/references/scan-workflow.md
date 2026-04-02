# Scan Workflow

Use `tu` to rank files before you read them. The goal is to discover hotspots without loading them into context prematurely.

## Default Command Set

Prefer these two commands first:

```sh
tu . --depth 2 --threshold 300
tu . --format json --threshold 800
```

Use the shallow scan to spot root-level hazards such as oversized guidance docs or root readmes. Use the JSON scan when you need repo-wide ranking, path classification, or comparison across multiple large files.

## Progressive Scan Strategy

### 1. Root triage

Start shallow:

```sh
tu . --depth 2 --threshold 300
```

Look for:

- `AGENTS.md`, `CLAUDE.md`, and root `README.md`
- `docs/architecture`, `docs/designs`, `docs/implementation`, `docs/runbooks`
- oversized top-level code or test files

### 2. Repo-wide ranking

Switch to JSON when you need the full shape:

```sh
tu . --format json --threshold 800
```

Use JSON to answer:

- Which files dominate the repo?
- Are the largest files docs, code, or tests?
- Are the hotspots clustered in one subtree?
- Should you rescan a subtree with tighter thresholds?

### 3. Targeted subtree rescan

If one directory dominates, narrow the scope:

```sh
tu docs/designs --threshold 400
tu internal/scan --format json --threshold 1200
```

Use subtree scans to avoid treating one large local hotspot as a repo-wide pattern.

## Exclude Noise Early

Exclude files that should not drive the remediation conversation:

```sh
tu . --format json --threshold 800 --exclude node_modules --exclude vendor --exclude '*.golden'
```

Common low-value candidates:

- vendored dependencies
- build artifacts
- generated code
- snapshot files
- fixtures that are intentionally bulky

Do not recommend refactoring these first unless the developer explicitly wants that class of file addressed.

## Reading Discipline After The Scan

After `tu` identifies a hotspot:

1. Classify the file before opening it.
2. Read the map of the file first.
3. Open only targeted sections.
4. Stop early if the problem is already obvious.

Large files are evidence, not an instruction to load everything.

## Threshold Guidance

Treat thresholds as triage levers, not universal truth:

- `300-600`: good for root docs and shallow triage
- `800-1500`: good for repo-wide hotspot ranking
- `2000+`: good for isolating severe outliers

Prefer class-specific budgets from `file-class-rubric.md` over one global threshold.

## When To Escalate

Escalate from report-only guidance to a proposed restructuring plan when:

- the same file blocks multiple tasks
- the file mixes several unrelated responsibilities
- ownership and freshness are unclear
- the blast radius is high for routine edits
- the developer asks for a concrete remediation plan
