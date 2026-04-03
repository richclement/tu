# tu

CLI for measuring token usage across files and directories. `du` for tokenization.

## Usage

```sh
tu .
tu -P path/to/directory
tu -H path/to/symlinked-directory
tu -L path/to/directory
tu path/to/directory --depth 1
tu path/to/directory --summarize
tu path/to/directory --threshold 500
tu path/to/directory --threshold -200
tu path/to/directory --max-file-size 1000000
tu path/to/directory --max-file-size 1MiB
tu path/to/directory --max-file-size 1.5MiB
tu path/to/directory --max-file-size "1.5 MB"
tu path/to/directory --exclude node_modules --exclude '*.min.js'
tu path/to/directory --format json --quiet
tu path/to/file --format plain --quiet
tu path/to/directory --format csv --file report.csv --quiet
```

Default output is a human-readable table on stdout plus summaries or warnings on stderr. Use `--format json`, `--format plain`, or `--format csv` for automation-friendly output, and `--file` when you want any format written to a specific path instead of stdout. Use `--depth 1` to stay at the top level, `--summarize` to emit a single aggregate row for the target, `--threshold` to filter displayed rows by token count, `--max-file-size` to skip files larger than a configured size before reading them, `--exclude` / `-I` to skip matching basenames during the scan, `--no-gitignore` to include files that ignore rules would normally skip, and `-P` / `-H` / `-L` to control symlink traversal.

Symlink handling mirrors `du`: `-P` is the default and does not follow symlinks, `-H` follows symlinks given on the command line only, and `-L` follows symlinks on the command line and during traversal. Non-followed symlinks are reported as skipped with reason `symlink`. Followed broken symlinks are reported as `broken-symlink`, and `-L` reports ancestor cycles as `symlink-cycle` instead of recursing forever. JSON output includes the active mode in top-level `symlink_mode` metadata.

Threshold filtering is strict: positive values keep only rows with token counts greater than the threshold, `0` keeps only rows above zero, and negative values keep only rows below the absolute value. Rows without token counts, such as skipped files, are omitted when threshold filtering is active. The filter affects displayed rows only; scan summaries still describe the full scan.

File-size limiting is opt-in: by default, `tu` does not enforce a maximum file size. Raw `--max-file-size` values without a unit are bytes and must be integers. Unit-bearing values may be integers or decimals. `KB` / `MB` / `GB` / `TB` / `PB` use powers of 1000, while `KiB` / `MiB` / `GiB` / `TiB` / `PiB` use powers of 1024. Decimal values round down to whole bytes, and files larger than the configured limit are skipped with reason `too-large`.

Exclude filtering is scan-time, not display-time: matching files and directories are skipped before counting, before skipped-file classification, and before summary totals are computed. Excluded paths do not appear in any output format, do not contribute to `files_seen`, `files_counted`, `files_skipped`, `heuristic_results`, or `total_tokens`, and are exposed in JSON output as top-level `exclude` metadata when set.

Quote patterns that contain shell metacharacters so your shell does not expand them before `tu` sees them. For example, prefer `--exclude '*.min.js'` over `--exclude *.min.js`.

Malformed exclude globs are rejected as usage errors.

`--exclude` does not disable `.gitignore` processing. If you want ignored files to participate in the scan at all, use `--no-gitignore`.

## Installation

### Homebrew

```sh
brew tap richclement/tap
brew install richclement/tap/tu
```

### GitHub Releases

Download the archive for your platform from the [GitHub Releases page](https://github.com/richclement/tu/releases).

## Agent Skill

```sh
npx skills add richclement/tu
```

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

Use this checklist when cutting a release:

1. Confirm the latest published tag with `git tag --sort=version:refname`.
2. Keep `VERSION` on the next unreleased target version and make sure it matches the version you plan to tag.
3. Move the entries under `## [Unreleased]` in `CHANGELOG.md` into a dated section for that version.
4. Run `make ci` and `make verify-release`.
5. Commit the release-prep changes, create the matching `vMAJOR.MINOR.PATCH` tag, and push the branch plus tag.
6. Watch the GitHub Actions release workflow, then verify the GitHub Release assets and the Homebrew tap pull request it opens.

## Documentation

- [CHANGELOG.md](CHANGELOG.md)
- [docs/designs/tu-context-budgeting.md](docs/designs/tu-context-budgeting.md)
- [skills/tu-context-hygiene/SKILL.md](skills/tu-context-hygiene/SKILL.md)
- [TODOS.md](TODOS.md)
