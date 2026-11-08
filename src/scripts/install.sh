#!/usr/bin/env bash
# Install vigil on macOS/Linux by downloading the latest prebuilt release and
# verifying its SHA-256. Falls back to building from source if Go is present.
#
#   curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/scripts/install.sh | sh
#   ./scripts/install.sh                  # binary only, to ~/.local/bin
#   ./scripts/install.sh --with-service   # also configure + install + start the daemon
#   sudo ./scripts/install.sh             # to /usr/local/bin
set -eu

REPO="simtabi/vigil"
BASE="https://github.com/${REPO}/releases/latest/download"

WITH_SERVICE="${VIGIL_WITH_SERVICE:-0}"
for a in "$@"; do
  case "$a" in
    --with-service) WITH_SERVICE=1 ;;
    *) echo "unknown option: $a" >&2; exit 1 ;;
  esac
done

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in darwin | linux) ;; *) echo "unsupported os: $os" >&2; exit 1 ;; esac

if [ "$(id -u)" = "0" ]; then PREFIX="${PREFIX:-/usr/local/bin}"; else PREFIX="${PREFIX:-$HOME/.local/bin}"; fi
mkdir -p "$PREFIX"

sha_check() { # file expected
  if command -v sha256sum >/dev/null 2>&1; then echo "$2  $1" | sha256sum -c - >/dev/null
  else echo "$2  $1" | shasum -a 256 -c - >/dev/null; fi
}

# On macOS (uname reports "darwin") use the universal binary; assets are named
# with the friendly "macos" token.
if [ "$os" = "darwin" ]; then
  asset="vigil_macos_universal.tar.gz"
else
  asset="vigil_${os}_${arch}.tar.gz"
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset}..."
# Show a progress bar (transfer rate + ETA) on an interactive terminal; stay
# quiet when piped (curl | sh, CI logs).
dl_opts="-fsSL"
[ -t 1 ] && dl_opts="-fL --progress-bar"
# shellcheck disable=SC2086
if curl $dl_opts "${BASE}/${asset}" -o "${tmp}/${asset}" && curl -fsSL "${BASE}/checksums.txt" -o "${tmp}/checksums.txt"; then
  want=$(awk -v a="$asset" '$NF==a || $NF=="./"a {print $1; exit}' "${tmp}/checksums.txt")
  if [ -z "$want" ]; then echo "checksum for ${asset} not found" >&2; exit 1; fi
  echo "Verifying checksum..."
  sha_check "${tmp}/${asset}" "$want"
  tar -C "$tmp" -xzf "${tmp}/${asset}"
  # The archive contains a flat-named binary (e.g. vigil_darwin_universal); install it as vigil.
  install -m 0755 "${tmp}/${asset%.tar.gz}" "${PREFIX}/vigil"
  echo "Installed: ${PREFIX}/vigil"
else
  echo "Download failed; trying to build from source..." >&2
  command -v go >/dev/null 2>&1 || { echo "Go not found and download failed." >&2; exit 1; }
  GOBIN="$PREFIX" go install "github.com/${REPO}/cmd/vigil@latest"
  echo "Installed (from source): ${PREFIX}/vigil"
fi

case ":$PATH:" in *":$PREFIX:"*) ;; *) echo "note: add $PREFIX to your PATH";; esac

if [ "$WITH_SERVICE" = "1" ]; then
  echo "Setting up the background service..."
  "${PREFIX}/vigil" install --init || echo "service setup failed; run '${PREFIX}/vigil' doctor"
  echo "Done. Manage it with: vigil status / vigil restart / vigil stop"
else
  cat <<EOF

Next steps:
  vigil config wizard    # guided setup (or: vigil config init)
  vigil doctor           # check capabilities & permissions
  vigil install          # install + start the background service
                       # (or re-run this installer with --with-service)

Uninstall later:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/uninstall.sh | sh
EOF
fi
