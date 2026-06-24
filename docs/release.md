# Release process

Releases are tag-driven. The single source of version truth is the git tag;
`mta version` is stamped at build time via ldflags.

## Cutting a release

1. Update `CHANGELOG.md`: move `[Unreleased]` items under a new
   `## [X.Y.Z]` section with the date.
2. Commit on `main`.
3. Tag and push:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

4. The `Release` workflow (`.github/workflows/release.yml`) builds native
   binaries on per-OS runners (macOS uses cgo), produces archives + SHA-256
   checksums, extracts the matching `CHANGELOG.md` section as the release body
   (with `generate_release_notes` for the PR/contributor list), and publishes a
   GitHub Release.

## Build matrix

| OS | Arch | cgo |
|----|------|-----|
| darwin | arm64, amd64 | yes |
| windows | amd64 | no |
| linux | amd64, arm64 | no |

## Notes

- Every release carries a real, human-readable description sourced from the
  CHANGELOG — never a bare "see changelog" stub.
- Keep GitHub Actions current via Dependabot; re-pin on merged dep PRs.

[← Docs index](../README.md#documentation)
