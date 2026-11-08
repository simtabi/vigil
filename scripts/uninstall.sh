#!/bin/sh
# Deprecated location, kept so the published one-liner keeps working:
#
#   curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/scripts/uninstall.sh | sh
#
# The script moved to src/scripts/uninstall.sh when the Go module moved into src/,
# and installers from 0.1.0 to 0.7.0 printed this URL. This file only forwards
# there, with every argument, run locally or piped. It can go in the first
# release after binary releases resume and the README has named the new URL
# for at least one release.
set -eu
case "$0" in
  *uninstall.sh)
    here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
    exec bash "$here/../src/scripts/uninstall.sh" "$@"
    ;;
esac
curl -fsSL https://raw.githubusercontent.com/simtabi/vigil/main/src/scripts/uninstall.sh | sh -s -- "$@"
