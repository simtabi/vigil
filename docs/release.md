# Release process

Releases are tag-driven, and binary releases are on hold. The git tag is the
single source of version truth; `vigil version` is stamped at build time via
ldflags (`internal/cli.version/commit/date`). A `vX.Y.Z` tag records a version;
binaries are built from it only when the release workflow is dispatched by hand
with a confirmation (below). Until binary releases resume, users build from
source ([installation](installation.md#install)).

## Cutting a release

1. Move `[Unreleased]` items in `CHANGELOG.md` under a new `## [X.Y.Z]` section
   with the date.
2. Commit on `main`.
3. Tag and push:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

   Pushing a tag publishes nothing: no workflow starts on a tag push.
4. Only when the owner has approved a binary release for this version, dispatch
   the `Release` workflow (`.github/workflows/release.yml`) on the tag:

   ```bash
   gh workflow run release.yml --repo simtabi/vigil --ref vX.Y.Z -f confirm=release-go
   ```

   `vX.Y.Z` must be a real tag already on GitHub. The command publishes for real
   (GitHub Release, assets, attestation), so run it only for an approved
   version. The first step refuses to run without `confirm=release-go`, or on
   any ref that is not a `v[0-9]*` tag, before anything is built.

## Build system (single-sourced)

The target list lives in **`build/targets.txt`** and the build/bundle logic in
**`scripts/build-all.sh`** — used identically by `make dist` (local) and CI, so
there's one source of truth.

The **Windows binary icon + version info** are embedded via committed
`cmd/vigil/rsrc_windows_*.syso` files that `go build` auto-links — no extra build
step. Regenerate them (and the brand PNGs) from `assets/vigil.svg` with
`make icons` after changing the logo; commit the results. See
[brand.md](brand.md). Targets cover 64-bit, 32-bit (386/armv6/armv7),
ARM64, RISC-V/ppc64le/s390x, Windows (incl. ARM64), macOS, and the BSDs.

`make dist` produces a clean `dist/` root of bare ready-to-run binaries, with
all archives and packages grouped under `dist/archives/`:

```
dist/
  vigil_<os>_<arch>[.exe]      # bare, self-describing binaries (macOS uses "macos"; + vigil_macos_universal)
  checksums.txt              # sha256 over the bare binaries
  archives/
    vigil_<os>_<arch>.tar.gz   # unix; the inner binary KEEPS the flat name
    vigil_windows_<arch>.zip   # windows; inner vigil_windows_<arch>.exe
    vigil_<arch>.deb / .rpm    # nfpm (build/nfpm.yaml)
    checksums.txt            # sha256 over archives/packages
```

Archive names are **version-less** (`vigil_<os>_<arch>.{tar.gz,zip}`) to keep the
self-update contract stable. macOS ships a **universal** binary
(`vigil_macos_universal`, Apple Silicon + Intel). GitHub release assets = the
contents of `dist/archives/`. macOS artifacts use the `macos` token (Go's
`darwin` is hard to recognize); self-update sets the updater's OS to `macos`
accordingly.

```bash
make dist          # build + bundle everything the local toolchain supports
```

## Workflows

```
 gh workflow run release.yml --ref vX.Y.Z -f confirm=release-go ─► release.yml
                      │
                      ├─ build-binaries.yml (reusable)
                      │     ├─ cross job (Linux/Windows/BSD, CGO-free, 1 runner)
                      │     └─ mac job  (darwin arm64+amd64 + universal, cgo)
                      │            └─► dist/archives/*  (+ Windows .ico embedded)
                      ▼
                  release job ─► checksums.txt + CHANGELOG body ─► GitHub Release
                      │
                      └─ brew-scoop ─► update tap + bucket (best-effort, needs token)

 push to main ─► ci.yml snapshot ─► same bundles as run artifacts (version 0.0.0-dev+sha)
```

> GoReleaser's `prebuilt` builder is Pro-only and the macOS backend needs cgo
> (no Linux cross-compile), so builds run natively: Linux/Windows/BSD cross-
> compile CGO-free on one Linux runner; macOS uses cgo on macOS runners.

- **`build-binaries.yml`** (reusable, `workflow_call`): the matrix build —
  a `cross` job (all CGO-free targets from `build/targets.txt`) + a `mac` job
  (darwin arm64/amd64). Uploads `dist-*` artifacts.
- **`release.yml`** — on a confirmed `workflow_dispatch` against a `vX.Y.Z`
  tag only (a tag push starts nothing): calls `build-binaries`, then `release`
  collects artifacts, writes `checksums.txt`, extracts the CHANGELOG section as
  the body, and publishes the GitHub Release; `brew-scoop` updates the
  tap/bucket (best-effort).
- **`ci.yml`** `snapshot` job — on pushes to `main` and on demand: calls
  `build-binaries` so **ready-to-run binaries are always available** as run
  artifacts even before a tag (versioned `0.0.0-dev+<sha>`, which self-update
  treats as a dev build).

## Self-update contract

`internal/selfupdate` downloads `vigil_<os>_<arch>.<ext>` and validates it against
`checksums.txt`. The inner binary keeps the **flat** name
(`vigil_<os>_<arch>[.exe]`), which `go-selfupdate`'s `matchExecutableName` accepts
(`^cmd([_-]v?semver)?([_-]os[_-]arch)?(\.exe)?$`). Keep these aligned: the
archive name, the flat inner-binary name, **bare-filename** `checksums.txt`, and
`vX.Y.Z` tags. Changing one side means changing both.

## First-release prerequisites

1. Make the repo public.
2. Create `simtabi/homebrew-tap` and `simtabi/scoop-bucket` repositories.
3. Add the write deploy keys `HOMEBREW_TAP_DEPLOY_KEY` (on `simtabi/homebrew-tap`) and `SCOOP_BUCKET_DEPLOY_KEY` (on `simtabi/scoop-bucket`), each a repository secret that reaches its one repository and nothing else (the deploy-key guide in
   `imanimanyara/github-apps` `docs/operations.md`). `TAP_GITHUB_TOKEN` is only
   a fallback, and the Winget pull request's token.
4. Push the first `vX.Y.Z` tag, then dispatch the release on it as above.

Every release carries a real, human-readable description sourced from the
CHANGELOG — never a bare "see changelog" stub. Keep GitHub Actions current via
Dependabot.

[← Docs index](../README.md#documentation)
