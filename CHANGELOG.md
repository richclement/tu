# Changelog

All notable changes to this project will be documented in this file.

## [0.1.1.1] - 2026-03-27

### Changed

- Pinned the GitHub Actions test and release workflows to immutable SHAs and upgraded `actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `actions/download-artifact`, and `peter-evans/create-pull-request` to Node 24 compatible releases.

## [0.1.0.0] - 2026-03-26

### Added

- You can now scan files and directories with `.gitignore` respected by default, recursive and non-recursive traversal, and deterministic human, `--json`, and `--plain` output modes.
- You can now get exact local `cl100k_base` token counts with heuristic fallback, explicit skipped-file reasons, large-file threshold handling, and deterministic result sorting.

### For Contributors

- Added fixture-backed unit, integration, benchmark, and release-verification coverage for the scan, count, formatting, and binary parity paths.
- Added GitHub Actions test and release workflows that run `go test ./...`, `go vet ./...`, verify native binaries on Linux, macOS, and Windows, cross-build release archives, and publish GitHub Releases with checksums.
- Added repo-local design and release documentation describing the shipped v1 scope, contracts, release flow, and deferred post-v1 work.
