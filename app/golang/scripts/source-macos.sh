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
#   ./scripts/source-macos.sh                   # build everything
#   ./scripts/source-macos.sh --client-only      # CLI client + keytool only, no GUI apps
#   ./scripts/source-macos.sh --macos --ios      # only these GUI platforms
#                                                 # (any combo of --macos/--ios/--windows/--linux/--android)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/build-common.sh"
build_common_parse_args "$@"
apply_platform_defaults macos ios windows linux android

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
  if [ "$DO_MACOS" -eq 1 ] || [ "$DO_IOS" -eq 1 ]; then
    if ! xcode-select -p >/dev/null 2>&1; then
      echo "error: Xcode Command Line Tools not found. Run: xcode-select --install" >&2
      exit 1
    fi
  fi

  if [ "$DO_MACOS" -eq 1 ]; then
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
  fi

  if [ "$DO_IOS" -eq 1 ]; then
    # --- iOS Xcode Project ---
    # Needs the *full* Xcode.app, not just the Command Line Tools checked
    # above (xcode-select -p can succeed with only CLT installed; `fyne
    # package -os ios` specifically requires the iOS SDK that ships inside
    # Xcode.app itself).
    if xcodebuild -version >/dev/null 2>&1; then
      echo "==> packaging Fyne GUI for iOS..."
      mkdir -p "$DIST_DIR/stage-ios"
      if ! command -v fyne >/dev/null 2>&1; then
        go install fyne.io/fyne/v2/cmd/fyne@latest
      fi
      # `fyne package -os ios` does NOT auto-detect a signing identity —
      # confirmed against fyne's own maintainer (fyne-io/fyne#1217): it
      # only ever signs when you pass -certificate explicitly. Without it,
      # it silently produces an *unsigned* .app, which Xcode's Devices
      # window refuses to install ("No code signature found") — iOS never
      # runs unsigned code on a real device, no exceptions. And no, there's
      # no .xcodeproj hiding in there either to open and sign by hand
      # instead — that was a wrong guess from an earlier round; fyne's iOS
      # packaging builds the .app directly, it never emits a project.
      #
      # IMPORTANT: don't also pass -profile by name, even though fyne's
      # flag exists for it. Confirmed the hard way: whenever -profile is
      # non-empty, fyne's generated Xcode project sets
      # ProvisioningStyle=Manual and pins that exact profile name — but a
      # free Personal Team's profile is "Xcode managed" (auto-rotated),
      # and Xcode flatly refuses to use an Xcode-managed profile under
      # Manual style ("is Xcode managed, but signing settings require a
      # manually managed profile" — this is exactly the error we hit).
      # Leaving -profile empty makes fyne use ProvisioningStyle=Auto
      # instead, and combined with `-allowProvisioningUpdates` (which fyne
      # always passes to xcodebuild), that lets Xcode resolve and rotate
      # the profile itself — exactly like a normal automatic-signing
      # Xcode project does. -certificate is still needed (fyne uses it
      # only to look up your Team ID, not to hardcode the identity).
      #
      # The free-Apple-ID ("Personal Team") path Xcode itself uses is
      # still available here, it just has to be surfaced to fyne
      # explicitly, as one env var:
      #
      #   IOS_CERTIFICATE   e.g. "Apple Development: you@example.com (AB12CD34EF)"
      #
      # One-time setup to obtain it (repeat whenever the free profile's
      # 7-day expiry lapses):
      #   1. In Xcode, create any throwaway iOS App project, set its Team
      #      to your Personal Team (Apple ID) under Signing & Capabilities,
      #      plug in your iPhone, select it as the run destination, hit
      #      Run once. This registers the device, and makes Xcode generate
      #      a matching certificate + provisioning profile for it.
      #   2. Get the certificate name:
      #        security find-identity -v -p codesigning
      #      (copy the quoted string, e.g. "Apple Development: ...").
      #   3. Re-run this script with it set, e.g.:
      #        IOS_CERTIFICATE="Apple Development: you@example.com (AB12CD34EF)" \
      #        ./scripts/source-macos.sh --ios
      #
      # Without it set, we still produce the unsigned .app below (it's
      # harmless to have around) but it will NOT install on a real device.
      (
        cd ./cmd/app
        FYNE_IOS_ARGS=(-os ios -appID network.openchat.app)
        if [ -n "${IOS_CERTIFICATE:-}" ]; then
          echo "==> signing with certificate \"$IOS_CERTIFICATE\" (automatic provisioning)"
          FYNE_IOS_ARGS+=(-certificate "$IOS_CERTIFICATE" -profile "")
        else
          echo "==> IOS_CERTIFICATE not set — building an UNSIGNED .app (won't install on a real iPhone; see comments in this script for how to sign it)" >&2
        fi
        if fyne package "${FYNE_IOS_ARGS[@]}"; then
          mv *.app "$DIST_DIR/stage-ios/" 2>/dev/null || true
        else
          echo "warning: fyne package -os ios failed — see output above" >&2
        fi
      )
    else
      echo "==> skipping iOS packaging: full Xcode.app not found (Command Line Tools alone aren't enough) — install Xcode from the App Store to enable this" >&2
    fi
  fi

  # --- Windows, Linux & Android GUI (Cross-compilation via fyne-cross) ---
  if [ "$DO_WINDOWS" -eq 1 ] || [ "$DO_LINUX" -eq 1 ] || [ "$DO_ANDROID" -eq 1 ]; then
    ensure_fyne_cross
  fi
  if [ "$DO_WINDOWS" -eq 1 ]; then build_fyne_cross_windows; fi
  if [ "$DO_LINUX" -eq 1 ]; then build_fyne_cross_linux; fi
  if [ "$DO_ANDROID" -eq 1 ]; then build_fyne_cross_android; fi
fi

zip_and_cleanup
if [ "$DO_IOS" -eq 1 ]; then
  echo "  (openchat-ios-$APP_VERSION.zip, if produced, contains OpenChat.app — signed and installable on your iPhone only if IOS_CERTIFICATE was set; see comments in scripts/source-macos.sh for how)"
fi
