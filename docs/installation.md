# Installation

## Install

vigil is built from source. Binary releases are on hold until the maintainer
decides otherwise, so this page offers no download, install-script,
package-manager or `go install` path. You need Go (the version `src/go.mod`
names), `git` and `make`.

```bash
git clone https://github.com/simtabi/vigil ~/src/vigil
make -C ~/src/vigil build                     # -> ~/src/vigil/build/<os>/vigil
~/src/vigil/build/macos/vigil self install    # Linux: build/linux/vigil; copies it to ~/.local/bin
vigil doctor
```

`make build` writes `build/macos/vigil` on a Mac and `build/linux/vigil` on
Linux. `install -m 0755 ~/src/vigil/build/<os>/vigil ~/.local/bin/vigil`, or any
directory on `PATH`, works in place of `vigil self install`. On Windows, clone the
same way and build with
`go -C $HOME\src\vigil\src build -o ..\build\windows\vigil.exe ./cmd/vigil` from
PowerShell (the Go module lives in `src/`), then run
`$HOME\src\vigil\build\windows\vigil.exe self install`.

> **macOS** needs a C toolchain (Xcode Command Line Tools) because the macOS
> input backend uses cgo. Windows and Linux are pure Go.

A version is a tag: check one out to build exactly that release.

```bash
git -C ~/src/vigil fetch --tags
git -C ~/src/vigil checkout v0.7.0
make -C ~/src/vigil build
```

### Updating

```bash
git -C ~/src/vigil pull --ff-only
make -C ~/src/vigil build
~/src/vigil/build/macos/vigil self install    # Linux: build/linux/vigil
vigil restart                    # if the background service is installed
```

Do not use `vigil upgrade` on a source build; see [updating.md](updating.md).
`vigil version` reports the version `git describe` gave the build, so a build
from a branch is still identifiable.

### Earlier binary releases

0.1.0 to 0.7.0 were published as archives on
[GitHub Releases](https://github.com/simtabi/vigil/releases). Those stay where
they are and an existing install keeps working, but no new release is published
while binary releases are on hold: build from source to get anything newer. The
install scripts in `src/scripts/` remain in the tree, but they download the latest
published release, so they cannot install anything newer than 0.7.0. For
reference, those archives are named `vigil_<os>_<arch>` (macOS uses the
`macos` token):

| Machine | Archive |
|---------|---------|
| macOS — any (Apple Silicon **or** Intel) | `vigil_macos_universal.tar.gz` |
| macOS — Apple Silicon only | `vigil_macos_arm64.tar.gz` |
| macOS — Intel only | `vigil_macos_amd64.tar.gz` |
| Windows 64-bit / 32-bit / ARM | `vigil_windows_amd64.zip` / `_386` / `_arm64` |
| Linux 64-bit / 32-bit / ARM64 / ARMv7 | `vigil_linux_amd64.tar.gz` / `_386` / `_arm64` / `_armv7` |

(Also: `riscv64`, `ppc64le`, `s390x`, and FreeBSD/OpenBSD/NetBSD.)

## Building all targets yourself

`make cross` (or `./src/scripts/build-all.sh [version]`) builds ready-to-run
binaries for every target in `build/targets.txt` into `build/`, one folder per OS
(`macos`, never `darwin`). `make dist` does the same and adds the archives (and,
with `VIGIL_PACKAGES=1` and `nfpm` on `PATH`, deb/rpm) under `build/archives/`:

```
build/
  checksums.txt   # {os}/{file}, so: cd build && shasum -a 256 -c checksums.txt
  macos/    vigil_macos_arm64  vigil_macos_amd64  vigil_macos_universal
  linux/    vigil_linux_amd64  vigil_linux_arm64  vigil_linux_armv7  …
  windows/  vigil_windows_amd64.exe  …
  archives/ vigil_<os>_<arch>.tar.gz · vigil_windows_<arch>.zip · *.deb · *.rpm · checksums.txt
```

It builds whatever your local toolchain supports (macOS targets need a C
compiler; everything else is pure-Go cross-compilation). CI runs the same script
on every pull request so a reviewer can try the exact build; those are
`0.0.0-dev` builds for review, not a distribution channel.

## Putting the binary on PATH

`vigil self install` copies the running binary to a standard location
(`~/.local/bin`, `/usr/local/bin` with `--scope system`, or
`%LOCALAPPDATA%\Programs\vigil` on Windows). `vigil self uninstall [--purge]`
removes the service and the binary (and, with `--purge`, config + data).

## First run

```bash
vigil config wizard   # guided setup (or: vigil config init for defaults)
vigil doctor          # verify capabilities and permissions
vigil install         # install + start the background service
vigil status          # check service + daemon state
```

## Daemon lifecycle

The background service is managed with these commands (they auto-select the
right per-OS mechanism — launchd / systemd / Windows service / Windows logon
task):

```bash
vigil install --init   # turnkey: write a default config if missing, then install + start
vigil start | stop | restart | status
vigil uninstall        # stop + remove the service (keeps the binary)
```

`vigil install --init` does the whole thing in one shot. To remove everything
later:

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/src/scripts/uninstall.sh | sh
curl -fsSL .../src/scripts/uninstall.sh | sh -s -- --purge   # also delete config + runtime data

# Windows (piped form can't pass flags)
irm https://raw.githubusercontent.com/simtabi/vigil/main/src/scripts/uninstall.ps1 | iex
./src/scripts/uninstall.ps1 -Purge                            # local file: also delete config + data

# Or directly, on any OS:
vigil self uninstall            # service + binary
vigil self uninstall --purge    # also config + runtime data
```

`--purge` (and `-Purge`) additionally delete the config and runtime/state
directories; without it, those are left in place. The scripts moved from
`scripts/` to `src/scripts/`; the old `main/scripts/…` URLs, which earlier
installers printed, still work and forward to the new ones.

## Choosing engine + scope

```
 which engine? ─┬─ input ─► must run in YOUR unlocked GUI session → --scope user
                │            macOS: LaunchAgent · Linux: systemd --user
                │            Windows: logon Scheduled Task
                │            (vigil install refuses input + --scope system)
                │
                └─ graph ─► headless OK → --scope user OR system
                             needs an admin-consented Presence.ReadWrite app
```

## Scope: user vs system

`--scope user` (default) installs a per-user service that runs in your desktop
session. `--scope system` installs a machine-wide service.

**The input engine requires a desktop (GUI) session**, so it must be installed
with `--scope user`. `vigil install` refuses `input` + `--scope system`. A
system-wide service is appropriate for the `graph` engine, which is headless.

## Per-OS service mechanism

| OS | `input` engine | `graph` engine |
|----|----------------|----------------|
| macOS | LaunchAgent (user) | LaunchAgent (user) or LaunchDaemon (system) |
| Linux | `systemd --user` (run `loginctl enable-linger $USER` to persist when logged out) | systemd user or system |
| Windows | **logon Scheduled Task** (interactive session) | Windows service |

## Platform prerequisites

- **macOS** — grant the `vigil` binary **Accessibility** permission
  (System Settings → Privacy & Security → Accessibility). Because TCC keys on
  the binary's signature, re-grant after rebuilding an unsigned binary. Run
  `vigil doctor` to confirm. Synthetic input cannot reset the *hardware* idle
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
