#!/usr/bin/env bash
# Remove vigil on macOS/Linux: stops + removes the background service and deletes
# the binary (delegates to `vigil self uninstall`, which knows the per-OS teardown).
#
#   curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/scripts/uninstall.sh | sh
#   ./scripts/uninstall.sh            # remove service + binary
#   ./scripts/uninstall.sh --purge    # also delete config and runtime data
set -eu

PURGE=0
for a in "$@"; do
  case "$a" in
    --purge) PURGE=1 ;;
    *) echo "unknown option: $a" >&2; exit 1 ;;
  esac
done

vigil="$(command -v vigil || true)"
if [ -z "$vigil" ]; then
  for p in "$HOME/.local/bin/vigil" /usr/local/bin/vigil; do
    [ -x "$p" ] && vigil="$p" && break
  done
fi
if [ -z "$vigil" ]; then
  echo "vigil not found on PATH or in the usual install dirs; nothing to do." >&2
  exit 1
fi

cmd="\"$vigil\" self uninstall --yes"
[ "$PURGE" = "1" ] && cmd="$cmd --purge"
echo "Removing vigil (service + binary)..."
eval "$cmd"
echo "Done."
