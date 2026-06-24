#!/usr/bin/env bash
# Build ready-to-run binaries for the targets listed in build/targets.txt into
# ./dist, with per-target archives and (for Linux) deb/rpm. The same script is
# used locally (`make dist`) and by CI, so the target list stays single-sourced.
#
#   ./scripts/build-all.sh [version]
#
# Env knobs (used by CI; sensible defaults locally):
#   MTA_SCOPE=all|cross|darwin   which targets to build (default: all)
#   MTA_DARWIN_ARCH=arm64|amd64  restrict darwin to one arch (for matrix runners)
#   MTA_PACKAGES=1               also build deb/rpm (needs nfpm on PATH)
#   MTA_CHECKSUMS=1              write dist/checksums.txt (default on for `all`)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
SCOPE="${MTA_SCOPE:-all}"
DARWIN_ARCH="${MTA_DARWIN_ARCH:-}"
PACKAGES="${MTA_PACKAGES:-0}"
CHECKSUMS="${MTA_CHECKSUMS:-}"
[ -z "$CHECKSUMS" ] && { [ "$SCOPE" = "all" ] && CHECKSUMS=1 || CHECKSUMS=0; }

DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X github.com/simtabi/ms-teams-activity/internal/cli.version=${VERSION} -X github.com/simtabi/ms-teams-activity/internal/cli.date=${DATE}"
host_os="$(go env GOOS)"; host_arch="$(go env GOARCH)"
mkdir -p dist out

archive_base() { # goos goarch goarm
  if [ "$2" = "arm" ]; then echo "mta_$1_armv$3"; else echo "mta_$1_$2"; fi
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

  local base bin dirn
  base="$(archive_base "$goos" "$goarch" "$goarm")"
  bin="mta"; [ "$goos" = "windows" ] && bin="mta.exe"
  dirn="dist/${base}"
  mkdir -p "$dirn"
  echo ">> ${base} (cgo=${cgo})"
  if ! env GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" CGO_ENABLED="$cgo" ${cc:+CC="$cc"} \
        go build -trimpath -ldflags "$LDFLAGS" -o "${dirn}/${bin}" ./cmd/mta; then
    echo "   FAILED ${base}"; rm -rf "$dirn"; return 1
  fi
  if [ "$goos" = "windows" ]; then
    (cd "$dirn" && zip -q "../../out/${base}.zip" "$bin")
  else
    tar -C "$dirn" -czf "out/${base}.tar.gz" "$bin"
  fi

  # deb/rpm for mainstream Linux arches.
  if [ "$PACKAGES" = "1" ] && [ "$goos" = "linux" ] && command -v nfpm >/dev/null 2>&1; then
    case "$goarch" in
      amd64 | arm64 | 386)
        ARCH="$goarch" VERSION="${VERSION#v}" nfpm pkg --config build/nfpm.yaml --packager deb --target out/ || true
        ARCH="$goarch" VERSION="${VERSION#v}" nfpm pkg --config build/nfpm.yaml --packager rpm --target out/ || true
        ;;
    esac
  fi
}

while read -r goos goarch goarm _; do
  case "$goos" in ''|\#*) continue ;; esac
  build_one "$goos" "$goarch" "$goarm"
done < build/targets.txt

cp out/* dist/ 2>/dev/null || true
if [ "$CHECKSUMS" = "1" ]; then
  ( cd dist && shasum -a 256 ./*.tar.gz ./*.zip ./*.deb ./*.rpm 2>/dev/null > checksums.txt || true )
fi

echo
echo "Bundled archives in ./dist:"
find dist -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' -o -name '*.deb' -o -name '*.rpm' \) | sort | sed 's/^/  /'
