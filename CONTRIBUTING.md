# Contributing

Thanks for your interest in improving `ms-teams-activity`.

## Development

Requirements: Go ≥ 1.23. The macOS build uses cgo (CoreGraphics/IOKit), so a C
toolchain is needed there; Windows and Linux builds are pure Go.

```bash
go build ./...            # build everything (current OS)
go test ./...             # run tests
go vet ./...              # static checks
gofmt -l .                # formatting (should print nothing)
golangci-lint run ./...   # linters (config in .golangci.yml)
```

Cross-platform compile check (no cgo needed for these targets):

```bash
CGO_ENABLED=0 GOOS=windows go build ./...
CGO_ENABLED=0 GOOS=linux   go build ./...
```

## Conventions

- Keep changes small and focused; match the surrounding style.
- Public packages and exported symbols carry doc comments.
- New behavior comes with table-driven tests where practical (see
  `internal/schedule` and `internal/config`).
- CLI output follows [docs/cli.md](docs/cli.md): route human status/prompts
  through `internal/cli/ui` (stderr, TTY/color-aware), keep primary and `--json`
  output on stdout, confirm destructive actions with `ui.Confirm`, and validate
  input early.
- Commit subjects are imperative and ≤ 72 chars; bodies explain *why*.

## Architecture

See [docs/architecture.md](docs/architecture.md) for the engine/daemon/control
design and the platform reliability rationale.

## Reporting issues

Use the GitHub issue templates. For security reports, follow
[SECURITY.md](SECURITY.md) instead of opening a public issue.
