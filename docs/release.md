# Release process

Releases are tag-driven, and binary releases are on hold. The git tag is the
single source of version truth; `vigil version` is stamped at build time via
ldflags (`github.com/simtabi/vigil/src/internal/cli.version/commit/date`). A `vX.Y.Z` tag records a version;
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
4. Only when the owner has approved a binary release for this version, turn the
   `Release` workflow (`.github/workflows/release.yml`) on, dispatch it on the
   tag, and turn it off again once the run has started:

   ```bash
   gh workflow enable release.yml --repo simtabi/vigil
   gh workflow run release.yml --repo simtabi/vigil --ref vX.Y.Z -f confirm=release-go
   gh workflow disable release.yml --repo simtabi/vigil
   ```

   `vX.Y.Z` must be a real tag already on GitHub. The command publishes for real
   (GitHub Release, assets, attestation), so run it only for an approved
   version. The first step refuses to run without `confirm=release-go`, or on
   any ref that is not a `v[0-9]*` tag, before anything is built.

**The `Release` workflow is disabled in Actions while binary releases are on
hold** (since 2026-11-08). A dispatch runs the copy of `release.yml` that the tag
carries, and the tags `v0.1.0` through `v0.7.0` carry an older one that accepts
a manual dispatch and publishes with no confirmation step. A tag cannot be
changed, so the workflow's enabled state is the only control that covers them:
a disabled workflow runs on no ref. Check it with
`gh workflow list --repo simtabi/vigil --all`, which shows `Release` as
`disabled_manually`.

## Repository layout

The Go module lives in **`src/`** (`github.com/simtabi/vigil/src`): `src/go.mod`,
`src/cmd/vigil`, `src/internal`, `src/tui`, `src/scripts`,
`src/.goreleaser.yaml` and `src/.golangci.yml`. Everything the repository
presents (README, docs, CHANGELOG, LICENSE, Makefile, `.github/`, `assets/`,
`build/`) stays at the root. Every `go` command names the module with
`go -C src …`; a bare `go test ./...` at the root finds no packages and passes.

Because GitHub's `go-import` tag has no subdirectory field, Go resolves a
version of a module in `src/` through a **`src/vX.Y.Z`** tag, not `vX.Y.Z`. The
release workflow mirrors `src/vX.Y.Z` onto the release tag's commit before
GoReleaser runs, and never moves one that already exists.

## Build system (single-sourced)

The target list lives in **`build/targets.txt`** and the build/bundle logic in
**`src/scripts/build-all.sh`** — used identically by `make cross` / `make dist`
(local) and CI, so there's one source of truth. Every target but darwin is a
static, CGO-free cross-compile; darwin needs cgo (the input engine links
CoreGraphics/IOKit), so it builds only on a macOS host, with `amd64` cross-built
through `clang -arch x86_64`.

The **Windows binary icon + version info** are embedded via committed
`src/cmd/vigil/rsrc_windows_*.syso` files that `go build` auto-links — no extra
build step. Regenerate them (and the brand PNGs) from `assets/vigil.svg` with
`make icons` after changing the logo; commit the results. See
[brand.md](brand.md). Targets cover 64-bit, 32-bit (386/armv6/armv7),
ARM64, RISC-V/ppc64le/s390x, Windows (incl. ARM64), macOS, and the BSDs.

`build/` holds one folder per OS and nothing else. `make build` writes the host
binary, `make cross` every target, and `make dist` adds the archives:

```
build/
  targets.txt  nfpm.yaml        # tracked inputs
  checksums.txt                 # sha256 over every binary, as {os}/{file}
  macos/    vigil  vigil_macos_arm64  vigil_macos_amd64  vigil_macos_universal
  linux/    vigil_linux_amd64  vigil_linux_386  vigil_linux_arm64  vigil_linux_armv7  …
  windows/  vigil_windows_amd64.exe  vigil_windows_386.exe  vigil_windows_arm64.exe
  freebsd/  openbsd/  netbsd/
  archives/                     # make dist only
    vigil_<os>_<arch>.tar.gz    # unix; the inner binary KEEPS the flat name
    vigil_windows_<arch>.zip    # windows; inner vigil_windows_<arch>.exe
    vigil_<arch>.deb / .rpm     # nfpm (build/nfpm.yaml), with VIGIL_PACKAGES=1
    checksums.txt               # sha256 over archives/packages, bare names
```

The OS folder is the OS token (`macos`, never `darwin`) and each file keeps its
full self-describing name. There is no `build/dist/` and no per-target subfolder;
`make check-build` (`TestBuildLayout` in `src/internal/release`) fails on anything
else, and runs after every `make build`, `make cross` and `make dist`. Verify a
build with:

```bash
make cross
cd build && shasum -a 256 -c checksums.txt
```

Archive names are **version-less** (`vigil_<os>_<arch>.{tar.gz,zip}`) to keep the
self-update contract stable. macOS ships a **universal** binary
(`vigil_macos_universal`, Apple Silicon + Intel). macOS artifacts use the `macos`
token (Go's `darwin` is hard to recognize); self-update sets the updater's OS to
`macos` accordingly. GitHub release assets stay flat: a dispatched release runs
GoReleaser from the repository root with `--config src/.goreleaser.yaml`, which
writes its scratch dist to `.cache/goreleaser/` (gitignored), never into `build/`,
and uploads from there.

## Workflows

```
 gh workflow run release.yml --repo simtabi/vigil --ref vX.Y.Z -f confirm=release-go
                      │
                      ▼
                 release.yml (macOS runner)
                      ├─ refuse unless confirm=release-go on a v[0-9]* tag
                      ├─ mirror src/vX.Y.Z onto the tag
                      ├─ GoReleaser --config src/.goreleaser.yaml ─► .cache/goreleaser/
                      │     binaries + archives + deb/rpm/apk + source tarball + SBOMs
                      │     + checksums.txt ─► GitHub Release, tap/bucket manifests
                      ├─ release body from the CHANGELOG section
                      └─ build-provenance attestation over .cache/goreleaser/checksums.txt

 pull request ─► ci.yml snapshot ─► build-binaries.yml ─► build/archives/* as run artifacts
                                                          (version 0.0.0-dev+sha)
```

- **`build-binaries.yml`** (reusable, `workflow_call`): the matrix build —
  a `cross` job (all CGO-free targets from `build/targets.txt`, plus deb/rpm) and
  a `mac` job (darwin arm64/amd64 + universal, cgo). Each runs
  `src/scripts/build-all.sh`, verifies `build/checksums.txt` and the `build/`
  layout, and uploads `build/archives/*`.
- **`release.yml`** — on a confirmed `workflow_dispatch` against a `vX.Y.Z`
  tag only (a tag push starts nothing): GoReleaser cross-compiles and publishes
  the GitHub Release, the body is set from the CHANGELOG section, and the
  artifacts get a build-provenance attestation.
- **`ci.yml`** `snapshot` job — on every pull request and on demand: calls
  `build-binaries` so **ready-to-run binaries are available** as run artifacts
  before a change merges (versioned `0.0.0-dev+<sha>`, which self-update treats
  as a dev build). It publishes nothing.

## Self-update contract

`src/internal/selfupdate` downloads `vigil_<os>_<arch>.<ext>` and validates it against
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
