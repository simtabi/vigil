#!/usr/bin/env bash
# Install mta on macOS/Linux by downloading the latest prebuilt release and
# verifying its SHA-256. Falls back to building from source if Go is present.
#
#   curl -fsSL https://raw.githubusercontent.com/simtabi/ms-teams-activity/main/scripts/install.sh | sh
#   ./scripts/install.sh            # to ~/.local/bin
#   sudo ./scripts/install.sh       # to /usr/local/bin
set -eu

REPO="simtabi/ms-teams-activity"
BASE="https://github.com/${REPO}/releases/latest/download"

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

# On macOS use the universal binary (Apple Silicon + Intel in one).
if [ "$os" = "darwin" ]; then
  asset="mta_darwin_universal.tar.gz"
else
  asset="mta_${os}_${arch}.tar.gz"
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset}..."
if curl -fsSL "${BASE}/${asset}" -o "${tmp}/${asset}" && curl -fsSL "${BASE}/checksums.txt" -o "${tmp}/checksums.txt"; then
  want=$(awk -v a="$asset" '$NF==a || $NF=="./"a {print $1; exit}' "${tmp}/checksums.txt")
  if [ -z "$want" ]; then echo "checksum for ${asset} not found" >&2; exit 1; fi
  echo "Verifying checksum..."
  sha_check "${tmp}/${asset}" "$want"
  tar -C "$tmp" -xzf "${tmp}/${asset}"
  # The archive contains a flat-named binary (e.g. mta_darwin_universal); install it as mta.
  install -m 0755 "${tmp}/${asset%.tar.gz}" "${PREFIX}/mta"
  echo "Installed: ${PREFIX}/mta"
else
  echo "Download failed; trying to build from source..." >&2
  command -v go >/dev/null 2>&1 || { echo "Go not found and download failed." >&2; exit 1; }
  GOBIN="$PREFIX" go install "github.com/${REPO}/cmd/mta@latest"
  echo "Installed (from source): ${PREFIX}/mta"
fi

case ":$PATH:" in *":$PREFIX:"*) ;; *) echo "note: add $PREFIX to your PATH";; esac
cat <<'EOF'

Next steps:
  mta config init      # or `mta config wizard` for guided setup
  mta doctor           # check capabilities & permissions
  mta install          # install + start the background service
EOF
