#!/usr/bin/env bash
#
# scripts/source-windows.sh — build everything OpenChat ships that's
# possible from a Windows host: the CLI client + keytool for every target
# OS, and the Fyne GUI app cross-compiled via fyne-cross/Docker for
# Windows, Linux and Android.
#
# Requires a POSIX shell (Git Bash or WSL) and Docker Desktop running —
# fyne-cross itself is a Go tool and behaves identically on any host once
# Docker is available, so this script's capabilities are the same as
# scripts/source-linux.sh's.
#
# What this CANNOT build: the native macOS GUI app and the iOS package —
# both require a real Mac with Xcode.app (Apple provides no cross
# toolchain for either). Run scripts/source-macos.sh on a Mac for those.
#
# Usage:
#   ./scripts/source-windows.sh                    # build everything possible here
#   ./scripts/source-windows.sh --client-only       # CLI client + keytool only, no GUI apps
#   ./scripts/source-windows.sh --windows --linux   # only these GUI platforms
#                                                    # (any combo of --windows/--linux/--android)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/build-common.sh"
build_common_parse_args "$@"
apply_platform_defaults windows linux android
reject_unsupported_platforms "windows linux android"

prepare_dist_dirs
go_mod_tidy
build_cli_clients

if [ "$CLIENT_ONLY" -eq 1 ]; then
  echo "==> --client-only set, skipping GUI app builds"
else
  echo "==> skipping native macOS GUI app + iOS packaging: both require a real Mac with Xcode.app — run scripts/source-macos.sh there instead" >&2
  if [ "$DO_WINDOWS" -eq 1 ] || [ "$DO_LINUX" -eq 1 ] || [ "$DO_ANDROID" -eq 1 ]; then
    ensure_fyne_cross
  fi
  if [ "$DO_WINDOWS" -eq 1 ]; then build_fyne_cross_windows; fi
  if [ "$DO_LINUX" -eq 1 ]; then build_fyne_cross_linux; fi
  if [ "$DO_ANDROID" -eq 1 ]; then build_fyne_cross_android; fi
fi

zip_and_cleanup
