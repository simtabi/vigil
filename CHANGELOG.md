# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release.
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
