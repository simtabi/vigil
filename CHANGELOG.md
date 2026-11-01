# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Releases keep their notes.** `changelog: disable: true` makes current
  GoReleaser skip the pipe that reads `--release-notes`, which published
  ssh-manager v3.2.0 and v3.4.0 with empty bodies while the run reported
  success; vigil's next release would have gone the same way. The config no
  longer disables it, the release workflow sets the body from the tagged
  CHANGELOG section after GoReleaser as a backstop, and a test fails if either
  regresses.

### Documentation
- Added ASCII diagrams across the docs: control-plane/data-flow and desired-state
  precedence (`architecture.md`), engine+scope decision (`installation.md`),
  "how it works" (README), the Graph device-code + presence lifecycle
  (`tools/graph-engine.md`), per-OS input mechanism (`tools/input-engine.md`), the
  TUI navigation tree (`tui.md`), the self-update flow (`updating.md`), and the
  release pipeline (`release.md`).
- Fixed two doc inaccuracies: the `schedule remove` example is now 0-based
  (`remove 0`), and the Settings access note reflects the menu-driven TUI (run
  `vigil`, select Settings) instead of the removed `c` shortcut. Clarified that
  `vigil status` prints `Internet: connected`.

## [0.7.0] - 2026-06-25

### Added
- **Dynamic shell completion** (`vigil completion <bash|zsh|fish|powershell>`):
  `config get`/`set` complete config keys; `config set <key>` completes values
  (engine, input.method, booleans, log.level, and prefix-filtered timezones);
  `schedule remove` completes window indices; `--scope` completes user/system.

## [0.6.0] - 2026-06-25

### Added
- **Searchable timezone picker** in the TUI: the Settings timezone row and the
  setup wizard now open a prepopulated, filterable list of all IANA zones (plus
  `Local`/`UTC`) instead of a free-text field. Type to filter (`/` and `_` count
  as spaces, so `new york` finds `America/New_York`), `↑/↓` to move the scrolling
  list, `Enter` to select. Backed by a new `internal/tz` package + a reusable
  `picker` component (`bubbles/textinput` + a windowed list).

## [0.5.2] - 2026-06-25

### Changed
- **First-run onboarding is now a navigable menu** (↑/↓ + Enter), consistent with
  the rest of the TUI, replacing the old single-key (`w`/`i`) prompts: choose
  *Guided setup wizard*, *Write default config*, or *Quit*. After the wizard
  finishes, the main menu loads automatically.

## [0.5.1] - 2026-06-25

### Changed
- Banner headline is now "Vigil for MS Teams and Slack" (Slack and other tools
  are planned). The compact name shown in the TUI header stays "Vigil".

## [0.5.0] - 2026-06-25

### Added
- **Brand identity**: an eye mark (open eye + green "available" pupil) in
  `assets/vigil.svg`, rasterized to PNGs and embedded as the **Windows binary
  icon** + version info (`make icons` regenerates both from the SVG).
- **Banner**: `vigil version` now prints a masthead — name, tagline, version/
  commit/date, author, contact, and product/source URLs — colored on a terminal,
  plain ASCII when piped or under `NO_COLOR` (`--json` unchanged).
- **Network-connection indicator**: a live **green ● online / red ● offline** dot
  in the TUI status strip, an `Internet:` line in `vigil status` (`online` in
  `--json`), and a `network` check in `vigil doctor`. Detection uses the standard
  `generate_204` captive-portal probe with a TCP-dial fallback and a short
  timeout (best-effort, silent on failure).
- **Glyph icons** in the TUI menu and header (with the eye mark).

## [0.4.0] - 2026-06-25

### Changed
- **Renamed the project to `vigil`.** The binary is now `vigil` (was `mta`), the
  Go module is `github.com/simtabi/vigil`, the service identifier/display name are
  `vigil`/`Vigil`, and all config/runtime directories use `vigil`
  (`~/.config/vigil`, `~/.local/state/vigil`; `%AppData%\vigil` etc.). Release
  artifacts are now `vigil_<os>_<arch>`.

### Upgrade notes
- This is effectively a fresh install. From a `0.3.x` (`mta`) install: remove the
  old service first (`mta uninstall`, or `mta self uninstall --purge`), then
  install `vigil` and re-run `vigil config wizard` (config does not migrate from
  `ms-teams-activity` automatically). For the Graph engine, `vigil auth login`
  again. `mta upgrade` cannot cross the rename — install vigil fresh.

## [0.3.1] - 2026-06-24

### Fixed
- **Self-update never downgrades**: `mta upgrade` now only replaces the binary
  when the release is strictly newer, and reports "already up to date" instead of
  a false "updated" on a no-op.
- **Install-channel detection** no longer treats any path *containing* "scoop"
  (e.g. `…/scoopproject/…`) as Scoop-managed; it matches a real path segment, so
  standalone binaries in such paths can still self-update.
- **Windows sleep-prevention assertion**: the daemon goroutine is pinned to its
  OS thread, so the thread-affine `SetThreadExecutionState` is set and cleared on
  the same thread and never leaks after deactivation/shutdown.
- `self install`/`self update` no longer leave a partial `*.tmp` file behind when
  a copy fails.
- TUI Settings clamps **jitter below the interval**, so you can no longer set a
  value that fails validation on save.
- `doctor` now emits a warning (instead of silently dropping the row) when the
  macOS screensaver idle time can't be parsed, and doesn't compare against a
  zero interval when the config failed to load.

### Added
- TUI Override menu offers timed "Force inactive" presets (1h / 2h / 4h),
  matching the CLI's `off --for` and the existing "Force active" presets.
- `mta upgrade --check --json` now reports the install `channel` name (string)
  plus a `self_updatable` boolean.

### Documentation
- Documented `--purge` / `-Purge` usage for the uninstall scripts and the
  one-hour DST-changeover-day caveat for overnight schedule windows.

## [0.3.0] - 2026-06-24

### Added
- **Confirmation prompts** on destructive/state-changing commands (`uninstall`,
  `self uninstall`, `upgrade`, `schedule clear`, `auth logout`, and `config init`
  when the file exists), each with a safe default. New global flags `--yes/-y`
  (assume yes), `--no-input` (never prompt; use defaults), and `--no-color`.
- **Spinners** for long operations (`upgrade`, service install/uninstall) and a
  **download progress bar** (rate + ETA) in `scripts/install.sh` on a terminal.
- Consistent status output with `[OK]/[i]/[WARN]/[FAIL]` icons; color is disabled
  for `NO_COLOR`, `TERM=dumb`, `--no-color`, and non-terminals.
- `docs/cli.md` documenting CLI conventions (streams, flags, prompts, color,
  exit codes, env vars), linked from the README docs index.

### Changed
- Human status/notes now go to **stderr**; stdout is reserved for primary/`--json`
  output so piping stays clean.

### Fixed
- `--scope` now **rejects** invalid values instead of silently falling back to
  `user`.
- `on`/`off --for` rejects zero/negative durations (previously created an
  already-expired override).
- Confirmation prompts no longer block when stdin is not a terminal (used the
  default instead of hanging).

## [0.2.3] - 2026-06-24

### Added
- Turnkey daemon setup: `mta install --init` writes a default config first if
  none exists; the install scripts gain `--with-service` (`-WithService` on
  Windows) to configure + install + start the service in one step; new
  `scripts/uninstall.sh` / `scripts/uninstall.ps1` for a clean removal (service +
  binary, `--purge` also wipes config/data).

### Changed
- `scripts/build-all.sh` now always cleans `dist/` before building, so stale or
  renamed artifacts never linger.
- `start`/`stop`/`restart` give a clearer "is the service installed? run
  `mta install`" hint on failure; `install` is idempotent (restarts if the
  service already exists).

## [0.2.2] - 2026-06-24

### Added
- TUI Override menu offers timed "force active" presets (1h / 2h / 4h) alongside
  indefinite — parity with the CLI's `on --for`.
- `docs/tui.md` documenting the interactive menu UI (navigation, screens, editor
  keys), linked from the README docs index.

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

[Unreleased]: https://github.com/simtabi/vigil/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/simtabi/vigil/releases/tag/v0.7.0
[0.6.0]: https://github.com/simtabi/vigil/releases/tag/v0.6.0
[0.5.2]: https://github.com/simtabi/vigil/releases/tag/v0.5.2
[0.5.1]: https://github.com/simtabi/vigil/releases/tag/v0.5.1
[0.5.0]: https://github.com/simtabi/vigil/releases/tag/v0.5.0
[0.4.0]: https://github.com/simtabi/vigil/releases/tag/v0.4.0
[0.3.1]: https://github.com/simtabi/vigil/releases/tag/v0.3.1
[0.3.0]: https://github.com/simtabi/vigil/releases/tag/v0.3.0
[0.2.3]: https://github.com/simtabi/vigil/releases/tag/v0.2.3
[0.2.2]: https://github.com/simtabi/vigil/releases/tag/v0.2.2
[0.2.1]: https://github.com/simtabi/vigil/releases/tag/v0.2.1
[0.2.0]: https://github.com/simtabi/vigil/releases/tag/v0.2.0
[0.1.3]: https://github.com/simtabi/vigil/releases/tag/v0.1.3
[0.1.2]: https://github.com/simtabi/vigil/releases/tag/v0.1.2
[0.1.1]: https://github.com/simtabi/vigil/releases/tag/v0.1.1
[0.1.0]: https://github.com/simtabi/vigil/releases/tag/v0.1.0
