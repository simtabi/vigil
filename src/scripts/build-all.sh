#!/usr/bin/env bash
# Build ready-to-run binaries for the targets in build/targets.txt.
#
# Layout — a clean dist/ root of bare ready-to-run binaries, with all archives
# and packages grouped under dist/archives/:
#   dist/
#     vigil_<os>_<arch>[.exe]      bare ready-to-run binaries (macOS uses "macos"; incl. vigil_macos_universal)
#     checksums.txt              sha256 over the bare binaries (bare names)
#     archives/
#       vigil_<os>_<arch>.tar.gz   unix archives (inner binary keeps the flat name)
#       vigil_windows_<arch>.zip   windows archives
#       vigil_<arch>.deb / .rpm    Linux packages
#       checksums.txt            sha256 over archives/packages (release + self-update + install)
#
# The same script powers `make dist` and CI, so the target list stays single-sourced.
#   ./scripts/build-all.sh [version]
#
# Env knobs (used by CI; sensible defaults locally):
#   VIGIL_SCOPE=all|cross|darwin   which targets to build (default: all)
#   VIGIL_DARWIN_ARCH=arm64|amd64  restrict darwin to one arch (matrix runners)
#   VIGIL_PACKAGES=1               also build deb/rpm (needs nfpm on PATH)
#   VIGIL_CHECKSUMS=1              write checksums.txt files (default on for `all`)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
SCOPE="${VIGIL_SCOPE:-all}"
DARWIN_ARCH="${VIGIL_DARWIN_ARCH:-}"
PACKAGES="${VIGIL_PACKAGES:-0}"
CHECKSUMS="${VIGIL_CHECKSUMS:-}"
[ -z "$CHECKSUMS" ] && { [ "$SCOPE" = "all" ] && CHECKSUMS=1 || CHECKSUMS=0; }

DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
P=github.com/simtabi/vigil/internal/cli
LDFLAGS="-s -w -X ${P}.version=${VERSION} -X ${P}.date=${DATE} -X ${P}.commit=${COMMIT}"
host_os="$(go env GOOS)"; host_arch="$(go env GOARCH)"

# Always start from a clean slate so stale/renamed artifacts never linger.
rm -rf dist out
mkdir -p dist/archives

# os_label maps a GOOS to the human-friendly OS token used in artifact names.
# darwin -> macos (much easier to recognize than "darwin").
os_label() { [ "$1" = "darwin" ] && echo "macos" || echo "$1"; }

# binary_name returns the flat, self-describing binary name for a target.
binary_name() { # goos goarch goarm
  local os base
  os="$(os_label "$1")"
  if [ "$2" = "arm" ]; then base="vigil_${os}_armv$3"; else base="vigil_${os}_$2"; fi
  [ "$1" = "windows" ] && base="${base}.exe"
  echo "$base"
}

# platform_label gives a human-friendly name for build output.
platform_label() { # goos goarch goarm
  case "$1/$2" in
    darwin/arm64) echo "macOS (Apple Silicon)" ;;
    darwin/amd64) echo "macOS (Intel)" ;;
    windows/amd64) echo "Windows 64-bit (x64)" ;;
    windows/386) echo "Windows 32-bit (x86)" ;;
    windows/arm64) echo "Windows ARM64" ;;
    linux/amd64) echo "Linux 64-bit (x64)" ;;
    linux/386) echo "Linux 32-bit (x86)" ;;
    linux/arm64) echo "Linux ARM64" ;;
    linux/arm) echo "Linux ARMv$3 (32-bit)" ;;
    *) echo "$1/$2" ;;
  esac
}

# archive_one bundles a flat binary into dist/archives/ (and deb/rpm for Linux).
# The bare binary stays in the dist/ root.
archive_one() { # goos goarch bin
  local goos="$1" goarch="$2" bin="$3"
  if [ "$goos" = "windows" ]; then
    (cd dist && zip -q "archives/${bin%.exe}.zip" "$bin")
  else
    tar -C dist -czf "dist/archives/${bin}.tar.gz" "$bin"
  fi
  if [ "$PACKAGES" = "1" ] && [ "$goos" = "linux" ] && command -v nfpm >/dev/null 2>&1; then
    case "$goarch" in
      amd64 | arm64 | 386)
        ARCH="$goarch" VERSION="${VERSION#v}" nfpm pkg --config build/nfpm.yaml --packager deb --target dist/archives/ || true
        ARCH="$goarch" VERSION="${VERSION#v}" nfpm pkg --config build/nfpm.yaml --packager rpm --target dist/archives/ || true
        ;;
    esac
  fi
}

build_one() { # goos goarch goarm
  local goos="$1" goarch="$2" goarm="${3:-}" cgo=0 cc=""
  if [ "$goos" = "darwin" ]; then
    cgo=1
    [ "$SCOPE" = "cross" ] && return 0
    [ -n "$DARWIN_ARCH" ] && [ "$DARWIN_ARCH" != "$goarch" ] && return 0
    if [ "$host_os" != "darwin" ]; then echo "   skip darwin/$goarch (needs macOS + cgo)"; return 0; fi
    [ "$goarch" != "$host_arch" ] && cc="clang -arch $([ "$goarch" = amd64 ] && echo x86_64 || echo arm64)"
  else
    [ "$SCOPE" = "darwin" ] && return 0
  fi

  local bin
  bin="$(binary_name "$goos" "$goarch" "$goarm")"
  echo ">> ${bin}  —  $(platform_label "$goos" "$goarch" "$goarm") (cgo=${cgo})"
  if ! env GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" CGO_ENABLED="$cgo" ${cc:+CC="$cc"} \
        go build -trimpath -ldflags "$LDFLAGS" -o "dist/${bin}" ./cmd/vigil; then
    echo "   FAILED ${bin}"; rm -f "dist/${bin}"; return 1
  fi
  archive_one "$goos" "$goarch" "$bin"
}

while read -r goos goarch goarm _; do
  case "$goos" in ''|\#*) continue ;; esac
  build_one "$goos" "$goarch" "$goarm"
done < build/targets.txt

# macOS universal binary — Apple Silicon + Intel in one file, runs on any Mac.
if [ -f dist/vigil_macos_arm64 ] && [ -f dist/vigil_macos_amd64 ] && command -v lipo >/dev/null 2>&1; then
  echo ">> vigil_macos_universal  —  macOS Universal (Apple Silicon + Intel)"
  # lipo to a temp path then rename, to avoid an occasional in-place temp-file race.
  lipo -create -output dist/.universal.tmp dist/vigil_macos_arm64 dist/vigil_macos_amd64
  mv -f dist/.universal.tmp dist/vigil_macos_universal
  tar -C dist -czf dist/archives/vigil_macos_universal.tar.gz vigil_macos_universal
fi

# checksums.txt for the bare binaries (dist root) and for archives/packages.
if [ "$CHECKSUMS" = "1" ]; then
  ( cd dist && shasum -a 256 -- vigil_* 2>/dev/null > checksums.txt || true )
  ( cd dist/archives && shasum -a 256 -- vigil_* 2>/dev/null > checksums.txt || true )
fi

echo
echo "Bare binaries in ./dist:"
find dist -maxdepth 1 -type f -name 'vigil_*' | sort | sed 's/^/  /'
echo "Archives/packages in ./dist/archives:"
find dist/archives -maxdepth 1 -type f -name 'vigil_*' | sort | sed 's/^/  /'
