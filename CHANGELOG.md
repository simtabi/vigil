# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/simtabi/ms-teams-activity/commits/main
