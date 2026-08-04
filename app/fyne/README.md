# app/fyne — OpenChat GUI (Fyne, cross-platform)

The current cross-platform desktop/mobile messenger UI: one Go codebase,
built natively for macOS, Windows, Linux, iOS and Android via
[Fyne](https://fyne.io) (no Electron/webview, no per-platform rewrite).

This is a **separate Go module** from the backend (`../golang`), on
purpose: Fyne's dependency tree (OpenGL bindings, mobile toolchain
support, systray, image codecs, etc.) is large and cgo-heavy, and none of
it has any business being in the validator/relay node's `go.sum` or its
Docker build context. This module depends on `../golang` only for its
two genuinely shared packages — `pkg/client` and `pkg/crypto` — via a
`replace` directive in `go.mod`, and nothing in `../golang` depends on
this module at all. See the repo root [README.md](../../README.md) for
how the two fit together, and
[`go.work`](../../go.work) for the workspace file that lets you edit both
modules together locally without publishing either one.

This module (and Fyne generally) is expected to become **deprecated** as
native shells land per platform — see [`../xcode`](../xcode) for the
first one, macOS/iOS. Until then, this is the one actually-working GUI,
and stays fully buildable.

## What's here

```
cmd/app/           entrypoint — onboarding, chat list/conversation, contacts, settings
internal/app/      local storage + service glue (wallet, contacts, message history)
scripts/           build/package this module for every platform, bundling
                    the CLI client + keytool from ../golang alongside it
```

Both `cmd/app` and `internal/app` reuse `../golang`'s `pkg/client` and
`pkg/crypto` unmodified — see `../golang/README.md`'s "Part 2 — client &
interaction" for what those actually do (key derivation, E2EE,
discovery/failover, message sync). Nothing under `../golang` was rewritten
or forked for this app; the GUI is purely a UI + local-storage layer on
top of the same client library the CLI (`../golang/cmd/client`) uses.

- **Onboarding** (`cmd/app/onboarding.go`): create a new identity (shows
  the 24-word recovery phrase once, for backup) or import an existing one,
  then set a local PIN.
- **Wallet storage** (`internal/app/wallet_store.go`): the mnemonic is
  encrypted at rest with a key derived from that PIN via `scrypt` +
  AES-256-GCM, written through Fyne's cross-platform storage API (works
  identically in a macOS/Windows app-support folder and an iOS/Android app
  sandbox — raw `os.UserConfigDir()` isn't reliably usable on mobile).
- **Contacts** (`internal/app/contactcode.go`): since a recipient needs
  both your address *and* X25519 key, the app combines them into one
  shareable `oc1:...` code (`Service.MyContactCode`) instead of making
  users copy two separate hex strings.
- **Chat list & conversation view** (`cmd/app/mainscreen.go`): local
  message history (`internal/app/chat_store.go`) plus a live
  `StreamIncomingSMS` subscription; incoming messages update the UI via
  `fyne.Do` (Fyne's thread-safe main-goroutine dispatch, since the stream
  read happens on a background goroutine).
- **History sync** (`internal/app/service.go`'s `SyncHistory`): on launch,
  before the live listener takes over, replays `pkg/client.FetchHistory`
  from the last checkpoint height so messages sent while the app was
  closed aren't lost.
- **Free to send:** there's no token, fee, or gas — sending costs nothing
  beyond the recipient having their gateway reachable (see the backend's
  mempool rate-limiting in `../golang/README.md` Part 1 for how spam is
  bounded instead).
- **Network settings** (`cmd/app/settings.go`): a small dialog (gear icon)
  to point the app at your own gateways instead of the placeholder
  `DefaultBootstrapGateways`, e.g. `localhost:9091,localhost:9092` against
  the `cicd/docker/docker-compose.yml` demo, with an optional CA cert /
  insecure-TLS toggle for self-signed certs during local testing.

For the exact functional breakdown of every screen/dialog (useful for
designing the native replacement), see
[../../docs/design-code.md](../../docs/design-code.md).

## Building it

Install the Fyne CLI once (needs your platform's normal cgo toolchain —
Xcode command line tools on macOS, a C compiler + graphics headers on
Linux, etc., since Fyne's OpenGL backend uses cgo):

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
go mod tidy
```

**Easiest path — one script per host OS, builds everything (this GUI app
for every platform it can, plus the CLI client + keytool from
`../golang`, all zipped up in `dist/`):**

```bash
./scripts/source-macos.sh     # macOS only: also builds iOS, cross-builds Windows/Linux/Android
./scripts/source-linux.sh     # Linux/Windows: builds everything except macOS/iOS
./scripts/source-windows.sh
```

Output lands in `dist/` as `openchat-<platform>-<version>.zip`. Pass
`--client-only` for just the CLI client + keytool (no GUI, no Fyne
toolchain needed at all — this mode only touches `../golang`), or e.g.
`--macos --ios` to limit which GUI platforms get built.
(`scripts/build-client.sh` still works as an alias for
`scripts/source-macos.sh`.)

**Manual, platform by platform**, if you'd rather call `fyne`/`fyne-cross`
yourself:

**macOS** (run on a Mac):

```bash
cd cmd/app
fyne package -os darwin -icon Icon.png
# -> OpenChat.app, drag into /Applications or notarize for distribution
```

**Windows** (run natively on Windows, or cross-compile from macOS/Linux
with `mingw-w64` installed):

```bash
cd cmd/app
fyne package -os windows -icon Icon.png
# -> OpenChat.exe
```

**Android** (from any OS, needs the Android SDK + NDK):

```bash
cd cmd/app
fyne package -os android -appID network.openchat.app -icon Icon.png
# -> OpenChat.apk
```

**iOS** (must run on macOS with Xcode + an Apple Developer account for
signing):

```bash
cd cmd/app
fyne package -os ios -appID network.openchat.app -icon Icon.png
# -> OpenChat.app / .ipa via Xcode's archive/export step
```

**Cross-compiling via Docker instead of installing every SDK locally:**
[`fyne-cross`](https://github.com/fyne-io/fyne-cross) runs each target's
build in a matching container:

```bash
go install github.com/fyne-io/fyne-cross@latest
fyne-cross darwin -arch=amd64,arm64 -app-id network.openchat.app ./cmd/app
fyne-cross windows -arch=amd64 -app-id network.openchat.app ./cmd/app
fyne-cross android -app-id network.openchat.app ./cmd/app
fyne-cross ios -app-id network.openchat.app ./cmd/app   # still needs a Mac for signing
```

`cmd/app/FyneApp.toml` carries the app ID/name/icon metadata both `fyne
package` and `fyne-cross` read by default; `cmd/app/Icon.png` is a
placeholder — swap it for real artwork before shipping.

## Honest caveats

- This module's `go.sum` doesn't exist yet in this reorganized layout —
  run `go mod tidy` here once (needs module-proxy network access) before
  building; see `go.mod`'s comment.
- Needs `fyne.Do`, added in Fyne v2.5 — `go.mod` pins `fyne.io/fyne/v2
  v2.6.0`; don't downgrade below v2.5 without replacing the `fyne.Do`
  calls in `cmd/app/mainscreen.go` with your own main-goroutine dispatch.
- Fyne's desktop backend uses cgo/OpenGL, so building needs a real C
  toolchain (Xcode CLT / a Linux C compiler + Mesa dev headers / MSVC or
  mingw on Windows).
- `cmd/app/Icon.png` is a placeholder generated for the reference
  implementation — replace it before shipping to a store.
- The relay-node/history-sync features on the backend side
  (`../golang`'s `NODE_ROLE=relay`, `FetchHistory`) are newer and less
  exercised than the rest — see `../golang/README.md`'s own "honest
  caveats" for what that means for this app's history-sync behavior.
