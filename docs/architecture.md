# Architecture

## Control plane at a glance

The front ends (CLI, TUI) never talk to the daemon directly — they coordinate
through two small JSON files in the runtime dir, written atomically and watched
with fsnotify. Only the daemon keeps long-lived OS state.

```
   ┌───────────┐                         ┌───────────┐
   │ vigil CLI │                         │ vigil TUI │      front ends
   └─────┬─────┘                         └─────┬─────┘
         │  write override.json (atomic temp + rename)
         └──────────────┬────────────────────┘
                        ▼
              ┌───────────────────┐   fsnotify (~1s)   ┌────────────────────────┐
              │   override.json   │ ─────────────────► │  internal/engine       │
              └───────────────────┘                    │  daemon loop           │
              ┌───────────────────┐  written each cycle│  (single goroutine,    │
              │   status.json     │ ◄───────────────── │   OS-thread pinned)    │
              └─────────┬─────────┘                    │                        │
                        │ read (status / TUI strip)    │  Desired =             │
                        └───────── front ends ◄────────┤   override ?? schedule │
                                                        └───────────┬────────────┘
                                  active: Tick ±jitter  ┌───────────┘
                       active→inactive / SIGTERM: Stop  │
                                                        ▼
                                  ┌─────────────────────────────────────┐
                                  │ activators (internal/activity)      │
                                  │   • input → resets the OS idle timer │
                                  │            (per-OS virtual HID)      │
                                  │   • graph → sets a sticky Teams      │
                                  │            "Available" via the API   │
                                  └─────────────────────────────────────┘
```

## Components

The Go module lives in `src/` (`github.com/simtabi/vigil/src`); everything the
repository presents stays at the root. Paths below are under `src/`.

```
cmd/vigil/      main entrypoint (the package directory names the `vigil` binary)
internal/cli/   cobra CLI (run, install, on/off/resume, status, config, schedule,
                auth, doctor, self/upgrade, tui)
tui/            bubbletea hub (dashboard, schedule & settings editors, wizard)
scripts/        build-all.sh (build/<os>/), extract-changelog.sh, install/uninstall
tools/iconz/    separate module: brand PNGs + Windows .syso from assets/vigil.svg
internal/
  selfupdate/   go-selfupdate wrapper + install-channel detection
  config/       versioned JSON schema, defaults, validation, OS paths
  schedule/     weekly-window + override evaluator (tz/DST/overnight aware)
  activity/     Activator interface + per-OS input backends + graph strategy
  graph/        MSAL device-code auth, token cache, presence HTTP calls
  control/      file control plane (status.json/override.json) + single-instance lock
  engine/       the daemon loop
  service/      kardianos install/control + Windows logon-task installer
  release/      tests only: release trigger, build/ layout, no container, no go install
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

Desired-state precedence (a manual override always wins over the schedule):

```
 read override.json
        │
        ├─ on                       ─────────────► DesiredActive = true
        ├─ off                      ─────────────► DesiredActive = false
        ├─ until T  ── T in future? ──┬─ yes ────► (the on/off it carries)
        │                             └─ no  ────► expired → fall through
        └─ none / expired           ─► schedule.Active(now) ? true : false
                                                  │
                              ┌───────── active? ─┴──────────┐
                            yes                              no
                              │                               │
                  Tick activators (interval ± jitter)   idle; on a true→false
                              │                          transition: Stop → revert
                              ▼                               ▼
                    OS idle reset / presence set        assertion released,
                                                        presence cleared
```

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
