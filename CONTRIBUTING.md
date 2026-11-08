# Contributing

Where this file is silent, the [Simtabi contributing guide](https://github.com/simtabi/.github/blob/HEAD/CONTRIBUTING.md) applies.

Thanks for your interest in improving `vigil`.

## Development

Requirements: Go ≥ 1.25. The macOS build uses cgo (CoreGraphics/IOKit), so a C
toolchain is needed there; Windows and Linux builds are pure Go.

The Go module lives in `src/` (`github.com/simtabi/vigil/src`), so every `go`
command names it with `-C src`. A bare `go test ./...` at the repository root
finds no packages and passes, which proves nothing.

```bash
go -C src build ./...     # build everything (current OS)
go -C src test ./...      # run tests (make test adds -race)
go -C src vet ./...       # static checks
gofmt -l src              # formatting (should print nothing)
make lint                 # golangci-lint, config in src/.golangci.yml
make build                # host binary -> build/<os>/vigil (macos, linux, windows)
```

Cross-platform builds go into `build/`, one folder per OS, and nothing else:

```bash
make cross                                  # every target in build/targets.txt -> build/<os>/
(cd build && shasum -a 256 -c checksums.txt)
make cross-check                            # compile-only check of the cgo-free targets
```

`make check-build` fails when `build/` holds anything but the OS folders, its
inputs, `checksums.txt` and `archives/`; every `make build`/`cross`/`dist` runs it.

## Conventions

- Keep changes small and focused; match the surrounding style.
- Public packages and exported symbols carry doc comments.
- New behavior comes with table-driven tests where practical (see
  `src/internal/schedule` and `src/internal/config`).
- CLI output follows [docs/cli.md](docs/cli.md): route human status/prompts
  through `src/internal/cli/ui` (stderr, TTY/color-aware), keep primary and `--json`
  output on stdout, confirm destructive actions with `ui.Confirm`, and validate
  input early.
- Commit subjects are imperative and ≤ 72 chars; bodies explain *why*.

## Architecture

See [docs/architecture.md](docs/architecture.md) for the engine/daemon/control
design and the platform reliability rationale.

## Reporting issues

Use the GitHub issue templates. For security reports, follow
[SECURITY.md](SECURITY.md) instead of opening a public issue.
