#!/usr/bin/env bash
#
# app/xcode/scripts/prepare-project.sh — bring the Xcode project's Go
# dependency up to date with app/golang/mobile, and sanity-check the
# Swift side (OpenChatKit) still builds. Run this any time either half
# of the bridge changes:
#
#   - Go core changed (app/golang/mobile/*.go, or anything it pulls in
#     from pkg/client / pkg/crypto): re-run this to regenerate
#     OpenChatMobile.xcframework so Xcode picks up the new API.
#   - OpenChatKit changed (anything under app/xcode/Sources/OpenChatKit):
#     re-run this (or just pass --skip-mobile) to catch Swift compile
#     errors from the command line before opening Xcode.
#
# This does NOT create the .xcodeproj itself (that's a one-time, do-it-
# in-Xcode step — see ../README.md "First-time setup"). It only keeps
# the generated framework and the Swift package healthy after that.
#
# Usage:
#   ./scripts/prepare-project.sh                # regenerate the xcframework + verify OpenChatKit builds
#   ./scripts/prepare-project.sh --skip-mobile   # OpenChatKit check only, don't touch gomobile/Go
#   ./scripts/prepare-project.sh --skip-swift    # xcframework only, don't run `swift build`
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
XCODE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$XCODE_DIR/.." && cd .. && pwd)"
GOLANG_DIR="$REPO_ROOT/app/golang"

# The Xcode project (app/xcode/app/app.xcodeproj) references the
# framework by the relative path "OpenChatMobile.xcframework" from its
# own directory (see app/xcode/app/app.xcodeproj/project.pbxproj —
# PBXFileReference with sourceTree "<group>"), so this is the one
# location that actually matters to a real Xcode build.
CANONICAL_XCFRAMEWORK="$XCODE_DIR/app/OpenChatMobile.xcframework"
# A second copy lives at the top of app/xcode/ from early setup, before
# the app/app.xcodeproj layout existed. Nothing in the build graph reads
# it anymore (Package.swift never links a binary framework — LiveCore.swift
# only `#if canImport(OpenChatMobile)`s against whatever the app target
# embeds) but it's kept in sync here too so it doesn't silently go stale
# and mislead the next person who finds it.
LEGACY_XCFRAMEWORK="$XCODE_DIR/OpenChatMobile.xcframework"

SKIP_MOBILE=0
SKIP_SWIFT=0
for arg in "$@"; do
  case "$arg" in
    --skip-mobile) SKIP_MOBILE=1 ;;
    --skip-swift) SKIP_SWIFT=1 ;;
    -h|--help)
      sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg' (see --help)" >&2
      exit 1
      ;;
  esac
done

if [ "$(uname -s)" != "Darwin" ]; then
  echo "error: this must run on a Mac — gomobile's ios/macos targets and \`swift build\` both need a real Xcode install." >&2
  exit 1
fi

if [ "$SKIP_MOBILE" -eq 0 ]; then
  if ! xcode-select -p >/dev/null 2>&1; then
    echo "error: Xcode Command Line Tools not found. Run: xcode-select --install" >&2
    exit 1
  fi

  if ! command -v go >/dev/null 2>&1; then
    echo "error: Go toolchain not found (need it to run gomobile). Install from https://go.dev/dl/" >&2
    exit 1
  fi

  # `go install ...@latest` below drops binaries under $(go env GOPATH)/bin,
  # which a fresh shell/CI runner won't have on PATH by default. Export it
  # for the rest of this script now, before installing anything — not just
  # so this script's own `command -v` checks find what it just installed,
  # but because `gomobile bind` itself execs a separate `gobind` binary as
  # a subprocess, found via this same inherited PATH (not via gomobile's
  # own location or $GOPATH directly) — see the gobind block below.
  export PATH="$PATH:$(go env GOPATH)/bin"

  if ! command -v gomobile >/dev/null 2>&1; then
    echo "==> gomobile not found, installing (golang.org/x/mobile/cmd/gomobile@latest)..."
    go install golang.org/x/mobile/cmd/gomobile@latest
  fi
  GOMOBILE_BIN="$(command -v gomobile || echo "$(go env GOPATH)/bin/gomobile")"
  if [ ! -x "$GOMOBILE_BIN" ]; then
    echo "error: gomobile installed but not found on PATH or in \$(go env GOPATH)/bin — add \$(go env GOPATH)/bin to PATH and re-run." >&2
    exit 1
  fi

  # `gomobile init` does NOT install gobind, despite what its own error
  # message ("gobind was not found. Please run gomobile init before
  # trying again") implies — that message is misleading/wrong. gobind is
  # a separate cmd that `gomobile bind` shells out to at run time, and it
  # has to be `go install`ed on its own.
  if ! command -v gobind >/dev/null 2>&1; then
    echo "==> gobind not found, installing (golang.org/x/mobile/cmd/gobind@latest)..."
    go install golang.org/x/mobile/cmd/gobind@latest
  fi
  if ! command -v gobind >/dev/null 2>&1; then
    echo "error: gobind installed but still not found on PATH — add \$(go env GOPATH)/bin to PATH and re-run." >&2
    exit 1
  fi

  echo "==> gomobile init (idempotent, safe to re-run)"
  "$GOMOBILE_BIN" init

  BUILD_TMPDIR="$(mktemp -d)"
  trap 'rm -rf "$BUILD_TMPDIR"' EXIT

  echo "==> gomobile bind (this takes a few minutes — it's a full cross-compile of the Go core for ios, iossimulator and macos)"
  (
    cd "$GOLANG_DIR"
    "$GOMOBILE_BIN" bind -target=ios,iossimulator,macos -o "$BUILD_TMPDIR/OpenChatMobile.xcframework" ./mobile
  )

  echo "==> installing rebuilt framework into $CANONICAL_XCFRAMEWORK"
  rm -rf "$CANONICAL_XCFRAMEWORK"
  mkdir -p "$(dirname "$CANONICAL_XCFRAMEWORK")"
  cp -R "$BUILD_TMPDIR/OpenChatMobile.xcframework" "$CANONICAL_XCFRAMEWORK"

  echo "==> syncing legacy copy at $LEGACY_XCFRAMEWORK (unused by the build, kept only so it doesn't drift/mislead)"
  rm -rf "$LEGACY_XCFRAMEWORK"
  cp -R "$BUILD_TMPDIR/OpenChatMobile.xcframework" "$LEGACY_XCFRAMEWORK"

  echo "==> done: OpenChatMobile.xcframework regenerated from app/golang/mobile"
  echo "    If any exported Go type/method signature changed, Xcode may now show new compile"
  echo "    errors in Sources/OpenChatKit/Core/LiveCore.swift — that file's own header comment"
  echo "    explains gomobile's Swift-mapping rules for fixing them."
else
  echo "==> --skip-mobile set, leaving OpenChatMobile.xcframework as-is"
fi

if [ "$SKIP_SWIFT" -eq 0 ]; then
  if ! command -v swift >/dev/null 2>&1; then
    echo "warning: 'swift' not found on PATH, skipping OpenChatKit build check (install Xcode / Command Line Tools to enable this)" >&2
  else
    echo "==> swift build (compiles OpenChatKit standalone, against MockCore — this is the fast way to catch Swift errors without opening Xcode)"
    (
      cd "$XCODE_DIR"
      swift build
    )
    echo "==> OpenChatKit builds cleanly"
  fi
else
  echo "==> --skip-swift set, not running 'swift build'"
fi

echo ""
echo "Next: open app/xcode/app/app.xcodeproj in Xcode and build/run as usual."
echo "(This script never touches the .xcodeproj itself — only the generated framework and the Swift package.)"
