# Updating

> **Binary releases are on hold.** No new GitHub release is being published, so
> `vigil upgrade` finds nothing newer than 0.7.0, the last published release.
> To get anything newer, build from source and update from git (below). The
> self-update mechanism described on this page is unchanged and still in the
> binary; it will pick up releases again if they resume.

## From source (the current way to update)

```bash
git -C ~/src/vigil pull --ff-only
make -C ~/src/vigil build
~/src/vigil/vigil self install   # replace the binary on PATH
vigil restart                    # if the background service is installed
```

The clone path is whatever you used in [installation](installation.md#install).
On macOS, re-grant Accessibility after the rebuild (see Caveats).

**Do not run `vigil upgrade` on a source build.** `make build` stamps the
version from `git describe`, so a build between tags reports something like
`v0.7.0-3-gabc1234`. Semver sorts that *below* `0.7.0`, so `vigil upgrade`
treats the 0.7.0 release as newer and would replace your build with it. A build
of exactly a tag (`v0.7.0`) is reported as up to date, and a plain `go build`
without the Makefile reports `dev` and is refused.

## Self-update (standalone installs)

```bash
vigil upgrade            # download + verify + replace this binary, then restart the service
vigil upgrade --check    # report only; don't change anything
vigil upgrade --yes      # skip the confirmation prompt
vigil self update        # identical to `vigil upgrade`
```

`vigil upgrade` checks the latest GitHub release, downloads the archive for your
OS/arch, **verifies it against `checksums.txt`**, replaces the running binary,
and — if a service/logon-task is installed — stops it around the swap and starts
it again.

```
 vigil upgrade
   ├─ dev build? ──────────► refuse (install a release); --check still works
   ├─ package-managed? ────► refuse → brew / scoop / apt (see below)
   └─ standalone:
        check latest ─ newer? ─ no ─► "already up to date"
                          │ yes
                   confirm (or --yes)
                          ▼
        stop service ─► download + verify checksums.txt ─► replace binary
                          ▼
        restart service  (macOS: re-grant Accessibility — signature changed)
```

## Package-manager installs

No package-manager channel is updated while binary releases are on hold. If you
installed through a package manager, `vigil upgrade` refuses and points you
to the right tool:

| Installed via | Update with |
|---------------|-------------|
| Homebrew | `brew upgrade vigil` |
| Scoop | `scoop update vigil` |
| deb/rpm | `apt upgrade` / `dnf upgrade` |
| `go install` | re-run `go install …/cmd/vigil@latest` |

The interactive TUI shows an "update available" banner only for standalone
installs.

## Caveats

- **Dev builds** (`vigil version` shows `dev`) can't self-update — install a
  released build. `--check` still works.
- **macOS Accessibility (TCC).** Replacing the binary changes its signature, so
  macOS may drop the Accessibility grant. After updating, re-grant it
  (System Settings → Privacy & Security → Accessibility) and run `vigil doctor`.
- **Windows.** A running executable can't be deleted; the updater replaces it by
  renaming. The background **logon task** is stopped first so its copy is free.
- **Service binary path.** Self-update replaces the binary you ran
  (`vigil upgrade`). If your service points at a different path, re-run
  `vigil install` afterwards so it uses the updated binary.

[← Docs index](../README.md#documentation)
