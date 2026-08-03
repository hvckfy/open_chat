#!/usr/bin/env bash
#
# scripts/lib/build-common.sh — shared helpers for scripts/source-{macos,
# linux,windows}.sh. This file is NOT meant to be run directly; each of
# those three thin, host-specific scripts sources it and then calls only
# the functions relevant to what that host can actually build (a real Mac
# is the only host that can produce the native macOS GUI app and the iOS
# package — Apple provides no cross toolchain for either — everything
# else here is buildable from any of the three).
#
# Splitting it out this way means the CLI cross-build loop, the stray
# fyne-cross-artifact cleanup, version tagging, and zip packaging are each
# defined exactly once, instead of three times slowly drifting apart.

if (return 0 2>/dev/null); then
  : # sourced — continue
else
  echo "error: $(basename "${BASH_SOURCE[0]}") is a library; run scripts/source-macos.sh, scripts/source-linux.sh or scripts/source-windows.sh instead" >&2
  exit 1
fi

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$LIB_DIR/../.." && pwd)"
cd "$REPO_ROOT"

if [ ! -f go.mod ]; then
  echo "error: $REPO_ROOT doesn't look like the openchat repo root (no go.mod)" >&2
  exit 1
fi

# Defensive cleanup: this repo has exactly one go.mod, at the repo root.
# fyne-cross's Docker container mounts ./cmd/app/ and, on some code paths
# (older fyne-cross versions, or an interrupted run), writes a *second*
# go.mod there (e.g. "module openchat.exe") as scratch state for the
# cross-build, and/or leaves a fyne_metadata_init.go generated during
# packaging. Since Go treats any directory containing a go.mod as its own
# module boundary, a leftover file like that makes every subsequent plain
# `go build ./cmd/app` from the repo root fail with "main module
# (openchat) does not contain package openchat/cmd/app" — a confusing
# error that has nothing to do with the actual source code. Remove any
# such stray file before building, every time, regardless of host OS.
for stray in cmd/app/go.mod cmd/app/go.sum cmd/app/fyne_metadata_init.go; do
  if [ -f "$stray" ]; then
    echo "==> removing stray $stray left behind by a previous fyne/fyne-cross run" >&2
    rm -f "$stray"
  fi
done

if ! command -v go >/dev/null 2>&1; then
  echo "error: Go toolchain not found. Install from https://go.dev first." >&2
  exit 1
fi

# `go install foo@latest` puts the binary in `go env GOBIN` (or
# `go env GOPATH`/bin if GOBIN is unset) — that directory is very
# commonly *not* on PATH unless the shell profile explicitly adds it, so
# every "go install ...@latest" call below would silently succeed and
# then immediately be reported as "command not found". Make sure it's on
# PATH for the rest of the run regardless of the calling shell's setup.
GOBIN_DIR="$(go env GOBIN)"
[ -z "$GOBIN_DIR" ] && GOBIN_DIR="$(go env GOPATH)/bin"
export PATH="$PATH:$GOBIN_DIR"

APP_VERSION="$(git describe --tags --always --dirty 2>/dev/null || date +%Y%m%d%H%M%S)"
DIST_DIR="$REPO_ROOT/dist"

# fyne-cross auto-detects app metadata (name, icon, app ID, version,
# build) by reading a FyneApp.toml *at the directory it considers the
# project root* — which, since every source-*.sh script cd's to the repo
# root before invoking fyne-cross (see the big comment above
# build_fyne_cross_windows below for why), is the repo root, not
# cmd/app/ where the real FyneApp.toml actually lives. Left alone, that
# means fyne-cross silently falls back to its own generic defaults
# instead: an app named after the current directory ("openchat", not
# "OpenChat" — this went unnoticed for a while because the mis-named
# output was never caught by the file check further down either), no
# app ID at all (which android's build outright refuses to run
# without), and version/build metadata that doesn't match FyneApp.toml.
# Reading the real values here and passing them explicitly to every
# fyne-cross invocation keeps FyneApp.toml as the single source of
# truth without needing fyne-cross to auto-discover it.
fyne_app_toml_value() {
  local key="$1"
  grep -E "^${key}[[:space:]]*=" cmd/app/FyneApp.toml 2>/dev/null | head -n1 | sed -E 's/^[^=]+=[[:space:]]*"?([^"]*)"?[[:space:]]*$/\1/'
}
FYNE_NAME="$(fyne_app_toml_value Name)"; FYNE_NAME="${FYNE_NAME:-OpenChat}"
FYNE_APPID="$(fyne_app_toml_value ID)"; FYNE_APPID="${FYNE_APPID:-network.openchat.app}"
FYNE_APPVERSION="$(fyne_app_toml_value Version)"; FYNE_APPVERSION="${FYNE_APPVERSION:-0.1.0}"
FYNE_APPBUILD="$(fyne_app_toml_value Build)"; FYNE_APPBUILD="${FYNE_APPBUILD:-1}"
# Same directory-mismatch story applies to the icon path specifically.
FYNE_ICON="cmd/app/Icon.png"

CLIENT_ONLY=0

# build_common_parse_args parses the one flag every source-*.sh script
# supports: --client-only (skip all GUI app builds, CLI/keytool only).
build_common_parse_args() {
  for arg in "$@"; do
    case "$arg" in
      --client-only) CLIENT_ONLY=1 ;;
      *) echo "unknown argument: $arg" >&2; exit 1 ;;
    esac
  done
}

# prepare_dist_dirs resets dist/ for a fresh run. The ios stage dir is
# created on demand only by the macOS script (see build_ios_package),
# since it's the only build that ever produces one.
prepare_dist_dirs() {
  rm -rf "$DIST_DIR/stage-macos" "$DIST_DIR/stage-windows" "$DIST_DIR/stage-linux" "$DIST_DIR/stage-android" "$DIST_DIR/stage-ios" "$DIST_DIR/fyne-cross"
  mkdir -p "$DIST_DIR/stage-macos" "$DIST_DIR/stage-windows" "$DIST_DIR/stage-linux" "$DIST_DIR/stage-android"
}

go_mod_tidy() {
  echo "==> go mod tidy"
  go mod tidy
}

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
  for host_tag in linux-x86_64 darwin-x86_64 windows-x86_64; do
    candidate="$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/$host_tag/bin/${triple}${api}-clang"
    if [ -x "$candidate" ]; then
      echo "$candidate"
      return 0
    fi
  done
  return 1
}

# build_cli_clients cross-compiles the CLI client + keytool for every
# target OS. None of this needs cgo (Android aside), so it behaves
# identically no matter which host OS runs it.
build_cli_clients() {
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

    # Android CLI (for Termux) — best-effort, skipped unless an NDK clang
    # is findable (see android_ndk_clang above).
    if ndk_cc="$(android_ndk_clang "$arch")"; then
      CC="$ndk_cc" CGO_ENABLED=1 GOOS=android GOARCH=$arch go build -trimpath -ldflags="-s -w -checklinkname=0" -o "$DIST_DIR/stage-android/openchat-client-$arch" ./cmd/client
      CC="$ndk_cc" CGO_ENABLED=1 GOOS=android GOARCH=$arch go build -trimpath -ldflags="-s -w -checklinkname=0" -o "$DIST_DIR/stage-android/openchat-keytool-$arch" ./cmd/keytool
    else
      echo "==> skipping Android CLI ($arch): no NDK clang found — set ANDROID_NDK_HOME (or ANDROID_NDK_CC) to build it, or just compile natively inside Termux instead" >&2
    fi
  done
}

# ensure_fyne_cross makes sure fyne-cross (and a running Docker) are
# available. Windows/Linux/Android GUI builds all go through it
# regardless of host OS — Docker abstracts the target toolchain away.
ensure_fyne_cross() {
  if ! command -v fyne-cross >/dev/null 2>&1; then
    echo "==> installing fyne-cross..."
    go install github.com/fyne-io/fyne-cross@latest
  fi
  if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
    echo "error: Docker is not running (or not installed). It is required by fyne-cross to build Windows/Linux/Android GUI apps." >&2
    exit 1
  fi
}

# This repo has a single go.mod at the repo root (cmd/app has no go.mod
# of its own) — `--dir=./cmd/app/` tells fyne-cross to treat cmd/app
# *itself* as the project root, where it then can't find go.mod and
# silently fabricates a broken throwaway one, breaking every import.
# Passing the package path as a plain trailing argument instead builds
# ./cmd/app while still resolving go.mod from the current directory
# (repo root, where every source-*.sh script already cd's to).
#
# Each fyne-cross output filename below is found by globbing rather than
# assuming an exact name: fyne-cross derives it from -name/whatever
# metadata it picked up, which has already been wrong once (see the big
# comment above FYNE_NAME) — globbing for "whatever's actually in the
# bin dir" and warning loudly if there's nothing there is more robust
# than hard-coding a guess and silently doing nothing when it's wrong.
build_fyne_cross_windows() {
  echo "==> building Windows GUI via fyne-cross..."
  fyne-cross windows -name="$FYNE_NAME" -icon="$FYNE_ICON" -app-id="$FYNE_APPID" -app-version="$FYNE_APPVERSION" -app-build="$FYNE_APPBUILD" --arch=amd64 ./cmd/app
  local out
  out="$(ls fyne-cross/bin/windows-amd64/*.exe 2>/dev/null | head -n1)"
  if [ -n "$out" ]; then
    mv "$out" "$DIST_DIR/stage-windows/OpenChat.exe"
  else
    echo "warning: fyne-cross windows build produced no .exe in fyne-cross/bin/windows-amd64/ — the GUI app will be missing from the windows zip" >&2
  fi
}

build_fyne_cross_linux() {
  echo "==> building Linux GUI via fyne-cross..."
  fyne-cross linux -name="$FYNE_NAME" -icon="$FYNE_ICON" -app-id="$FYNE_APPID" -app-version="$FYNE_APPVERSION" -app-build="$FYNE_APPBUILD" --arch=amd64 ./cmd/app
  local out
  out="$(ls fyne-cross/bin/linux-amd64/* 2>/dev/null | head -n1)"
  if [ -n "$out" ]; then
    mv "$out" "$DIST_DIR/stage-linux/OpenChat"
  else
    echo "warning: fyne-cross linux build produced no binary in fyne-cross/bin/linux-amd64/ — the GUI app will be missing from the linux zip" >&2
  fi
}

build_fyne_cross_android() {
  echo "==> building Android GUI (APK) via fyne-cross..."
  # -app-id is not just cosmetic here: fyne-cross's android command
  # refuses to run at all without one ("appID is mandatory for
  # android") — this used to come from cmd/app/FyneApp.toml automatically,
  # back when fyne-cross was invoked from inside cmd/app itself.
  fyne-cross android -name="$FYNE_NAME" -icon="$FYNE_ICON" -app-id="$FYNE_APPID" -app-version="$FYNE_APPVERSION" -app-build="$FYNE_APPBUILD" ./cmd/app
  local out
  out="$(ls fyne-cross/bin/android*/*.apk 2>/dev/null | head -n1)"
  if [ -n "$out" ]; then
    mv "$out" "$DIST_DIR/stage-android/OpenChat.apk"
  else
    echo "warning: fyne-cross android build produced no .apk — the GUI app will be missing from the android zip" >&2
  fi
}

# zip_and_cleanup packages every non-empty stage-* dir into
# dist/openchat-<platform>-<version>.zip and removes the stage dirs
# (and fyne-cross's own scratch dir) afterwards.
zip_and_cleanup() {
  echo "==> zipping up final distributions..."
  local platform
  for platform in macos ios windows linux android; do
    local dir="$DIST_DIR/stage-$platform"
    if [ -d "$dir" ] && [ -n "$(ls -A "$dir" 2>/dev/null)" ]; then
      ( cd "$dir" && zip -qr -X "$DIST_DIR/openchat-$platform-$APP_VERSION.zip" . )
      echo "  - openchat-$platform-$APP_VERSION.zip"
    fi
  done
  rm -rf "$DIST_DIR/stage-macos" "$DIST_DIR/stage-windows" "$DIST_DIR/stage-linux" "$DIST_DIR/stage-android" "$DIST_DIR/stage-ios" "$DIST_DIR/fyne-cross"
  echo
  echo "Done. Generated files are in $DIST_DIR/"
}
