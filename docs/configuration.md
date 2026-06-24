# Configuration

Configuration is a single JSON file. Manage it with:

```bash
mta config path        # print the effective path
mta config init        # write defaults
mta config edit        # open in $EDITOR, validated on save
mta config validate    # check without editing
mta config show        # print the effective config
```

## File locations

| Scope | Config | Runtime (status/override/log/token) |
|-------|--------|-------------------------------------|
| user | `os.UserConfigDir()/ms-teams-activity/config.json` | `os.UserCacheDir()/ms-teams-activity/` |
| system (Linux) | `/etc/ms-teams-activity/config.json` | `/etc/ms-teams-activity/` |
| system (macOS) | `/Library/Application Support/ms-teams-activity/` | same |
| system (Windows) | `C:\ProgramData\ms-teams-activity\` | same |

> Multi-user note: with `--scope user`, runtime files live under your own cache
> dir so the CLI can always write the override file the daemon watches. For a
> `--scope system` graph daemon, runtime files are root-owned; control commands
> then require matching privilege (or enable the optional loopback API via
> `control.port`).

## Schema

```jsonc
{
  "version": 1,
  "engine": "input",                 // "input" | "graph" | "both"
  "timezone": "America/New_York",    // IANA tz; "Local" uses the host zone
  "schedule": {
    "enabled": true,                 // false => only manual overrides apply
    "always": false,                 // true => active whenever the daemon runs
    "windows": [
      { "days": ["Mon","Tue","Wed","Thu","Fri"], "start": "08:00", "end": "17:00" }
    ]
  },
  "input": {
    "interval_seconds": 60,          // pulse cadence; must be in [5,300)
    "jitter_seconds": 25,            // +/- randomization for natural cadence
    "method": "mouse",              // "mouse" (real small move) | "key" (F15) | "zen"
    "prevent_sleep": true            // hold a sleep/display assertion while active
  },
  "graph": {
    "tenant_id": "common",           // "common" | "organizations" | tenant GUID
    "client_id": "",                 // your Entra public-client app id
    "availability": "Available",
    "activity": "Available",
    "expiration": "PT8H",            // ISO-8601 duration; re-asserted on refresh
    "refresh_minutes": 60
  },
  "control": { "port": 0 },          // 0 = file-based control plane
  "log": { "level": "info", "max_size_mb": 5, "max_backups": 3 }
}
```

### Schedules

- Times are `HH:MM` 24-hour in `timezone`. Days are `Mon`..`Sun`.
- A window whose `end` is **not after** `start` is treated as **overnight**
  (e.g. `22:00`–`06:00`): the morning segment belongs to the day after a listed
  start day.
- Multiple windows are OR-combined. DST transitions are handled by evaluating in
  the configured zone.

### Overrides ("at will")

Overrides take precedence over the schedule and persist across restarts:

```bash
mta on                 # force active, indefinitely
mta on --for 2h30m     # force active for a duration
mta off --until 18:00  # force inactive until the next 18:00
mta resume             # clear the override, follow the schedule again
```

The config is hot-reloaded; edits and overrides take effect within ~1 second
without restarting the service.

[← Docs index](../README.md#documentation)
