# Release process

Releases are tag-driven. The git tag is the single source of version truth;
`vigil version` is stamped at build time via ldflags
(`github.com/simtabi/vigil/src/internal/cli.version/commit/date`). Binaries ship
on the GitHub release only: no Homebrew, Scoop, Winget, `go install`, image or
any other channel. A release builds in one of two ways:

```bash
git push origin vX.Y.Z                                            # builds and publishes vX.Y.Z
gh workflow run release.yml --repo simtabi/vigil -f tag=vX.Y.Z    # the same, for a tag already pushed
```

Because a pushed `vX.Y.Z` tag publishes, push one only for final code. A tag is
final: it is never moved, deleted or reused. Test builds use `vX.Y.Z-rc.N`, which
publishes a GitHub pre-release.

## Cutting a release

1. Move `[Unreleased]` items in `CHANGELOG.md` under a new `## [X.Y.Z]` section
   with the date.
2. Land it on `main` through a pull request. A final version with no
   `## [X.Y.Z]` section is refused.
3. Tag the merge commit and push the tag. The push builds and publishes the
   release:

   ```bash
   git -C ~/src/vigil tag vX.Y.Z
   git -C ~/src/vigil push origin vX.Y.Z
   ```

4. To build a release for a tag already on GitHub (a first release of an older
   tag, or a re-run after a failed one), request it instead:

   ```bash
   gh workflow run release.yml --repo simtabi/vigil -f tag=vX.Y.Z
   ```

   This runs `main`'s `release.yml` and builds the named tag. Do not add
   `--ref vX.Y.Z`: that runs the copy of `release.yml` the tag carries, which for
   `v0.1.0` is the older dispatch-only workflow.

**The `Release` workflow is still disabled in Actions** (`disabled_manually`
since 2026-11-08, when binary releases were on hold), so neither a tag push nor a
request runs it until the owner enables it:

```bash
gh workflow enable release.yml --repo simtabi/vigil
gh workflow list --repo simtabi/vigil --all     # Release: active
```

It was disabled because a dispatch with `--ref` runs the workflow the tag
carries. Measured 2026-11-13, the only version tag on GitHub is `v0.1.0`, and its
`release.yml` is dispatch-only and refuses to run without `-f confirm=release-go`,
so enabling the workflow exposes no unconfirmed release path.

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
`macos` accordingly. GitHub release assets stay flat: a release runs
GoReleaser from the repository root with `--config src/.goreleaser.yaml`, which
writes its scratch dist to `.cache/goreleaser/` (gitignored), never into `build/`,
and uploads from there.

## Workflows

```
 git push origin vX.Y.Z   or   gh workflow run release.yml --repo simtabi/vigil -f tag=vX.Y.Z
                      │
                      ▼
                 release.yml ─► simtabi/.github go-release.yml (macos-latest, environment release)
                      ├─ refuse anything but a vX.Y.Z[-pre] tag already on GitHub
                      ├─ refuse a GoReleaser config holding any other publishing block
                      ├─ notes from the CHANGELOG section (a final version without one is refused)
                      ├─ mirror src/vX.Y.Z onto the tag
                      ├─ GoReleaser --config src/.goreleaser.yaml ─► .cache/goreleaser/
                      │     binaries + archives + deb/rpm/apk + source tarball + SBOMs
                      │     + checksums.txt ─► GitHub Release
                      ├─ read the release body back; fail if shorter than the section
                      └─ build-provenance attestation over .cache/goreleaser/checksums.txt

 pull request ─► ci.yml snapshot ─► build-binaries.yml ─► build/archives/* as run artifacts
                                                          (version 0.0.0-dev+sha)
```

- **`build-binaries.yml`** (reusable, `workflow_call`): the matrix build —
  a `cross` job (all CGO-free targets from `build/targets.txt`, plus deb/rpm) and
  a `mac` job (darwin arm64/amd64 + universal, cgo). Each runs
  `src/scripts/build-all.sh`, verifies `build/checksums.txt` and the `build/`
  layout, and uploads `build/archives/*`.
- **`release.yml`** — on a pushed `vX.Y.Z` tag or a `workflow_dispatch` naming
  one, a thin caller of simtabi/.github's shared `go-release.yml` with
  `runs-on: macos-latest` (the darwin targets need cgo) and
  `environment: release`: GoReleaser cross-compiles and publishes the GitHub
  Release, its body is the CHANGELOG section (read back and checked), and the
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
2. Enable the `Release` workflow (see [Cutting a release](#cutting-a-release)).
3. No publishing secrets are needed: the release uses the run's own
   `GITHUB_TOKEN`. The tap, bucket and Winget credentials
   (`HOMEBREW_TAP_DEPLOY_KEY`, `SCOOP_BUCKET_DEPLOY_KEY`, `TAP_GITHUB_TOKEN`) are
   no longer read.
4. Push the first `vX.Y.Z` tag, or request the release of an existing one, as
   above.

Every release carries a real, human-readable description sourced from the
CHANGELOG — never a bare "see changelog" stub. Keep GitHub Actions current via
Dependabot.

[← Docs index](../README.md#documentation)
