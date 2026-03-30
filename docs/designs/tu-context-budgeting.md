# tu Context Budgeting

Status: Approved for implementation
Owner: `tu`
Last updated: 2026-03-26

## Why This Exists

`tu` helps humans and agents find the files that consume the most context in a repository without loading those files into the agent's working set. The first useful outcome is simple: point `tu` at a repo and quickly see which files should be split, trimmed, or treated carefully.

This document is the repo-local source of truth for implementation. It folds together the initial product design, the engineering review decisions, and the test/release requirements so coding can proceed without re-litigating scope.

## Problem Statement

Large repositories degrade agent performance because a small number of oversized files dominate context. `tu` should make that visible immediately for both:

- humans working in a terminal
- agents and scripts consuming structured output

The first user is a principal engineer operating in a very large legacy repository. v1 is about repo triage, not token-pricing analytics and not a general-purpose tokenizer platform.

## V1 Goal

Ship a boring, offline-first CLI that:

- scans a file or directory
- respects `.gitignore` by default
- ranks files by token usage
- emits human-readable output by default
- emits deterministic `--format json`, `--format plain`, and `--format csv` output for automation
- uses one exact local tokenizer path plus a heuristic fallback
- ships installable binaries via GitHub Releases

## Implementation Status

### Slice 1 Completed

Completed on branch `feat/cli-skeleton`.

Delivered:

- initialized the Go module
- added the `cmd/tu` entrypoint
- added the `internal/cli` package for option parsing and command execution
- implemented `--help` and `--version`
- implemented documented validation rules for path count, output-mode conflicts, and supported sort modes
- implemented documented exit-code behavior for success, invalid usage, and runtime-not-implemented cases
- added tests covering parsing, validation, help/version, and exit behavior

Current runtime behavior:

- `tu --help` works
- `tu --version` works
- invalid CLI usage returns exit code `2` with actionable stderr
- valid non-help invocations currently return exit code `1` with a clear "not implemented yet" runtime message

This is intentional. Slice 1 establishes the CLI contract before scan/count logic is added.

### Slice 2 Completed

Delivered:

- added `internal/report` with the versioned scan report, summary, result, status, and method types
- added `internal/format` with the JSON and plain-text formatters
- locked the initial JSON schema in code under `schema_version = "v1"`
- locked the plain-text row contract in code
- normalized empty JSON reports to emit `results: []`
- escaped delimiter-breaking characters in plain-text paths
- added golden tests for JSON and plain formatter output

Current runtime behavior after Slice 2:

- the CLI still does not scan files yet
- the output contracts are now implemented and test-backed before scan logic begins
- follow-up work should build scanning against these types instead of inventing output ad hoc

### Slice 3 Completed

Delivered:

- added `internal/scan` with file and directory targeting
- implemented recursive walking plus depth-based traversal controls
- implemented repo-aware `.gitignore` handling, including nested `.gitignore` files under the scan root
- classified skipped files as `binary`, `decode-failed`, `permission-denied`, or `unreadable`
- wired the CLI to emit real human, JSON, and plain outputs from populated `ScanReport` values
- added fixture-based scan coverage and CLI execution tests

Current runtime behavior after Slice 3:

- `tu` now scans files and directories successfully
- counted files currently use heuristic token estimates
- skipped files are preserved in machine output with explicit reasons
- default human output prints a triage table to stdout and a summary to stderr

### Slice 4 Completed

Delivered:

- added `internal/count` with a small exact-first counter seam
- integrated a local `cl100k_base` tokenizer for exact counts without runtime downloads
- kept heuristic counting as the fallback path when exact counting is unavailable
- added large-file threshold handling that records oversized files as `skipped` with reason `too-large`
- preserved the existing JSON and plain-text contracts while introducing exact-vs-heuristic provider labeling

Current runtime behavior after Slice 4:

- normal text files are counted exactly with `method=exact` and `provider=openai`
- heuristic counting remains available as a fallback path
- files above the configured max-file-size limit are skipped explicitly as `too-large`
- machine output remains stable while count metadata is now richer

### Slice 5 Completed

Delivered:

- replaced directory scan counting with a bounded worker pool fed by a single walk
- load nested `.gitignore` files lazily as that walk enters each directory instead of pre-scanning the tree
- kept deterministic output by collecting results concurrently and sorting only after the full scan completes
- added deterministic-order tests that run the same scan repeatedly under concurrent execution
- added benchmark coverage for the checked-in fixture repo and a larger synthetic repo shape

Current runtime behavior after Slice 5:

- file counting is now concurrent for directory scans but still bounded
- final JSON, plain, and human ordering remains deterministic
- benchmark targets exist for future threshold and worker-count tuning without changing the user-facing contract

### Slice 6 Completed

Delivered:

- added a push and pull-request GitHub Actions test workflow
- added a tagged GitHub Actions release workflow with a cross-build OS/arch matrix
- added a reusable binary-verification path that compares a built `tu` binary against a same-version reference build on canonical fixtures
- gated release publishing on both quality checks and binary verification before attaching assets
- added checksum generation for published release archives

Current runtime behavior after Slice 6:

- pushes and pull requests run `go test ./...` and `go vet ./...` in CI
- tags matching `v*` build release archives for `darwin`, `linux`, and `windows` across `amd64` and `arm64`
- release publishing now waits for native binary verification against the canonical fixture corpus on Linux, macOS, and Windows runners
- secondary cross-built arch archives are packaged in CI but are not executed on GitHub-hosted runners
- GitHub Release assets are published idempotently and can be updated safely on re-runs

### Next Slice

The planned v1 implementation is complete.

Do next:

- run pre-landing review on the current branch
- ship once the branch is clean
- start post-v1 work from [TODOS.md](/Users/rich/Projects/tu/TODOS.md) instead of expanding the v1 scope in place

Do not change the JSON or plain-text contract casually while building scan logic. Extend the scan pipeline to fit the existing report types and golden tests.

## Not In Scope

- remote provider counting in v1
- multi-provider plugin or registry architecture
- built-in agent-sensitive heuristics such as special treatment for `AGENTS.md`
- package-manager distribution in the first release
- interactive prompts, destructive commands, or config-file complexity

Deferred work is tracked in [TODOS.md](/Users/rich/Projects/tu/TODOS.md).

## Core Decisions

### Runtime

Use Go.

Why:

- single static binaries are straightforward
- good fit for filesystem walking and concurrent work queues
- easy GitHub Actions release story
- no runtime install story for end users

### Counter Abstraction

Keep one small seam, not a framework.

```text
                +------------------+
input file ---> | Counter          |
                |------------------|
                | Count(file)      |
                +---------+--------+
                          |
          +---------------+---------------+
          |                               |
          v                               v
+--------------------+        +--------------------+
| ExactLocalCounter  |        | HeuristicCounter   |
| local tokenizer    |        | chars / 4 fallback |
+--------------------+        +--------------------+
```

v1 has exactly two implementations:

- `ExactLocalCounter`
- `HeuristicCounter`

Do not build a provider registry, plugin loader, or remote adapter system in v1.

### File Handling

Files must never be silently omitted.

- Count normal text files.
- Record binary or unreadable files as skipped with explicit reasons.
- Keep skipped-file visibility in machine output.
- Show human-readable summary counts for skipped files.

### Performance Model

Use a single directory walk feeding a bounded worker pool.

```text
target path
    |
    v
+-----------+      +------------------+      +------------------+
| Walk files | ---> | classify file    | ---> | work queue       |
| once       |      | ignore/skip/read |      | bounded workers  |
+-----------+      +------------------+      +------------------+
                                                   |
                                                   v
                                      +--------------------------+
                                      | count + produce result   |
                                      +------------+-------------+
                                                   |
                                                   v
                                      +--------------------------+
                                      | sort + format output     |
                                      +--------------------------+
```

Rules:

- never spawn one goroutine per file
- never read the whole repo into memory
- keep ordering deterministic regardless of concurrency
- introduce a large-file threshold and avoid unbounded full-file reads above it

The large-file threshold should be configurable via a dedicated CLI flag and remain separate from the user-facing `--threshold` result filter, which only affects which counted rows are displayed.

## CLI Specification

### One-liner

`tu` shows token usage for files so humans and agents can identify context-heavy files quickly.

### Command Shape

No subcommands in v1.

Usage:

```text
tu [path] [--format <human|json|plain|csv>] [--file <path|-] [--sort <mode>] [--depth <n>] [--threshold <tokens>] [--exclude <glob>]... [--summarize] [--no-gitignore]
tu --help
tu --version
```

`path` defaults to `.`.

Accepted targets:

- file
- directory

Reject:

- missing path
- unsupported sort mode
- malformed `--exclude` glob
- invalid flag combinations

### Flags

| Flag | Type | Default | Meaning |
|------|------|---------|---------|
| `-h`, `--help` | bool | `false` | Show usage and exit |
| `--version` | bool | `false` | Print version and exit |
| `--format` | string | `human` | One of `human`, `json`, `plain`, `csv` |
| `--file` | string | `""` | Write primary output to a file path, or `-` for stdout |
| `--sort` | string | `tokens-desc` | One of `tokens-desc`, `tokens-asc`, `path-asc`, `path-desc` |
| `-d`, `--depth` | int | unset | Limit file results by relative depth; use `0` for a summary row and `1` for top-level files only |
| `-t`, `--threshold` | int64 | unset | Filter displayed rows by token count; negative values keep rows below the absolute value |
| `-I`, `--exclude` | string[] | unset | Ignore files and directories whose basename matches the glob; repeatable |
| `-s`, `--summarize` | bool | `false` | Alias for `--depth 0` |
| `--no-gitignore` | bool | `false` | Include files that `.gitignore` would exclude |
| `-q`, `--quiet` | bool | `false` | Suppress non-essential stderr messages |

Semantics:

- `--format` selects exactly one primary output mode.
- `--file` changes only the primary output destination.
- `--sort` applies to file and directory targets, even when the result set has one item.
- `.gitignore` applies only when the target is inside a repo with ignore rules available.
- `--depth` applies only to directory targets; file targets still emit one file result.
- `--threshold` filters displayed rows after the full scan summary is computed.
- positive `--threshold` values keep rows with `tokens > threshold`.
- `--threshold 0` keeps rows with `tokens > 0`.
- negative `--threshold` values keep rows with `tokens < abs(threshold)`.
- rows with no token count, including skipped files, never match a threshold filter.
- `--exclude` is a scan-time filter, not a display-time filter.
- `--exclude` matches file and directory basenames with shell-style glob semantics.
- malformed `--exclude` globs are invalid usage.
- excluded files and directories are never counted, never reported as skipped, and never included in summary totals.
- repeat `--exclude` / `-I` to apply multiple masks in the order provided.
- `--exclude` does not disable `.gitignore`; use `--no-gitignore` if you want ignored files included in the scan.
- `--summarize` is equivalent to `--depth 0`.
- v1 does not read from stdin.
- v1 does not support config files or custom environment-variable configuration.

### I/O Contract

stdout:

- primary result only
- human table by default
- full JSON document for `--format json`
- stable line-oriented records for `--format plain`
- headered CSV rows for `--format csv`

stderr:

- validation errors
- runtime errors
- optional warnings and summaries
- never required to parse the primary result

TTY behavior:

- no color support in v1

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | success |
| `1` | runtime failure while scanning or counting |
| `2` | invalid usage or validation failure |

Examples:

- missing path: `2`
- invalid `--sort`: `2`
- unsupported `--format`: `2`
- unreadable file encountered but scan completes with skipped result: `0`
- fatal walk failure with no usable result: `1`

### Determinism Rules

To keep tests and automation stable:

- all output paths are relative to the scan root
- machine-readable paths use forward slashes
- sort ties break by `path-asc`
- `--format json` output must not include volatile timestamps
- `--format plain` output must be stable across runs for the same inputs
- `--format csv` output must be stable across runs for the same inputs

## Scan Semantics

### Target Behavior

#### File target

- count exactly one file
- emit one result row/object
- preserve the same status/method schema as directory scans

#### Directory target

- walk the directory tree once
- recurse by default
- honor `.gitignore` by default
- include ignored files only with `--no-gitignore`

### Status Model

Each file result must have a status.

Allowed statuses:

- `counted`
- `skipped`

Required per-file fields:

- `path`
- `tokens`
- `method`
- `provider`
- `status`
- `reason`

Constraints:

- `tokens` may be `null` for skipped files
- `reason` is `null` for counted files
- `method` is one of `exact` or `heuristic` for counted files
- skipped files keep `status=skipped` and a non-null `reason`

Recommended skip reasons:

- `binary`
- `unreadable`
- `permission-denied`
- `too-large`
- `decode-failed`

### Large Files

Large files are a real use case, not an edge case.

v1 requirements:

- stat before reading
- compare against a configured large-file limit when one is set
- for files above the threshold, take a bounded-memory path or skip/approximate explicitly
- never hide that fallback from the user

The configured size-limit behavior cannot remain unspecified.

## Output Contract

### Human Output

Default output is a compact table designed for triage.

Suggested columns:

- `tokens`
- `method`
- `status`
- `path`

After the table, print a short stderr summary when relevant:

- files counted
- files skipped
- whether heuristic fallback was used

### Plain Output

`--format plain` is for shell composition. Keep it line-based and boring.

Suggested format:

```text
<tokens>\t<method>\t<status>\t<path>
```

Rules:

- one result per line
- no headers
- no color
- no human commentary
- the `path` field escapes backslash, tab, carriage return, and newline as `\\`, `\t`, `\r`, and `\n`

### JSON Output

`--format json` is the contract for agents and scripts.

Schema shape:

```json
{
  "schema_version": "v1",
  "target": ".",
  "root": ".",
  "recursive": true,
  "respect_gitignore": true,
  "sort": "tokens-desc",
  "threshold": 500,
  "exclude": ["node_modules", "*.min.js"],
  "summary": {
    "files_seen": 0,
    "files_counted": 0,
    "files_skipped": 0,
    "total_tokens": 0
  },
  "results": [
    {
      "path": "README.md",
      "tokens": 321,
      "method": "exact",
      "provider": "openai",
      "status": "counted",
      "reason": null
    },
    {
      "path": "assets/logo.png",
      "tokens": null,
      "method": null,
      "provider": null,
      "status": "skipped",
      "reason": "binary"
    }
  ]
}
```

Notes:

- `schema_version` starts at `v1`
- `root` should be the scan root as given to the formatter, not an absolute machine-specific path
- `threshold` is omitted when no display filter is applied
- `exclude` is omitted when no scan-time exclusions are applied

### CSV Output

`--format csv` is for spreadsheet imports and artifact export.

Suggested format:

```text
kind,path,tokens,method,provider,status,reason
file,README.md,321,exact,openai,counted,
file,assets/logo.png,,,,skipped,binary
```

Rules:

- always include a header row
- one result per row
- keep skipped files visible
- no summary rows or footers
- empty fields represent missing values

## Suggested Package Layout

Keep the first implementation small.

```text
cmd/tu/
internal/cli/
internal/scan/
internal/count/
internal/format/
testdata/
.github/workflows/
```

The point is separation by responsibility, not abstraction for abstraction's sake.

## Implementation Order

### Milestone 1: Skeleton

- initialize Go module
- add CLI parsing
- implement `--help` and `--version`
- define option validation rules

### Milestone 2: Contracts First

- define result structs
- define JSON schema
- define plain-text format
- write golden tests for formatter output before scan logic grows

### Milestone 3: Scanning

- implement file vs directory targeting
- implement recursive walk
- implement `.gitignore` handling
- implement skipped-file classification

### Milestone 4: Counting

- add exact local tokenizer path
- retain heuristic fallback once exact counting exists
- refine provider labeling for exact vs heuristic counts
- add large-file handling

### Milestone 5: Performance

- add bounded worker pool
- preserve deterministic sorting
- benchmark representative fixture repos

### Milestone 6: Release

- add GitHub Actions test workflow
- add tagged release workflow
- cross-build release binaries
- verify release binaries against fixture corpus before publishing

## Testing Plan

Use Go's standard `testing` package.

Test layers:

- unit tests for option validation, sort behavior, formatter behavior, and counter selection
- fixture-based scan tests under `testdata/`
- golden tests for `--format json`, `--format plain`, and `--format csv`
- subprocess integration tests for the CLI binary
- release-workflow verification against canonical fixtures

Fixture corpus must include:

- normal text files
- single-file target cases
- nested directories
- ignored files
- binary files
- unreadable files
- undecodable text
- empty directories
- large files

Every branch from the engineering review must map to tests, especially:

- invalid usage returning `2`
- runtime failure returning `1`
- skipped-file reporting
- deterministic output under concurrency
- release binary parity with source-built behavior

## Release Plan

Ship GitHub Release binaries in v1.

Minimum CI:

- run tests on pushes and pull requests
- build release artifacts on tags
- target `darwin`, `linux`, and `windows`
- target `amd64` and `arm64` where Go tooling supports it cleanly
- run a binary verification step against canonical fixtures before attaching release artifacts

Package-manager distribution comes later.

## Risks And Mitigations

### Risk: output contract drift

Mitigation:

- version the JSON schema
- add golden tests for `--format json`, `--format plain`, and `--format csv`

### Risk: memory blowups on large files

Mitigation:

- stat before read
- enforce a large-file strategy
- benchmark with large fixtures

### Risk: concurrency breaks determinism

Mitigation:

- separate counting from final sort
- use stable tie-breaking by path
- test large fixture repos repeatedly

### Risk: release binaries differ from source behavior

Mitigation:

- verify built binaries against the same fixture corpus used in source tests

## Future Work

Future work should start from [TODOS.md](/Users/rich/Projects/tu/TODOS.md), not by expanding v1 in place.

Likely next steps after a stable v1:

- package-manager installation
- additional provider adapters
- opt-in agent-sensitive heuristics once real usage validates them
