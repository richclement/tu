# tu
CLI for measuring token usage across files and directories. `du` for tokenization.

## Usage

```sh
tu .
tu path/to/repo --depth 1
tu path/to/repo --summarize
tu path/to/repo --threshold 500
tu path/to/repo --threshold -200
tu path/to/repo --exclude node_modules --exclude '*.min.js'
tu path/to/repo --format json --quiet
tu path/to/file --format plain --quiet
tu path/to/repo --format csv --file report.csv --quiet
```

Default output is a human-readable table on stdout plus summaries or warnings on stderr. Use `--format json`, `--format plain`, or `--format csv` for automation-friendly output, and `--file` when you want any format written to a specific path instead of stdout. Use `--depth 1` to stay at the top level, `--summarize` to emit a single aggregate row for the target, `--threshold` to filter displayed rows by token count, `--exclude` / `-I` to skip matching basenames during the scan, and `--no-gitignore` to include files that ignore rules would normally skip.

Threshold filtering is strict: positive values keep only rows with token counts greater than the threshold, `0` keeps only rows above zero, and negative values keep only rows below the absolute value. Rows without token counts, such as skipped files, are omitted when threshold filtering is active. The filter affects displayed rows only; scan summaries still describe the full scan.

Exclude filtering is scan-time, not display-time: matching files and directories are skipped before counting, before skipped-file classification, and before summary totals are computed. Excluded paths do not appear in any output format, do not contribute to `files_seen`, `files_counted`, `files_skipped`, `heuristic_results`, or `total_tokens`, and are exposed in JSON output as top-level `exclude` metadata when set.

## Installation

### Homebrew

```sh
brew tap richclement/tap
brew install richclement/tap/tu
```

### GitHub Releases

Download the archive for your platform from the [GitHub Releases page](https://github.com/richclement/tu/releases).

## Development

```sh
make build
make fmt
make test
make vet
make ci
```

To verify a built binary locally against the canonical fixture corpus:

```sh
make verify-release
```

## Release

Tags matching `v*` trigger the GitHub Actions release workflow. It runs tests, verifies native binaries against the canonical fixture corpus on Linux, macOS, and Windows runners, cross-builds release archives for `darwin`, `linux`, and `windows` across `amd64` and `arm64`, then publishes or updates the matching GitHub Release.

After the release assets are live, the workflow parses `dist/release/checksums.txt`, renders the Homebrew formula update, and opens or updates a PR against `richclement/homebrew-tap`. That PR links back to the GitHub Release and is validated in the tap repo on both macOS and Linux before merge.

The release workflow expects a `HOMEBREW_TAP_TOKEN` secret with write access to `richclement/homebrew-tap` contents and pull requests. The secondary cross-built arch archives are packaged in CI but not executed on GitHub-hosted runners.

## Documentation

- [CHANGELOG.md](CHANGELOG.md)
- [docs/designs/tu-context-budgeting.md](docs/designs/tu-context-budgeting.md)
- [TODOS.md](TODOS.md)
