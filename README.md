<p align="center">
  <img src="assets/icons/vigil_128.png" width="96" height="96" alt="vigil" />
</p>

# vigil

Keep Microsoft Teams showing as **Available** on a configurable work schedule —
or at will. Cross-platform (macOS, Windows, Linux), JSON-configured, with a CLI
and an interactive TUI, installable per-user or system-wide.

```bash
vigil config init        # write a default config (Mon–Fri 08:00–17:00)
vigil doctor             # check platform capabilities & permissions
vigil install            # install + start the background service
vigil on --for 2h        # at-will override; or `vigil off`, `vigil resume`
vigil                    # open the TUI dashboard
```

## How it works

Teams marks you **Away** after ~5 minutes of OS idle and immediately when the
screen locks. `vigil` runs a small background service that, during your active
schedule, keeps the session non-idle with one of two **engines**:

- **`input` (default)** — injects tiny, periodic, human-like input that resets
  the OS idle timer. No accounts or admin needed.
- **`graph`** — sets a sticky *preferred presence* via the Microsoft Graph API.
  Cleaner, survives lock, but requires an Entra app and **admin-consented**
  `Presence.ReadWrite`.

`both` runs them together. See the platform reliability notes below.

```
   Teams: Available ──(~5 min OS idle, or screen lock)──► Away

   vigil, during your schedule, keeps you Available:

     input engine ─► periodic human-like HID activity resets the OS idle timer
                     Linux/Windows: reliable · macOS: keeps Teams green but
                     can't beat a hardware lock
     graph engine ─► sets a sticky "Available" via Microsoft Graph
                     works through locks · needs admin-consented Presence.ReadWrite
     both         ─► run them together

   manual override (on / off / for 2h / until 17:00) always wins over the schedule
```

### Platform reliability

| OS | Mechanism | Reliability | Notes |
|----|-----------|-------------|-------|
| **Linux** | `/dev/uinput` virtual device (real kernel events) | Highest | Needs `uinput` access; Teams on Linux is web/PWA. |
| **Windows** | `SendInput` (real small move / F15) | High | Input engine installs as a **logon Scheduled Task** (a session-0 service can't inject). |
| **macOS** | `CGEventPost` + power assertion | Good* | *Synthetic events can't reset the **hardware** idle timer, so a forced screen-lock still marks you Away. Keep Available by disabling/lengthening auto-lock, or use the `graph` engine. Requires Accessibility permission. |

> The synthetic-input engine only works while a desktop session is **unlocked**.
> It cannot defeat a manual or policy-forced lock — that is what the `graph`
> engine is for.

## Install

Pick whichever fits (full details in [docs/installation.md](docs/installation.md)):

```bash
# Script (macOS/Linux) — downloads + checksum-verifies the prebuilt binary
curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/scripts/install.sh | sh
# …or in one shot, also configure + install + start the background daemon:
curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/scripts/install.sh | sh -s -- --with-service

# Homebrew (macOS/Linux)
brew install simtabi/tap/vigil

# Scoop (Windows, PowerShell)
scoop bucket add simtabi https://github.com/simtabi/scoop-bucket; scoop install vigil

# Debian/Ubuntu or RHEL — grab the .deb/.rpm from the latest release, then:
sudo dpkg -i vigil_*_amd64.deb        # or: sudo rpm -i vigil_*_amd64.rpm

# Go toolchain (Windows/Linux; macOS needs a C compiler for cgo)
go install github.com/simtabi/vigil/cmd/vigil@latest
```

Or download a prebuilt archive from the [releases page](https://github.com/simtabi/vigil/releases),
or build from source (`go build -o vigil ./cmd/vigil`). Then run `vigil config wizard`
(or `vigil config init`) and `vigil install`.

**Which download?** (assets are named `vigil_<os>_<arch>` — macOS uses the friendly
`macos` token; self-update matches automatically):

| Your machine | Asset |
|--------------|-------|
| macOS — any (Apple Silicon **or** Intel) | `vigil_macos_universal.tar.gz` |
| macOS — Apple Silicon only | `vigil_macos_arm64.tar.gz` |
| macOS — Intel only | `vigil_macos_amd64.tar.gz` |
| Windows 64-bit / 32-bit / ARM | `vigil_windows_amd64.zip` / `_386` / `_arm64` |
| Linux 64-bit / 32-bit / ARM64 / ARMv7 | `vigil_linux_amd64.tar.gz` / `_386` / `_arm64` / `_armv7` |

(Also: `riscv64`, `ppc64le`, `s390x`, and FreeBSD/OpenBSD/NetBSD.)

The **input** engine must be installed with `--scope user` (default, GUI
session); a system-wide service (`--scope system`) is intended for the `graph`
engine.

## Quick start guide and usage

### Getting started

1. Write a config: `vigil config wizard` (guided: engine, tz, schedule, graph) or
   `vigil config init` (defaults, Mon–Fri 08:00–17:00).
2. Check the platform: `vigil doctor` reports capabilities and permissions.
3. Install and start the background service: `vigil install`
   (`vigil install --init` writes a default config first if none exists).

The `graph` engine also needs `graph.client_id` set before `engine` can be
switched to `graph` or `both`; see [tools/graph-engine.md](docs/tools/graph-engine.md).

### Usage

```bash
vigil on --for 2h        # at-will override; or `vigil off`, `vigil resume`
vigil status             # service + daemon state
vigil                    # open the TUI dashboard
```

```bash
vigil schedule add --days Mon,Tue,Wed,Thu,Fri --start 08:00 --end 17:00
vigil config set engine both
```

The full configuration walkthrough is in [docs/configuration.md](docs/configuration.md); service control is under [The daemon](#the-daemon), and everything else is in the [documentation table](#documentation).

## Updating

```bash
vigil upgrade            # self-update to the latest release (verifies checksum)
vigil upgrade --check    # just report whether an update is available
```

`vigil upgrade` only manages standalone installs; if you installed via Homebrew,
Scoop, or a system package it tells you to use that manager instead. See
[docs/updating.md](docs/updating.md).

## The daemon

```bash
vigil install --init     # turnkey: write a default config if needed, install + start the service
vigil status             # service + daemon state
vigil restart            # restart after config changes (config is also hot-reloaded)
vigil stop / vigil start
vigil uninstall          # stop + remove the service (keeps the binary)
```

Remove everything (service + binary, `--purge` also deletes config/data):

```bash
curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/scripts/uninstall.sh | sh
# or: vigil self uninstall --purge
```

## Documentation

| Page | What it covers |
|------|----------------|
| [installation.md](docs/installation.md) | Building, installing, scope, and the per-OS service mechanism |
| [configuration.md](docs/configuration.md) | The `config.json` schema, schedules, overrides, and file locations |
| [tui.md](docs/tui.md) | The interactive menu UI — navigation, screens, and editor keys |
| [cli.md](docs/cli.md) | CLI conventions — flags, prompts, color, exit codes, env vars |
| [brand.md](docs/brand.md) | The eye mark, brand assets, banner, and regenerating icons |
| [architecture.md](docs/architecture.md) | Engines, the daemon loop, control plane, and design decisions |
| [tools/input-engine.md](docs/tools/input-engine.md) | The synthetic-input engine and its per-OS behavior |
| [tools/graph-engine.md](docs/tools/graph-engine.md) | The Microsoft Graph engine, Entra app setup, and admin consent |
| [updating.md](docs/updating.md) | Self-update (`vigil upgrade`), package managers, and caveats |
| [release.md](docs/release.md) | Tag-driven release & packaging process |

All actions are available from both the CLI and the interactive TUI (`vigil` on a
terminal): live status, overrides, schedule & settings editors, service control,
auth, and updates. Configure non-interactively with `vigil config set <key> <value>`
and `vigil schedule add/list/remove`.

## Responsible use

`vigil` is a personal-productivity tool. The input engine is dual-use
(equivalent to a "mouse jiggler"/Caffeine). **Respect your employer's policies
and applicable agreements** — presence is often used by colleagues to gauge
availability. Don't use this to misrepresent working hours where that would
breach policy. You are responsible for how you use it.

## License

MIT © 2026 Simtabi LLC. See [LICENSE](LICENSE).

- Product: <https://opensource.simtabi.com/products/vigil>
- Docs: <https://opensource.simtabi.com/documentation/vigil>
- Issues: <https://github.com/simtabi/vigil/issues>
