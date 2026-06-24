# Installation

## Build from source

Requires Go ≥ 1.23. On macOS a C toolchain is needed (cgo); Windows and Linux
are pure Go.

```bash
git clone https://github.com/simtabi/ms-teams-activity
cd ms-teams-activity
go build -o mta .
# optionally put it on your PATH, e.g.
sudo install -m 0755 mta /usr/local/bin/mta     # macOS/Linux
```

Or use the helper scripts in `scripts/` (`install.sh`, `install.ps1`).

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
