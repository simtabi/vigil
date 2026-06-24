# ms-teams-activity (`mta`)

Keep Microsoft Teams showing as **Available** on a configurable work schedule —
or at will. Cross-platform (macOS, Windows, Linux), JSON-configured, with a CLI
and an interactive TUI, installable per-user or system-wide.

```bash
mta config init        # write a default config (Mon–Fri 08:00–17:00)
mta doctor             # check platform capabilities & permissions
mta install            # install + start the background service
mta on --for 2h        # at-will override; or `mta off`, `mta resume`
mta                    # open the TUI dashboard
```

## How it works

Teams marks you **Away** after ~5 minutes of OS idle and immediately when the
screen locks. `mta` runs a small background service that, during your active
schedule, keeps the session non-idle with one of two **engines**:

- **`input` (default)** — injects tiny, periodic, human-like input that resets
  the OS idle timer. No accounts or admin needed.
- **`graph`** — sets a sticky *preferred presence* via the Microsoft Graph API.
  Cleaner, survives lock, but requires an Entra app and **admin-consented**
  `Presence.ReadWrite`.

`both` runs them together. See the platform reliability notes below.

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
curl -fsSL https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/install.sh | sh
# …or in one shot, also configure + install + start the background daemon:
curl -fsSL https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/install.sh | sh -s -- --with-service

# Homebrew (macOS/Linux)
brew install simtabi/tap/mta

# Scoop (Windows, PowerShell)
scoop bucket add simtabi https://github.com/simtabi/scoop-bucket; scoop install mta

# Debian/Ubuntu or RHEL — grab the .deb/.rpm from the latest release, then:
sudo dpkg -i mta_*_amd64.deb        # or: sudo rpm -i mta_*_amd64.rpm

# Go toolchain (Windows/Linux; macOS needs a C compiler for cgo)
go install github.com/simtabi/ms-teams-activity/cmd/mta@latest
```

Or download a prebuilt archive from the [releases page](https://github.com/simtabi/ms-teams-activity/releases),
or build from source (`go build -o mta ./cmd/mta`). Then run `mta config wizard`
(or `mta config init`) and `mta install`.

**Which download?** (assets are named `mta_<os>_<arch>` — macOS uses the friendly
`macos` token; self-update matches automatically):

| Your machine | Asset |
|--------------|-------|
| macOS — any (Apple Silicon **or** Intel) | `mta_macos_universal.tar.gz` |
| macOS — Apple Silicon only | `mta_macos_arm64.tar.gz` |
| macOS — Intel only | `mta_macos_amd64.tar.gz` |
| Windows 64-bit / 32-bit / ARM | `mta_windows_amd64.zip` / `_386` / `_arm64` |
| Linux 64-bit / 32-bit / ARM64 / ARMv7 | `mta_linux_amd64.tar.gz` / `_386` / `_arm64` / `_armv7` |

(Also: `riscv64`, `ppc64le`, `s390x`, and FreeBSD/OpenBSD/NetBSD.)

The **input** engine must be installed with `--scope user` (default, GUI
session); a system-wide service (`--scope system`) is intended for the `graph`
engine.

## Updating

```bash
mta upgrade            # self-update to the latest release (verifies checksum)
mta upgrade --check    # just report whether an update is available
```

`mta upgrade` only manages standalone installs; if you installed via Homebrew,
Scoop, or a system package it tells you to use that manager instead. See
[docs/updating.md](docs/updating.md).

## The daemon

```bash
mta install --init     # turnkey: write a default config if needed, install + start the service
mta status             # service + daemon state
mta restart            # restart after config changes (config is also hot-reloaded)
mta stop / mta start
mta uninstall          # stop + remove the service (keeps the binary)
```

Remove everything (service + binary, `--purge` also deletes config/data):

```bash
curl -fsSL https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/uninstall.sh | sh
# or: mta self uninstall --purge
```

## Documentation

| Page | What it covers |
|------|----------------|
| [installation.md](docs/installation.md) | Building, installing, scope, and the per-OS service mechanism |
| [configuration.md](docs/configuration.md) | The `config.json` schema, schedules, overrides, and file locations |
| [tui.md](docs/tui.md) | The interactive menu UI — navigation, screens, and editor keys |
| [cli.md](docs/cli.md) | CLI conventions — flags, prompts, color, exit codes, env vars |
| [architecture.md](docs/architecture.md) | Engines, the daemon loop, control plane, and design decisions |
| [tools/input-engine.md](docs/tools/input-engine.md) | The synthetic-input engine and its per-OS behavior |
| [tools/graph-engine.md](docs/tools/graph-engine.md) | The Microsoft Graph engine, Entra app setup, and admin consent |
| [updating.md](docs/updating.md) | Self-update (`mta upgrade`), package managers, and caveats |
| [release.md](docs/release.md) | Tag-driven release & packaging process |

All actions are available from both the CLI and the interactive TUI (`mta` on a
terminal): live status, overrides, schedule & settings editors, service control,
auth, and updates. Configure non-interactively with `mta config set <key> <value>`
and `mta schedule add/list/remove`.

## Responsible use

`mta` is a personal-productivity tool. The input engine is dual-use
(equivalent to a "mouse jiggler"/Caffeine). **Respect your employer's policies
and applicable agreements** — presence is often used by colleagues to gauge
availability. Don't use this to misrepresent working hours where that would
breach policy. You are responsible for how you use it.

## License

MIT © 2026 Simtabi LLC. See [LICENSE](LICENSE).

- Product: <https://opensource.simtabi.com/products/ms-teams-activity>
- Docs: <https://opensource.simtabi.com/documentation/ms-teams-activity>
- Issues: <https://github.com/simtabi/ms-teams-activity/issues>
