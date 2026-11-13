#!/usr/bin/env bash
#
# Cross-compile every target in build/targets.txt into build/, one folder per OS.
# It builds on this machine and publishes nothing; GoReleaser
# (src/.goreleaser.yaml) is the release path, run by release.yml on a pushed
# vX.Y.Z tag or on request (docs/release.md).
#
# build/ holds one folder per OS and nothing else:
#
#   build/targets.txt                     tracked input, "GOOS GOARCH [GOARM]"
#   build/nfpm.yaml                       tracked input, deb/rpm packaging
#   build/checksums.txt                   SHA-256 over every binary, as {os}/{file},
#                                         so `cd build && shasum -a 256 -c checksums.txt`
#   build/macos/vigil_macos_arm64         ... _amd64, and _universal (lipo, macOS host)
#   build/linux/vigil_linux_amd64         ... _386, _arm64, _armv6, _armv7, ...
#   build/windows/vigil_windows_amd64.exe
#   build/freebsd/ build/openbsd/ build/netbsd/
#   build/archives/                       only with ARCHIVES=1: .tar.gz / .zip per binary,
#                                         deb/rpm with VIGIL_PACKAGES=1, and their own
#                                         bare-name checksums.txt
#
# The OS folder and the name use `macos`, never `darwin`; arm is armvN; Windows
# binaries end in .exe. Every target but darwin is static (CGO_ENABLED=0). The
# darwin targets need cgo, because the input engine
# (src/internal/activity/input_darwin.go) links CoreGraphics/IOKit, so they build
# only on a macOS host, with amd64 cross-built through clang -arch x86_64.
#
# Every generated file under build/ is removed first, so nothing stale from a
# removed target survives. targets.txt and nfpm.yaml are inputs and are never
# touched.
#
#   ./src/scripts/build-all.sh [version]        # binaries only (make cross)
#   ARCHIVES=1 ./src/scripts/build-all.sh       # binaries plus build/archives/ (make dist)
#
# Env knobs (CI sets them; the defaults suit a local build):
#   VERSION / COMMIT / DATE           stamped into the binary (default: from git)
#   ARCHIVES=1                        also write build/archives/
#   VIGIL_SCOPE=all|cross|darwin      which targets to build (default: all)
#   VIGIL_DARWIN_ARCH=arm64|amd64     restrict darwin to one arch
#   VIGIL_PACKAGES=1                  also build deb/rpm into build/archives/ (needs nfpm)
#   VIGIL_CHECKSUMS=0|1               write build/archives/checksums.txt
#                                     (default: on for VIGIL_SCOPE=all)
#   TARGETS=path/to/list              another target list
#
# build/checksums.txt is always written: it is what `make cross` is checked by.
#
# Portable to bash 3.2 (the macOS system bash): no mapfile / wait -n / assoc arrays.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

BINARY="vigil"
PKG="github.com/simtabi/vigil/src/internal/cli"
BUILD_DIR="build"
ARCHIVE_DIR="${BUILD_DIR}/archives"
TARGETS_FILE="${TARGETS:-${BUILD_DIR}/targets.txt}"

VERSION="${1:-${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
LDFLAGS="-s -w -X ${PKG}.version=${VERSION} -X ${PKG}.commit=${COMMIT} -X ${PKG}.date=${DATE}"

SCOPE="${VIGIL_SCOPE:-all}"
DARWIN_ARCH="${VIGIL_DARWIN_ARCH:-}"
PACKAGES="${VIGIL_PACKAGES:-0}"
WITH_ARCHIVES="${ARCHIVES:-0}"
# Packages land in build/archives/, so asking for them asks for that folder.
[ "$PACKAGES" = "1" ] && WITH_ARCHIVES=1
ARCHIVE_CHECKSUMS="${VIGIL_CHECKSUMS:-}"
if [ -z "$ARCHIVE_CHECKSUMS" ]; then
  if [ "$SCOPE" = "all" ]; then ARCHIVE_CHECKSUMS=1; else ARCHIVE_CHECKSUMS=0; fi
fi

# --- preflight ---------------------------------------------------------------
command -v go >/dev/null 2>&1 || { echo "error: 'go' is not on PATH" >&2; exit 1; }
[ -f "$TARGETS_FILE" ] || { echo "error: no target list at ${TARGETS_FILE}" >&2; exit 1; }
case "$SCOPE" in all | cross | darwin) ;; *) echo "error: VIGIL_SCOPE must be all, cross or darwin" >&2; exit 1 ;; esac

host_os="$(go env GOOS)"
host_arch="$(go env GOARCH)"

# Portable SHA-256: Linux ships sha256sum, macOS/BSD ship shasum.
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$@"
  else echo "error: no sha256sum or shasum found" >&2; return 1; fi
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

# --- clear generated output, keep the inputs ---------------------------------
echo "==> ${BINARY} ${VERSION} (${COMMIT}) into ${BUILD_DIR}/"
mkdir -p "$BUILD_DIR"
for entry in "$BUILD_DIR"/* "$BUILD_DIR"/.[!.]*; do
  [ -e "$entry" ] || continue
  case "$(basename "$entry")" in
    targets.txt | nfpm.yaml) ;;
    *) rm -rf "$entry" ;;
  esac
done
[ "$WITH_ARCHIVES" = "1" ] && mkdir -p "$ARCHIVE_DIR"

# make_archive <os-folder> <bare-binary-filename> <goos> <goarch>
# Bundles one binary (inner name kept flat, for self-update) and, for Linux with
# VIGIL_PACKAGES=1, the deb/rpm packages.
make_archive() {
  [ "$WITH_ARCHIVES" = "1" ] || return 0
  local folder="$1" bin="$2" goos="$3" goarch="$4"
  if [ "$goos" = "windows" ]; then
    (cd "${BUILD_DIR}/${folder}" && zip -q "${ROOT}/${ARCHIVE_DIR}/${bin%.exe}.zip" "$bin")
  else
    tar -C "${BUILD_DIR}/${folder}" -czf "${ARCHIVE_DIR}/${bin}.tar.gz" "$bin"
  fi
  if [ "$PACKAGES" = "1" ] && [ "$goos" = "linux" ] && command -v nfpm >/dev/null 2>&1; then
    case "$goarch" in
      amd64 | arm64 | 386)
        ARCH="$goarch" GOARCH="$goarch" VERSION="${VERSION#v}" nfpm pkg --config build/nfpm.yaml --packager deb --target "${ARCHIVE_DIR}/" || true
        ARCH="$goarch" GOARCH="$goarch" VERSION="${VERSION#v}" nfpm pkg --config build/nfpm.yaml --packager rpm --target "${ARCHIVE_DIR}/" || true
        ;;
    esac
  fi
}

# --- build every target ------------------------------------------------------
count=0
failed=0

# `|| [ -n "$goos" ]` processes a final line with no trailing newline; the \r in
# IFS tolerates CRLF line endings.
while IFS=$' \t\r' read -r goos goarch goarm _ || [ -n "${goos:-}" ]; do
  case "$goos" in '' | \#*) continue ;; esac

  cgo=0
  cc=""
  if [ "$goos" = "darwin" ]; then
    [ "$SCOPE" = "cross" ] && continue
    [ -n "$DARWIN_ARCH" ] && [ "$DARWIN_ARCH" != "$goarch" ] && continue
    if [ "$host_os" != "darwin" ]; then
      echo "  ! skip darwin/${goarch} (needs a macOS host and cgo)"
      continue
    fi
    cgo=1
    [ "$goarch" != "$host_arch" ] && cc="clang -arch $([ "$goarch" = amd64 ] && echo x86_64 || echo arm64)"
  else
    [ "$SCOPE" = "darwin" ] && continue
  fi

  os_token="$goos"
  [ "$goos" = "darwin" ] && os_token="macos"
  arch_token="$goarch"
  [ "$goarch" = "arm" ] && arch_token="armv${goarm}"
  ext=""
  [ "$goos" = "windows" ] && ext=".exe"

  name="${BINARY}_${os_token}_${arch_token}${ext}"
  mkdir -p "${BUILD_DIR}/${os_token}"
  printf '  -> %-36s %s (cgo=%s)\n' "${os_token}/${name}" "$(platform_label "$goos" "$goarch" "${goarm:-}")" "$cgo"

  if ! env GOOS="$goos" GOARCH="$goarch" GOARM="${goarm:-}" CGO_ENABLED="$cgo" ${cc:+CC="$cc"} \
    go -C src build -trimpath -ldflags "$LDFLAGS" \
    -o "${ROOT}/${BUILD_DIR}/${os_token}/${name}" ./cmd/vigil; then
    echo "     FAILED ${name}" >&2
    rm -f "${BUILD_DIR}/${os_token}/${name}"
    failed=$((failed + 1))
    continue
  fi

  make_archive "$os_token" "$name" "$goos" "$goarch"
  count=$((count + 1))
done <"$TARGETS_FILE"

[ "$count" -gt 0 ] || { echo "error: no targets built (empty matrix, or a scope this host cannot build)" >&2; exit 1; }

# --- macOS universal (Intel + Apple Silicon in one binary) -------------------
mac="${BUILD_DIR}/macos"
if [ -f "${mac}/${BINARY}_macos_arm64" ] && [ -f "${mac}/${BINARY}_macos_amd64" ]; then
  if command -v lipo >/dev/null 2>&1; then
    uni="${BINARY}_macos_universal"
    printf '  -> %-36s %s\n' "macos/${uni}" "macOS Universal (Apple Silicon + Intel)"
    # lipo to a temp path in the system temp dir, then move: never a stray file in build/.
    tmp="$(mktemp "${TMPDIR:-/tmp}/vigil-universal.XXXXXX")"
    lipo -create -output "$tmp" "${mac}/${BINARY}_macos_arm64" "${mac}/${BINARY}_macos_amd64"
    mv -f "$tmp" "${mac}/${uni}"
    chmod 0755 "${mac}/${uni}"
    make_archive macos "$uni" darwin universal
  else
    echo "  ! 'lipo' not available; skipping macos_universal" >&2
  fi
fi

# --- checksums ---------------------------------------------------------------
# build/checksums.txt names {os}/{file}, so it verifies from inside build/ as-is.
(cd "$BUILD_DIR" && find . -mindepth 2 -maxdepth 2 -type f -name "${BINARY}_*" -not -path './archives/*' |
  sed 's|^\./||' | LC_ALL=C sort | while IFS= read -r f; do sha256 "$f"; done >checksums.txt)
echo "==> wrote ${BUILD_DIR}/checksums.txt"
# The archives' checksums keep bare names, the form release assets use.
if [ "$WITH_ARCHIVES" = "1" ] && [ "$ARCHIVE_CHECKSUMS" = "1" ] && ls "${ARCHIVE_DIR}/${BINARY}_"* >/dev/null 2>&1; then
  (cd "$ARCHIVE_DIR" && sha256 "${BINARY}"_* >checksums.txt)
  echo "==> wrote ${ARCHIVE_DIR}/checksums.txt"
fi

echo "==> done: ${count} binaries in ${BUILD_DIR}/<os>/$([ "$WITH_ARCHIVES" = "1" ] && echo ", archives in ${ARCHIVE_DIR}/")"
[ "$failed" -eq 0 ] || { echo "error: ${failed} target(s) failed to build" >&2; exit 1; }
