# CLI conventions

`vigil` follows common command-line conventions so it behaves predictably in
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
vigil uninstall --yes              # no prompt; remove the service
vigil self uninstall --no-input    # safe: declines (default is no), so nothing is removed
```

## Color

Color and unicode icons are used only on an interactive terminal. They are
disabled when any of these hold: `--no-color`, `NO_COLOR` is set, `TERM=dumb`, or
stdout is not a TTY. (ASCII tags like `[OK]` are used in place of icons.)

## Progress

Long-running operations (`upgrade`, service `install`/`uninstall`) show a spinner
on a terminal and a plain "…" line otherwise. The install **script**
(`src/scripts/install.sh`) shows a download progress bar with transfer rate and ETA
when run interactively.

## Input validation

Invalid input is rejected early with a clear message rather than acted on:

- `--scope` must be `user` or `system`.
- `on`/`off --for` must be a **positive** Go duration (`30m`, `2h`, `1h30m`); you
  can't combine `--for` with `--until`.
- `config set` re-validates the whole config before writing (e.g. `engine=graph`
  with an empty `graph.client_id` fails).
- `schedule add` checks day names and `HH:MM` times.

## Network status

The TUI status strip shows a live connectivity dot — **green ● online** /
**red ● offline**. `vigil status` reports it as an `Internet: connected`/`offline`
line (and `"online": true/false` under `--json`), and `vigil doctor` has a
`network` check. It matters for the Graph engine and self-update, which need a
connection; the input engine does not.

Detection follows the standard captive-portal method: a tiny HTTP request to a
well-known `generate_204` endpoint (Cloudflare, then Google) with a short timeout,
falling back to a raw TCP dial. It is best-effort and silent on failure. The only
outbound traffic is that small probe, made to detect connectivity.

## Shell completion

`vigil completion <bash|zsh|fish|powershell>` prints a completion script; run
`vigil completion <shell> --help` for install instructions. Completion is
**dynamic** where it helps:

- `config get`/`config set` complete the dotted config keys;
- `config set <key>` completes values — `engine` (input/graph/both),
  `input.method` (mouse/key/zen), booleans (true/false), `log.level`, and
  **timezones** (prefix-filtered from the full IANA list);
- `schedule remove` completes the current window indices (annotated with their
  days/times);
- `--scope` completes `user`/`system`.

```bash
# zsh, current shell
source <(vigil completion zsh)
# bash, persistent
vigil completion bash | sudo tee /etc/bash_completion.d/vigil >/dev/null
```

## Honored environment variables

`NO_COLOR`, `TERM`, `EDITOR` (used by `config edit`), and
`XDG_CONFIG_HOME` / `XDG_STATE_HOME` (config and runtime/state locations on
macOS/Linux). Secrets are never read from flags or environment variables.

[← Docs index](../README.md#documentation)
