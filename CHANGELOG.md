# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0.0] - 2026-03-26

### Added

- Added the initial `tu` Go CLI with file and directory scanning, recursive and non-recursive traversal, `.gitignore` support, and deterministic human, `--json`, and `--plain` output modes.
- Added exact local `cl100k_base` token counting with heuristic fallback, explicit skipped-file reasons, large-file threshold handling, and deterministic result sorting.
- Added fixture-backed unit, integration, benchmark, and release-verification coverage for the scan, count, formatting, and binary parity paths.
- Added GitHub Actions test and release workflows that run `go test ./...`, `go vet ./...`, verify native binaries on Linux, macOS, and Windows, cross-build release archives, and publish GitHub Releases with checksums.
- Added repo-local design and release documentation describing the shipped v1 scope, contracts, release flow, and deferred post-v1 work.
