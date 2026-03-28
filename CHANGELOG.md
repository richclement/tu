# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Added `--format csv` output with deterministic headered rows and `--file` support for writing reports to disk.

### Changed

- Replaced the old `--json` and `--plain` flags with a canonical `--format <human|json|plain|csv>` interface.
- Made `--file <path|->` a generic output destination override for all formats, including human output.
- Updated release verification, documentation, and formatter coverage to match the new output contract.
- Removed per-file byte-size fields and `summary.total_bytes` from human, `--json`, and `--plain` output so the tool stays focused on token budgeting.

### Removed

- Removed the legacy `--json`, `--plain`, and dead `--no-color` flags from the CLI surface.

## [0.1.1] - 2026-03-27

### Changed

- Pinned the GitHub Actions test and release workflows to immutable SHAs and upgraded `actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `actions/download-artifact`, and `peter-evans/create-pull-request` to Node 24 compatible releases.

## [0.1.0] - 2026-03-26

### Added

- You can now scan files and directories with `.gitignore` respected by default, recursive and non-recursive traversal, and deterministic human, `--json`, and `--plain` output modes.
- You can now get exact local `cl100k_base` token counts with heuristic fallback, explicit skipped-file reasons, large-file threshold handling, and deterministic result sorting.

### For Contributors

- Added fixture-backed unit, integration, benchmark, and release-verification coverage for the scan, count, formatting, and binary parity paths.
- Added GitHub Actions test and release workflows that run `go test ./...`, `go vet ./...`, verify native binaries on Linux, macOS, and Windows, cross-build release archives, and publish GitHub Releases with checksums.
- Added repo-local design and release documentation describing the shipped v1 scope, contracts, release flow, and deferred post-v1 work.
