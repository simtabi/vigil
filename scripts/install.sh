#!/usr/bin/env bash
# Build mta from source and install it on macOS/Linux.
#
#   ./scripts/install.sh            # install to ~/.local/bin (no sudo)
#   sudo ./scripts/install.sh       # install to /usr/local/bin (all users)
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v go >/dev/null 2>&1; then
  echo "error: Go is required (https://go.dev/dl). Install Go >= 1.23 and retry." >&2
  exit 1
fi

if [ "$(id -u)" -eq 0 ]; then
  PREFIX="${PREFIX:-/usr/local/bin}"
else
  PREFIX="${PREFIX:-$HOME/.local/bin}"
fi
mkdir -p "$PREFIX"

echo "Building mta..."
go build -trimpath -o "$PREFIX/mta" .

echo "Installed: $PREFIX/mta"
case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *) echo "note: add $PREFIX to your PATH (e.g. in ~/.profile or ~/.zshrc)";;
esac

cat <<'EOF'

Next steps:
  mta config init      # write the default config
  mta doctor           # check capabilities & permissions
  mta install          # install + start the background service (use --scope user for input)
EOF
