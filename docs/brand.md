# Brand & visuals

vigil's mark is an **open eye with a green pupil** — vigilance (keeping watch)
plus the Teams "available" dot.

## Assets

| File | Use |
|------|-----|
| `assets/vigil.svg` | Source of truth (hand-authored, scalable). |
| `assets/icons/vigil_{16..512}.png` | Rasterized PNGs for docs, READMEs, packaging. |
| `src/cmd/vigil/rsrc_windows_*.syso` | Windows resource objects — embed the icon + version info into `vigil.exe` (auto-linked by `go build`; ignored on other OSes). |

Regenerate everything from the SVG with:

```bash
make icons        # = cd src/tools/iconz && go run .
```

`src/tools/iconz` is a separate throwaway module (oksvg/rasterx + tc-hib/winres) so the
main module's dependencies stay clean. Editing `assets/vigil.svg` then running
`make icons` rewrites the PNGs and the `.syso` files; commit the results.

A CLI binary only carries an embedded icon on **Windows**. macOS/Linux command-line
binaries have no icon (you'd need a `.app`/`.desktop` wrapper), so they use the
brand assets in docs and package metadata instead.

## Banner

`vigil version` prints a masthead with the name, tagline, version/commit/date,
author, contact, and URLs. It renders a colored rounded box on a terminal and a
plain ASCII box when piped or under `NO_COLOR` (so logs stay clean). `--json` is
unaffected. The identity values live in one place: `src/internal/brand`.

## Icons & status in the TUI

The TUI uses Unicode glyphs (◉ Status, ⏯ Override, ▦ Schedule, ⚙ Settings,
⛭ Service, ◐ Account, ⭮ Update) with the eye (◉) in the header. The status strip
shows a live **connectivity dot**: green ● online, red ● offline (see
[cli.md](cli.md#network-status)).

[← Docs index](../README.md#documentation)
