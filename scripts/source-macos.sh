#!/usr/bin/env bash
#
# scripts/source-macos.sh — build EVERYTHING OpenChat ships, from a Mac:
# the CLI client + keytool for macOS/Windows/Linux/Android, the Fyne GUI
# app natively for macOS, the Fyne GUI packaged for iOS, and the Fyne GUI
# cross-compiled via fyne-cross/Docker for Windows, Linux and Android.
#
# macOS is the only host that can produce the macOS and iOS GUI builds —
# both need a real Xcode install and Apple provides no cross toolchain
# for either. If you're on Linux or Windows, use scripts/source-linux.sh
# or scripts/source-windows.sh instead: they build everything else.
#
# Usage:
#   ./scripts/source-macos.sh                # build everything
#   ./scripts/source-macos.sh --client-only   # CLI client + keytool only, no GUI apps
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/build-common.sh"
build_common_parse_args "$@"

if [ "$(uname -s)" != "Darwin" ]; then
  echo "error: scripts/source-macos.sh must run on an actual Mac (the macOS/iOS builds need Xcode, which only exists there)." >&2
  echo "       On Linux or Windows, use scripts/source-linux.sh or scripts/source-windows.sh — they build everything except those two." >&2
  exit 1
fi

prepare_dist_dirs
go_mod_tidy
build_cli_clients

if [ "$CLIENT_ONLY" -eq 1 ]; then
  echo "==> --client-only set, skipping GUI app builds"
else
  if ! xcode-select -p >/dev/null 2>&1; then
    echo "error: Xcode Command Line Tools not found. Run: xcode-select --install" >&2
    exit 1
  fi

  # --- macOS App Bundle ---
  echo "==> building OpenChat.app GUI bundle for macOS"
  APP_BUNDLE="$DIST_DIR/stage-macos/OpenChat.app"
  mkdir -p "$APP_BUNDLE/Contents/MacOS" "$APP_BUNDLE/Contents/Resources"

  HOST_ARCH="$(uname -m)"
  NATIVE_GOARCH="amd64" && [ "$HOST_ARCH" = "arm64" ] && NATIVE_GOARCH="arm64"

  BUILT_ARCHS=()
  for goarch in arm64 amd64; do
    out="$DIST_DIR/stage-macos/OpenChat-$goarch"
    # Any real compile error here needs to stay visible — don't redirect
    # stderr to /dev/null in this loop.
    if CGO_ENABLED=1 GOOS=darwin GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$out" ./cmd/app; then
      BUILT_ARCHS+=("$goarch")
    else
      rm -f "$out"
    fi
  done

  if [ "${#BUILT_ARCHS[@]}" -eq 0 ]; then
    echo "error: both arm64 and amd64 builds of ./cmd/app failed — see the go build output above" >&2
    exit 1
  fi

  if [ "${#BUILT_ARCHS[@]}" -eq 2 ] && command -v lipo >/dev/null 2>&1; then
    lipo -create -output "$APP_BUNDLE/Contents/MacOS/OpenChat" "$DIST_DIR/stage-macos/OpenChat-arm64" "$DIST_DIR/stage-macos/OpenChat-amd64"
  else
    cp "$DIST_DIR/stage-macos/OpenChat-$NATIVE_GOARCH" "$APP_BUNDLE/Contents/MacOS/OpenChat" 2>/dev/null || cp "$DIST_DIR/stage-macos/OpenChat-${BUILT_ARCHS[0]}" "$APP_BUNDLE/Contents/MacOS/OpenChat"
  fi
  chmod +x "$APP_BUNDLE/Contents/MacOS/OpenChat"
  rm -f "$DIST_DIR/stage-macos"/OpenChat-arm64 "$DIST_DIR/stage-macos"/OpenChat-amd64

  if [ -f cmd/app/Icon.png ]; then cp cmd/app/Icon.png "$APP_BUNDLE/Contents/Resources/Icon.png"; fi
  cat > "$APP_BUNDLE/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://apple.com"><plist version="1.0"><dict><key>CFBundleName</key><string>OpenChat</string><key>CFBundleDisplayName</key><string>OpenChat</string><key>CFBundleIdentifier</key><string>network.openchat.app</string><key>CFBundleVersion</key><string>$APP_VERSION</string><key>CFBundleShortVersionString</key><string>$APP_VERSION</string><key>CFBundleExecutable</key><string>OpenChat</string><key>CFBundleIconFile</key><string>Icon.png</string><key>CFBundlePackageType</key><string>APPL</string><key>LSMinimumSystemVersion</key><string>11.0</string><key>NSHighResolutionCapable</key><true/></dict></plist>
PLIST

  # --- iOS Xcode Project ---
  # Needs the *full* Xcode.app, not just the Command Line Tools checked
  # above (xcode-select -p can succeed with only CLT installed; `fyne
  # package -os ios` specifically requires the iOS SDK that ships inside
  # Xcode.app itself).
  if xcodebuild -version >/dev/null 2>&1; then
    echo "==> packaging Fyne GUI for iOS (creating Xcode project)..."
    mkdir -p "$DIST_DIR/stage-ios"
    if ! command -v fyne >/dev/null 2>&1; then
      go install fyne.io/fyne/v2/cmd/fyne@latest
    fi
    # Generates the iOS Xcode build project structure inside stage-ios.
    # This produces an unsigned Xcode project, not a ready-to-install
    # .ipa — actually running on a device or submitting to the App Store
    # still needs Xcode itself (to open the project, set a signing team,
    # and archive/export), since that step requires your Apple
    # Developer credentials, which nothing in this script has.
    ( cd ./cmd/app && fyne package -os ios -appID network.openchat.app && mv *.app "$DIST_DIR/stage-ios/" 2>/dev/null || mv *.xcodeproj "$DIST_DIR/stage-ios/" 2>/dev/null || true )
  else
    echo "==> skipping iOS packaging: full Xcode.app not found (Command Line Tools alone aren't enough) — install Xcode from the App Store to enable this" >&2
  fi

  # --- Windows, Linux & Android GUI (Cross-compilation via fyne-cross) ---
  ensure_fyne_cross
  build_fyne_cross_windows
  build_fyne_cross_linux
  build_fyne_cross_android
fi

zip_and_cleanup
[ "$(uname -s)" == "Darwin" ] && [ "$CLIENT_ONLY" -eq 0 ] && echo "  (openchat-ios-$APP_VERSION.zip, if produced, contains an Xcode project — open it in Xcode to sign and run/archive it)"
