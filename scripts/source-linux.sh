#!/usr/bin/env bash
#
# scripts/source-linux.sh — build everything OpenChat ships that's
# possible from a Linux host: the CLI client + keytool for every target
# OS, and the Fyne GUI app cross-compiled via fyne-cross/Docker for
# Windows, Linux and Android.
#
# What this CANNOT build: the native macOS GUI app and the iOS package —
# both require a real Mac with Xcode.app (Apple provides no cross
# toolchain for either). Run scripts/source-macos.sh on a Mac for those.
#
# Usage:
#   ./scripts/source-linux.sh                # build everything possible here
#   ./scripts/source-linux.sh --client-only   # CLI client + keytool only, no GUI apps
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/build-common.sh"
build_common_parse_args "$@"

prepare_dist_dirs
go_mod_tidy
build_cli_clients

if [ "$CLIENT_ONLY" -eq 1 ]; then
  echo "==> --client-only set, skipping GUI app builds"
else
  echo "==> skipping native macOS GUI app + iOS packaging: both require a real Mac with Xcode.app — run scripts/source-macos.sh there instead" >&2
  ensure_fyne_cross
  build_fyne_cross_windows
  build_fyne_cross_linux
  build_fyne_cross_android
fi

zip_and_cleanup
