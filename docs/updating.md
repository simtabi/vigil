# Updating

> Binaries ship on the [GitHub release](https://github.com/simtabi/vigil/releases)
> only. A release install updates with `vigil upgrade` (below), which reads the
> latest GitHub release. GitHub lists no vigil release yet (checked 2026-11-13),
> so until the first one is published, update a source build from git.

## From source

```bash
git -C ~/src/vigil pull --ff-only
make -C ~/src/vigil build        # -> ~/src/vigil/build/<os>/vigil
~/src/vigil/build/macos/vigil self install   # replace the binary on PATH (Linux: build/linux/vigil)
vigil restart                    # if the background service is installed
```

The clone path is whatever you used in [installation](installation.md#install).
On macOS, re-grant Accessibility after the rebuild (see Caveats).

**`vigil upgrade` never replaces a source build with an older release.**
`make build` stamps the version from `git describe`, so a build between tags
reports something like `v0.7.0-3-gabc1234` (with `-dirty` for uncommitted
changes). Plain semver would sort that *below* `0.7.0`; vigil counts it as newer
than its base tag instead, so `vigil upgrade` refuses to install the 0.7.0
release over it and the TUI shows no "update available" banner. A build of
exactly a tag (`v0.7.0`) is reported as up to date, and a plain `go build`
without the Makefile reports `dev` and is refused. A release tagged *after* your
base tag (`v0.8.0` over `v0.7.0-3-gabc1234`) is still offered as an update.

## Self-update (standalone installs)

```bash
vigil upgrade            # download + verify + replace this binary, then restart the service
vigil upgrade --check    # report only; don't change anything
vigil upgrade --yes      # skip the confirmation prompt
vigil upgrade --force    # install the latest release even if it is older than this build
vigil self update        # identical to `vigil upgrade`
```

`vigil upgrade` checks the latest GitHub release, downloads the archive for your
OS/arch, **verifies it against `checksums.txt`**, replaces the running binary,
and — if a service/logon-task is installed — stops it around the swap and starts
it again.

It only ever moves forward. If the running version is newer than the latest
release, is a source build at or past that release's tag, or cannot be parsed,
`vigil upgrade` refuses with `refusing to replace <current> with <release>` and
changes nothing; `--check` reports `no update: <current> is newer than the
latest release`. Pass `--force` to install the release anyway, for example to go
back from a source build to the published binary. `--force` never reinstalls
the version you are already running.

```
 vigil upgrade
   ├─ dev build? ──────────► refuse (install a release, or --force); --check still works
   ├─ package-managed? ────► refuse → brew / scoop / apt (see below)
   └─ standalone:
        check latest ─ same? ──► "already up to date"
                     ─ older? ─► refuse, unless --force  (source build, newer tag, unparseable)
                     ─ newer:
                          │
                   confirm (or --yes)
                          ▼
        stop service ─► download + verify checksums.txt ─► replace binary
                          ▼
        restart service  (macOS: re-grant Accessibility — signature changed)
```

## Package-manager installs

No package-manager channel is published: the GitHub release is the only one. If
a binary sits under a package manager's path, `vigil upgrade` still refuses and
points you to that tool:

| Installed via | Update with |
|---------------|-------------|
| Homebrew | `brew upgrade vigil` |
| Scoop | `scoop update vigil` |
| deb/rpm | `apt upgrade` / `dnf upgrade` |
| `go install` (0.7.0 and earlier) | frozen at 0.7.0: switch to a release or a [source build](#from-source) |

The interactive TUI shows an "update available" banner only for standalone
installs.

## Caveats

- **Dev builds** (`vigil version` shows `dev`) can't self-update — install a
  released build, or pass `--force` to replace the build with the latest
  release. `--check` still works.
- **macOS Accessibility (TCC).** Replacing the binary changes its signature, so
  macOS may drop the Accessibility grant. After updating, re-grant it
  (System Settings → Privacy & Security → Accessibility) and run `vigil doctor`.
- **Windows.** A running executable can't be deleted; the updater replaces it by
  renaming. The background **logon task** is stopped first so its copy is free.
- **Service binary path.** Self-update replaces the binary you ran
  (`vigil upgrade`). If your service points at a different path, re-run
  `vigil install` afterwards so it uses the updated binary.

[← Docs index](../README.md#documentation)
