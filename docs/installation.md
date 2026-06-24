# Installation

## Options

| Method | Command |
|--------|---------|
| Install script (macOS/Linux) | `curl -fsSL https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/install.sh \| sh` |
| Install script (Windows) | `irm https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/install.ps1 \| iex` |
| Homebrew | `brew install simtabi/tap/mta` |
| Scoop | `scoop bucket add simtabi https://github.com/simtabi/scoop-bucket; scoop install mta` |
| deb/rpm | download from releases, `sudo dpkg -i mta_*.deb` / `sudo rpm -i mta_*.rpm` |
| go install | `go install github.com/simtabi/ms-teams-activity/cmd/mta@latest` |
| Prebuilt archive | download `mta_<os>_<arch>.{tar.gz,zip}` from the releases page |
| From source | `go build -o mta ./cmd/mta` |

The install scripts download the prebuilt binary and **verify its SHA-256**
against the release `checksums.txt`, falling back to a source build if the
download fails and Go is present.

> **go install / source on macOS** needs a C toolchain (Xcode CLT) because the
> macOS input backend uses cgo. Windows and Linux are pure Go.

## Building all targets yourself

`make dist` (or `./scripts/build-all.sh [version]`) builds ready-to-run binaries
for every target in `build/targets.txt`. The output is **flat** — each binary is
a self-describing file you can run directly — with compressed archives + deb/rpm
grouped under `dist/archives/`:

```
dist/
  mta_darwin_arm64   mta_linux_amd64   mta_windows_amd64.exe   mta_darwin_universal   …
  checksums.txt
  archives/  mta_<os>_<arch>.tar.gz · mta_windows_<arch>.zip · *.deb · *.rpm · checksums.txt
```

It builds whatever your local toolchain supports (macOS targets need a C
compiler; everything else is pure-Go cross-compilation). CI runs the same script
on every push to `main`, so prebuilt binaries are always downloadable from the
latest run's Artifacts.

## Putting the binary on PATH

`mta self install` copies the running binary to a standard location
(`~/.local/bin`, `/usr/local/bin` with `--scope system`, or
`%LOCALAPPDATA%\Programs\mta` on Windows). `mta self uninstall [--purge]`
removes the service and the binary (and, with `--purge`, config + data).

## First run

```bash
mta config init     # writes the default config for the chosen --scope
mta doctor          # verify capabilities and permissions
mta install         # install + start the background service
mta status          # check service + daemon state
```

## Scope: user vs system

`--scope user` (default) installs a per-user service that runs in your desktop
session. `--scope system` installs a machine-wide service.

**The input engine requires a desktop (GUI) session**, so it must be installed
with `--scope user`. `mta install` refuses `input` + `--scope system`. A
system-wide service is appropriate for the `graph` engine, which is headless.

## Per-OS service mechanism

| OS | `input` engine | `graph` engine |
|----|----------------|----------------|
| macOS | LaunchAgent (user) | LaunchAgent (user) or LaunchDaemon (system) |
| Linux | `systemd --user` (run `loginctl enable-linger $USER` to persist when logged out) | systemd user or system |
| Windows | **logon Scheduled Task** (interactive session) | Windows service |

## Platform prerequisites

- **macOS** — grant the `mta` binary **Accessibility** permission
  (System Settings → Privacy & Security → Accessibility). Because TCC keys on
  the binary's signature, re-grant after rebuilding an unsigned binary. Run
  `mta doctor` to confirm. Synthetic input cannot reset the *hardware* idle
  timer, so disable or lengthen auto-lock to stay Available.
- **Linux** — `/dev/uinput` must exist (`sudo modprobe uinput`) and be writable
  by your user. Add a udev rule / group so the device is accessible without
  root, e.g.:

  ```
  KERNEL=="uinput", GROUP="input", MODE="0660", OPTIONS+="static_node=uinput"
  ```

  then add yourself to the `input` group and re-login.
- **Windows** — no special setup; the input engine installs as a logon task.

[← Docs index](../README.md#documentation)
