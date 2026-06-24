# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-06-24

### Changed
- Redesigned the TUI around a **navigable main menu** (↑/↓ or j/k, Enter to
  select, Esc to go back) with Status / Override / Schedule / Settings / Service /
  Account / Check-for-updates / Help screens and a live status strip — far more
  discoverable than the old single-key dashboard. Override, Service, and Account
  are now submenus.

### Added
- Comprehensive deterministic TUI model tests (menu navigation + clamping, screen
  transitions, Esc-back, override file writes, schedule/settings editors incl. the
  `graph`-without-`client_id` validation path, onboarding, and view smoke for
  every screen).

## [0.2.0] - 2026-06-24

### Changed
- **macOS artifacts use the friendly `macos` token** instead of Go's `darwin`
  (e.g. `mta_macos_universal.tar.gz`, `mta_macos_arm64`); self-update sets the
  updater OS to `macos` to match. The Go build still targets `GOOS=darwin`.
- **User files moved to XDG locations** (macOS + Linux): config →
  `~/.config/ms-teams-activity/` (`$XDG_CONFIG_HOME`), runtime/state/token →
  `~/.local/state/ms-teams-activity/` (`$XDG_STATE_HOME`); no longer under
  `~/Library`. Windows uses `%AppData%` / `%LocalAppData%`.

### Upgrade notes
- From v0.1.x: **reinstall** (old binaries look for `darwin` assets and the old
  config path). Re-run `mta config init` (or `mta config wizard`); for the Graph
  engine, `mta auth login` again (token cache path changed).

## [0.1.3] - 2026-06-24

### Changed
- Build layout settled: a clean `dist/` root of bare ready-to-run binaries plus a
  `dist/archives/` folder grouping all archives + deb/rpm (each with its own
  `checksums.txt`). Release assets are the `dist/archives/` contents. Documented
  as the canonical Go build/distribution layout for all Simtabi projects.
- Hardened the macOS universal (`lipo`) step with a temp-file + atomic rename.

### Changed
- `dist/` is now fully flat: bare ready-to-run binaries and their archives live
  side by side (no `dist/archives/` subfolder), with one `checksums.txt` over
  everything. Release assets now include both the bare binaries and the archives.

## [0.1.1] - 2026-06-24

### Changed
- Flattened the build output: `dist/` now holds self-describing, ready-to-run
  binaries (`mta_<os>_<arch>[.ext]`, plus `mta_darwin_universal`) with archives,
  deb/rpm, and `checksums.txt` grouped under `dist/archives/`. Archive inner
  binaries keep the flat name (self-update-compatible). Documented as the
  canonical build/distribution layout for all Simtabi Go projects.

## [0.1.0] - 2026-06-24

### Added
- Initial release.
- Prebuilt release binaries for macOS/Windows/Linux with `checksums.txt`, plus
  Homebrew tap, Scoop bucket, deb/rpm packages, `go install`, and download
  install scripts (`scripts/install.sh`, `scripts/install.ps1`).
- `mta upgrade` / `mta self update` self-update (checksum-verified, package-manager
  aware), and `mta self install` / `mta self uninstall [--purge]`.
- Configure from the CLI: `config get/set/keys/wizard` and `schedule list/add/
  remove/clear`.
- Full TUI hub: first-run onboarding, settings editor, schedule editor,
  service/auth/update actions, and an update-available banner.
- Natural, non-repetitive input: randomized 1..`input.move_pixels` offset on a
  random axis plus jittered timing.
- `mta run --dry-run` (log intended actions only) and global `--verbose` logging.
- `doctor` performs a live Graph presence read to verify token + admin consent;
  Graph `availability`/`activity` values are validated.
- Engine-loop unit tests and a `golangci-lint` config + CI lint job.
- Broad release matrix (single-sourced in `build/targets.txt`): Linux
  (amd64/386/arm64/armv7/armv6/riscv64/ppc64le/s390x), Windows
  (amd64/386/arm64), macOS (Apple Silicon, Intel, and a **universal** binary),
  and FreeBSD/OpenBSD/NetBSD — built and bundled by a reusable workflow on every
  tag, with `make dist` for local builds and a CI `snapshot` job that always
  publishes binaries as run artifacts. macOS builds (both arches + universal via
  lipo) run on a single Apple-Silicon runner.

### Changed
- Entry point moved to `./cmd/mta` so `go install …/cmd/mta@latest` produces a
  binary named `mta`; the cobra package moved to `internal/cli`.
- Two pluggable engines: synthetic `input` (default) and Microsoft `graph`
  preferred presence; `both` runs them together.
- Per-OS input backends: `SendInput` (Windows), `CGEventPost` + power assertion
  (macOS), `/dev/uinput` (Linux).
- Configurable weekly schedule (timezone-aware, overnight windows) plus at-will
  `on`/`off`/`resume` overrides with optional `--for`/`--until` expiry.
- JSON configuration with versioning, validation, and atomic writes.
- Cross-platform service install via launchd/systemd/Windows service, with a
  Windows logon Scheduled Task for the input engine.
- Cobra CLI (`run`, `install`, `on`/`off`/`resume`, `status`, `config`, `auth`,
  `doctor`, `version`) and a Bubble Tea TUI dashboard.
- `doctor` diagnostics for permissions, capabilities, and configuration.

[Unreleased]: https://github.com/simtabi/ms-teams-activity/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/simtabi/ms-teams-activity/releases/tag/v0.2.1
[0.2.0]: https://github.com/simtabi/ms-teams-activity/releases/tag/v0.2.0
[0.1.3]: https://github.com/simtabi/ms-teams-activity/releases/tag/v0.1.3
[0.1.2]: https://github.com/simtabi/ms-teams-activity/releases/tag/v0.1.2
[0.1.1]: https://github.com/simtabi/ms-teams-activity/releases/tag/v0.1.1
[0.1.0]: https://github.com/simtabi/ms-teams-activity/releases/tag/v0.1.0
