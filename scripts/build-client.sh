#!/usr/bin/env bash
#
# build-all.sh — builds binaries for macOS, Windows, Linux, Android, and iOS
# (CLI client, keytool, and the Fyne GUI app) and packages them into zip files.
#
# IMPORTANT: The Fyne GUI app uses CGO and graphics bindings. 
# - macOS/iOS GUI builds require a Mac host with Xcode CLT installed.
# - Windows/Linux/Android GUI builds require Docker + fyne-cross installed.

set -euo pipefail

CLIENT_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --client-only) CLIENT_ONLY=1 ;;
    *) echo "unknown argument: $arg" >&2; exit 1 ;;
  esac
done

REPO_ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE}")/.." && pwd)}"
cd "$REPO_ROOT"

if [ ! -f go.mod ]; then
  echo "error: $REPO_ROOT doesn't look like the openchat repo root (no go.mod)" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "error: Go toolchain not found. Install from https://go.dev first." >&2
  exit 1
fi

# Verify Docker and fyne-cross for cross-compilation if GUI is enabled
if [ "$CLIENT_ONLY" -eq 0 ]; then
  if ! command -v fyne-cross >/dev/null 2>&1; then
    echo "==> installing fyne-cross..."
    go install github.com/fyne-io/fyne-cross@latest
  fi
  if ! docker info >/dev/null 2>&1; then
    echo "error: Docker is not running. It is required by fyne-cross to build Windows/Linux/Android GUI apps." >&2
    exit 1
  fi
fi

APP_VERSION="$(git describe --tags --always --dirty 2>/dev/null || date +%Y%m%d%H%M%S)"
DIST_DIR="$REPO_ROOT/dist"

# Clean up previous temporary build dirs if they exist
rm -rf "$DIST_DIR/stage-macos" "$DIST_DIR/stage-windows" "$DIST_DIR/stage-linux" "$DIST_DIR/stage-android" "$DIST_DIR/stage-ios" "$DIST_DIR/fyne-cross"
mkdir -p "$DIST_DIR/stage-macos" "$DIST_DIR/stage-windows" "$DIST_DIR/stage-linux" "$DIST_DIR/stage-android"

echo "==> go mod tidy"
go mod tidy

# android_ndk_clang tries to locate the Android NDK's clang for a given
# Go GOARCH, so the "Android CLI" build below can actually cross-compile.
# GOOS=android *always* requires cgo/external linking in the Go toolchain
# — CGO_ENABLED=0 does not work for it no matter what the code itself
# does, since the runtime needs Android's bionic libc, not Go's normal
# syscall path. Termux itself ships its own toolchain and compiles Go
# natively on-device, which is the far more common way to get an
# openchat-client binary running inside Termux — this cross-compiled
# build is a convenience for those who specifically want to produce it
# from a desktop instead, and only runs if an NDK is actually configured.
android_ndk_clang() {
  local arch="$1" triple api="21"
  case "$arch" in
    amd64) triple="x86_64-linux-android" ;;
    arm64) triple="aarch64-linux-android" ;;
    *) return 1 ;;
  esac
  if [ -n "${ANDROID_NDK_CC:-}" ]; then
    echo "$ANDROID_NDK_CC"
    return 0
  fi
  if [ -z "${ANDROID_NDK_HOME:-}" ]; then
    return 1
  fi
  local candidate host_tag
  for host_tag in linux-x86_64 darwin-x86_64; do
    candidate="$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/$host_tag/bin/${triple}${api}-clang"
    if [ -x "$candidate" ]; then
      echo "$candidate"
      return 0
    fi
  done
  return 1
}

# ==========================================
# 1. BUILD CLI CLIENTS & KEYTOOLS (All OS)
# ==========================================
echo "==> building CLI clients and keytools..."
for arch in arm64 amd64; do
  # macOS
  CGO_ENABLED=0 GOOS=darwin GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/stage-macos/openchat-client-$arch" ./cmd/client
  CGO_ENABLED=0 GOOS=darwin GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/stage-macos/openchat-keytool-$arch" ./cmd/keytool

  # Windows
  CGO_ENABLED=0 GOOS=windows GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/stage-windows/openchat-client-$arch.exe" ./cmd/client
  CGO_ENABLED=0 GOOS=windows GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/stage-windows/openchat-keytool-$arch.exe" ./cmd/keytool

  # Linux
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/stage-linux/openchat-client-$arch" ./cmd/client
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/stage-linux/openchat-keytool-$arch" ./cmd/keytool

  # Android CLI (for Termux) — best-effort, skipped unless an NDK clang is
  # findable (see android_ndk_clang above). Set ANDROID_NDK_HOME (or
  # ANDROID_NDK_CC to the exact clang path) to enable it.
  if ndk_cc="$(android_ndk_clang "$arch")"; then
    CC="$ndk_cc" CGO_ENABLED=1 GOOS=android GOARCH=$arch go build -trimpath -ldflags="-s -w -checklinkname=0" -o "$DIST_DIR/stage-android/openchat-client-$arch" ./cmd/client
    CC="$ndk_cc" CGO_ENABLED=1 GOOS=android GOARCH=$arch go build -trimpath -ldflags="-s -w -checklinkname=0" -o "$DIST_DIR/stage-android/openchat-keytool-$arch" ./cmd/keytool
  else
    echo "==> skipping Android CLI ($arch): no NDK clang found — set ANDROID_NDK_HOME (or ANDROID_NDK_CC) to build it, or just compile natively inside Termux instead" >&2
  fi
done


# ==========================================
# 2. BUILD GUI APPS (Fyne)
# ==========================================
if [ "$CLIENT_ONLY" -eq 1 ]; then
  echo "==> --client-only set, skipping GUI app builds"
else
  # --- macOS & iOS GUI (Native Apple Host Required) ---
  if [ "$(uname -s)" == "Darwin" ]; then
    if ! xcode-select -p >/dev/null 2>&1; then
      echo "error: Xcode Command Line Tools not found. Run: xcode-select --install" >&2
      exit 1
    fi

    # macOS App Bundle
    echo "==> building OpenChat.app GUI bundle for macOS"
    APP_BUNDLE="$DIST_DIR/stage-macos/OpenChat.app"
    mkdir -p "$APP_BUNDLE/Contents/MacOS" "$APP_BUNDLE/Contents/Resources"

    HOST_ARCH="$(uname -m)"
    NATIVE_GOARCH="amd64" && [ "$HOST_ARCH" = "arm64" ] && NATIVE_GOARCH="arm64"

    BUILT_ARCHS=()
    for goarch in arm64 amd64; do
      out="$DIST_DIR/stage-macos/OpenChat-$goarch"
      if CGO_ENABLED=1 GOOS=darwin GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$out" ./cmd/app 2>/dev/null; then
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

    # iOS Xcode Project
    echo "==> packaging Fyne GUI for iOS (creating Xcode project)..."
    mkdir -p "$DIST_DIR/stage-ios"
    if ! command -v fyne >/dev/null 2>&1; then
      go install fyne.io/fyne/v2/cmd/fyne@latest
    fi
    # Generates iOS Xcode build project structure inside stage-ios
    ( cd ./cmd/app && fyne package -os ios -appID network.openchat.app && mv *.app "$DIST_DIR/stage-ios/" 2>/dev/null || mv *.xcodeproj "$DIST_DIR/stage-ios/" 2>/dev/null || true )
  else
    echo "==> Not on macOS. Skipping native macOS and iOS GUI App builds."
  fi

  # --- Windows, Linux & Android GUI (Cross-compilation via fyne-cross) ---
  echo "==> building Cross-Platform GUI bundles via fyne-cross..."
  
  # Windows GUI (amd64)
  fyne-cross windows --arch=amd64 --dir=./cmd/app/
  if [ -f fyne-cross/bin/windows-amd64/app.exe ]; then
    mv fyne-cross/bin/windows-amd64/app.exe "$DIST_DIR/stage-windows/OpenChat.exe"
  fi

  # Linux GUI (amd64)
  fyne-cross linux --arch=amd64 --dir=./cmd/app/
  if [ -f fyne-cross/bin/linux-amd64/app ]; then
    mv fyne-cross/bin/linux-amd64/app "$DIST_DIR/stage-linux/OpenChat"
  fi

  # Android GUI APK
  fyne-cross android --dir=./cmd/app/
  if [ -f fyne-cross/bin/android/app.apk ]; then
    mv fyne-cross/bin/android/app.apk "$DIST_DIR/stage-android/OpenChat.apk"
  fi
fi

# ==========================================
# 3. PACKAGING ZIP ARCHIVES
# ==========================================
echo "==> zipping up final distributions..."

# Package macOS
if [ "$(ls -A "$DIST_DIR/stage-macos")" ]; then
  ( cd "$DIST_DIR/stage-macos" && zip -qr -X "$DIST_DIR/openchat-macos-$APP_VERSION.zip" . )
fi

# Package iOS Project
if [ -d "$DIST_DIR/stage-ios" ] && [ "$(ls -A "$DIST_DIR/stage-ios")" ]; then
  ( cd "$DIST_DIR/stage-ios" && zip -qr -X "$DIST_DIR/openchat-ios-$APP_VERSION.zip" . )
fi

# Package Windows
( cd "$DIST_DIR/stage-windows" && zip -qr -X "$DIST_DIR/openchat-windows-$APP_VERSION.zip" . )

# Package Linux
( cd "$DIST_DIR/stage-linux" && zip -qr -X "$DIST_DIR/openchat-linux-$APP_VERSION.zip" . )

# Package Android
( cd "$DIST_DIR/stage-android" && zip -qr -X "$DIST_DIR/openchat-android-$APP_VERSION.zip" . )

# Clean temporary folders
rm -rf "$DIST_DIR/stage-macos" "$DIST_DIR/stage-windows" "$DIST_DIR/stage-linux" "$DIST_DIR/stage-android" "$DIST_DIR/stage-ios" "$DIST_DIR/fyne-cross"

echo
echo "Done! Generated files in $DIST_DIR/:"
echo "  - openchat-macos-$APP_VERSION.zip"
echo "  - openchat-windows-$APP_VERSION.zip"
echo "  - openchat-linux-$APP_VERSION.zip"
echo "  - openchat-android-$APP_VERSION.zip"
[ "$(uname -s)" == "Darwin" ] && echo "  - openchat-ios-$APP_VERSION.zip (Contains iOS project targets)" || true
