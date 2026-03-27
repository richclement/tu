# tu
CLI for measuring token usage across files and directories. `du` for tokenization.

## Usage

```sh
tu .
tu path/to/repo --json --quiet
tu path/to/file --plain --quiet
```

Default output is a human-readable table on stdout plus summaries or warnings on stderr. Use `--json` or `--plain` for automation, `--non-recursive` to stay at the top level, and `--no-gitignore` to include files that ignore rules would normally skip.

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/tu
```

To verify a built binary locally against the canonical fixture corpus:

```sh
mkdir -p ./dist
go build -ldflags "-X main.version=dev" -o ./dist/tu ./cmd/tu
go run ./tools/releaseverify --binary ./dist/tu --version dev
```

## Release

Tags matching `v*` trigger the GitHub Actions release workflow. It runs tests, verifies native binaries against the canonical fixture corpus on Linux, macOS, and Windows runners, cross-builds release archives for `darwin`, `linux`, and `windows` across `amd64` and `arm64`, then publishes or updates the matching GitHub Release. The secondary cross-built arch archives are packaged in CI but not executed on GitHub-hosted runners.

## Documentation

- [CHANGELOG.md](CHANGELOG.md)
- [docs/designs/tu-context-budgeting.md](docs/designs/tu-context-budgeting.md)
- [TODOS.md](TODOS.md)
