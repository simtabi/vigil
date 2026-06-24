# CLI conventions

`mta` follows common command-line conventions so it behaves predictably in
scripts, pipelines, and CI as well as interactively.

## Output streams

- **stdout** carries a command's primary result — the data you'd pipe or capture
  (`config get`, `config show`, `config keys`, `schedule list`, `status`, `version`,
  the `auth login` device-code instructions, and anything under `--json`).
- **stderr** carries status lines, progress, prompts, warnings, and errors
  (`[OK]` / `[i]` / `[WARN]` / `[FAIL]`). Redirecting stdout keeps these visible.

Every command returns **exit code 0** on success and **non-zero** on failure.

## Global flags

| Flag | Meaning |
|------|---------|
| `--scope user\|system` | Which config/runtime scope to act on (default `user`). Invalid values are rejected. |
| `--config <path>` | Use an explicit `config.json` instead of the scope default. |
| `--json` | Emit machine-readable JSON where supported. |
| `--verbose` | Debug-level logging. |
| `-y`, `--yes` | Assume "yes" for every confirmation prompt. |
| `--no-input` | Never prompt; use the safe default for each prompt (for scripts/CI). |
| `--no-color` | Disable colored output and icons. |

## Confirmations

Destructive or state-changing commands ask before acting, with a **safe default**
(usually "no"). These prompt: `uninstall`, `self uninstall`, `upgrade`,
`schedule clear`, `auth logout`, and `config init` when the file already exists.

- Bypass non-interactively with `--yes` (proceed) or `--no-input` (take the
  default, i.e. *don't* do the destructive thing).
- When stdin is **not a terminal**, prompts are skipped and the default is used —
  commands never hang waiting for input in a pipeline.

```bash
mta uninstall --yes              # no prompt; remove the service
mta self uninstall --no-input    # safe: declines (default is no), so nothing is removed
```

## Color

Color and unicode icons are used only on an interactive terminal. They are
disabled when any of these hold: `--no-color`, `NO_COLOR` is set, `TERM=dumb`, or
stdout is not a TTY. (ASCII tags like `[OK]` are used in place of icons.)

## Progress

Long-running operations (`upgrade`, service `install`/`uninstall`) show a spinner
on a terminal and a plain "…" line otherwise. The install **script**
(`scripts/install.sh`) shows a download progress bar with transfer rate and ETA
when run interactively.

## Input validation

Invalid input is rejected early with a clear message rather than acted on:

- `--scope` must be `user` or `system`.
- `on`/`off --for` must be a **positive** Go duration (`30m`, `2h`, `1h30m`); you
  can't combine `--for` with `--until`.
- `config set` re-validates the whole config before writing (e.g. `engine=graph`
  with an empty `graph.client_id` fails).
- `schedule add` checks day names and `HH:MM` times.

## Honored environment variables

`NO_COLOR`, `TERM`, `EDITOR` (used by `config edit`), and
`XDG_CONFIG_HOME` / `XDG_STATE_HOME` (config and runtime/state locations on
macOS/Linux). Secrets are never read from flags or environment variables.

[← Docs index](../README.md#documentation)
