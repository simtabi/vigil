# Architecture

## Components

```
cmd/            cobra CLI (run, install, on/off/resume, status, config, auth, doctor, tui)
tui/            bubbletea dashboard (live status, overrides, log tail)
internal/
  config/       versioned JSON schema, defaults, validation, OS paths
  schedule/     weekly-window + override evaluator (tz/DST/overnight aware)
  activity/     Activator interface + per-OS input backends + graph strategy
  graph/        MSAL device-code auth, token cache, presence HTTP calls
  control/      file control plane (status.json/override.json) + single-instance lock
  engine/       the daemon loop
  service/      kardianos install/control + Windows logon-task installer
```

## The daemon loop (`internal/engine`)

On each cycle the engine:

1. Loads the override file and evaluates `DesiredActive = override ?? schedule`.
2. If active, calls `Tick` on every configured activator at `interval ± jitter`.
3. On an active→inactive transition, calls `Stop` on each activator to revert
   externally-visible state (release the macOS power assertion, clear the Graph
   preferred presence).
4. Publishes `status.json`.

Config and override files are watched with fsnotify for ~1s hot-reload; an
inactive engine still polls every 30s to catch schedule transitions. On
SIGTERM/service stop, all activators are `Stop`-ped (graceful revert).

## Engines (`internal/activity`)

`Activator` is the strategy interface (`Tick`, `Stop`, `Name`). Backends are
build-tagged so each binary contains only its platform's code:

- `input_windows.go` — `SendInput` (pure Go); optional `SetThreadExecutionState`.
- `input_darwin.go` — cgo `CGEventPost` + `IOPMAssertion`.
- `input_linux.go` — `/dev/uinput` virtual device (real kernel events).
- `graph.go` — `setUserPreferredPresence` with refresh, `clear…` on stop.

## Why these choices

- **Synthetic input over "prevent sleep".** Teams idles on the OS idle timer;
  power assertions alone don't reset it. Real input does. (macOS is the
  exception — see below.)
- **uinput on Linux** injects *real* kernel events, the most reliable way to
  reset idle under both X11 and Wayland. robotgo was rejected: on macOS it uses
  the same CGEvent path (no extra reliability) and adds heavy deps.
- **Windows logon task for input.** A session-0 Windows service cannot inject
  into the interactive desktop, so the input engine runs as a per-user logon
  Scheduled Task instead.
- **macOS honesty.** Synthetic `CGEventPost` events do not reset the *hardware*
  `HIDIdleTime` (the same limitation as Screen Sharing input). Teams generally
  reads the synthetic-aware combined-session idle, so events keep it green, but
  the screensaver/auto-lock uses the hardware timer — so a forced lock still
  marks you Away. We hold a display-sleep assertion to defer auto-lock, and the
  `graph` engine is the answer where a lock policy is enforced.
- **File-based control plane.** No network listener by default; the CLI/TUI and
  daemon coordinate through atomic file writes + fsnotify, which is simple and
  cross-platform. A single-instance lock prevents two daemons per runtime dir.

[← Docs index](../README.md#documentation)
