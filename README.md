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

See [docs/installation.md](docs/installation.md). In short: build from source
with Go ≥ 1.23 (`go build -o mta .`) or use a release binary, then
`mta install`. The input engine should be installed with `--scope user`
(default); a system-wide service (`--scope system`) is intended for the `graph`
engine.

## Documentation

| Page | What it covers |
|------|----------------|
| [installation.md](docs/installation.md) | Building, installing, scope, and the per-OS service mechanism |
| [configuration.md](docs/configuration.md) | The `config.json` schema, schedules, overrides, and file locations |
| [architecture.md](docs/architecture.md) | Engines, the daemon loop, control plane, and design decisions |
| [tools/input-engine.md](docs/tools/input-engine.md) | The synthetic-input engine and its per-OS behavior |
| [tools/graph-engine.md](docs/tools/graph-engine.md) | The Microsoft Graph engine, Entra app setup, and admin consent |
| [release.md](docs/release.md) | Tag-driven release process |

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
