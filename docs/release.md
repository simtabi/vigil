# Release process

Releases are tag-driven. The git tag is the single source of version truth;
`mta version` is stamped at build time via ldflags
(`internal/cli.version/commit/date`).

## Cutting a release

1. Move `[Unreleased]` items in `CHANGELOG.md` under a new `## [X.Y.Z]` section
   with the date.
2. Commit on `main`.
3. Tag and push:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

The `Release` workflow (`.github/workflows/release.yml`) then runs.

## Build system (single-sourced)

The target list lives in **`build/targets.txt`** and the build/bundle logic in
**`scripts/build-all.sh`** — used identically by `make dist` (local) and CI, so
there's one source of truth. Targets cover 64-bit, 32-bit (386/armv6/armv7),
ARM64, RISC-V/ppc64le/s390x, Windows (incl. ARM64), macOS, and the BSDs.

`make dist` produces a **flat** dist of ready-to-run binaries plus grouped
archives:

```
dist/
  mta_<os>_<arch>[.exe]      # flat, self-describing binaries (+ mta_darwin_universal)
  checksums.txt              # sha256 over the flat binaries
  archives/
    mta_<os>_<arch>.tar.gz   # unix; the inner binary KEEPS the flat name
    mta_windows_<arch>.zip   # windows; inner mta_windows_<arch>.exe
    mta_<arch>.deb / .rpm     # nfpm (build/nfpm.yaml)
    checksums.txt            # sha256 over archives/packages
```

Archive names are **version-less** (`mta_<os>_<arch>.{tar.gz,zip}`) to keep the
self-update contract stable. macOS ships a **universal** binary
(`mta_darwin_universal`, Apple Silicon + Intel). GitHub release assets = the
contents of `dist/archives/`.

```bash
make dist          # build + bundle everything the local toolchain supports
```

## Workflows

> GoReleaser's `prebuilt` builder is Pro-only and the macOS backend needs cgo
> (no Linux cross-compile), so builds run natively: Linux/Windows/BSD cross-
> compile CGO-free on one Linux runner; macOS uses cgo on macOS runners.

- **`build-binaries.yml`** (reusable, `workflow_call`): the matrix build —
  a `cross` job (all CGO-free targets from `build/targets.txt`) + a `mac` job
  (darwin arm64/amd64). Uploads `dist-*` artifacts.
- **`release.yml`** — on every `vX.Y.Z` tag (and `workflow_dispatch` for a given
  tag): calls `build-binaries`, then `release` collects artifacts, writes
  `checksums.txt`, extracts the CHANGELOG section as the body, and publishes the
  GitHub Release; `brew-scoop` updates the tap/bucket (best-effort).
- **`ci.yml`** `snapshot` job — on pushes to `main` and on demand: calls
  `build-binaries` so **ready-to-run binaries are always available** as run
  artifacts even before a tag (versioned `0.0.0-dev+<sha>`, which self-update
  treats as a dev build).

## Self-update contract

`internal/selfupdate` downloads `mta_<os>_<arch>.<ext>` and validates it against
`checksums.txt`. The inner binary keeps the **flat** name
(`mta_<os>_<arch>[.exe]`), which `go-selfupdate`'s `matchExecutableName` accepts
(`^cmd([_-]v?semver)?([_-]os[_-]arch)?(\.exe)?$`). Keep these aligned: the
archive name, the flat inner-binary name, **bare-filename** `checksums.txt`, and
`vX.Y.Z` tags. Changing one side means changing both.

## First-release prerequisites

1. Make the repo public.
2. Create `simtabi/homebrew-tap` and `simtabi/scoop-bucket` repositories.
3. Add a repo secret **`TAP_GITHUB_TOKEN`** (a repo-scoped PAT that can push to
   those two repos).
4. Push the first `vX.Y.Z` tag.

Every release carries a real, human-readable description sourced from the
CHANGELOG — never a bare "see changelog" stub. Keep GitHub Actions current via
Dependabot.

[← Docs index](../README.md#documentation)
