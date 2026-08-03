#!/usr/bin/env bash
#
# scripts/build-client.sh — kept for backward compatibility. The build was
# split by host OS into scripts/source-macos.sh, scripts/source-linux.sh
# and scripts/source-windows.sh, since only a real Mac can produce the
# macOS/iOS builds and this single script used to just silently assume it
# was always run on one. This forwards to source-macos.sh, which has the
# same "build everything" behavior this script always had.
#
# On Linux or Windows, call scripts/source-linux.sh / scripts/source-windows.sh
# directly instead — they build everything except the macOS/iOS GUI apps.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "note: build-client.sh now forwards to scripts/source-macos.sh (call scripts/source-linux.sh or scripts/source-windows.sh directly on those hosts)" >&2
exec "$SCRIPT_DIR/source-macos.sh" "$@"
