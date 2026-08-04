# app/xcode — native macOS/iOS app

SwiftUI implementation of the OpenChat client for macOS and iOS, built
from [`docs/design-spec.md`](../../docs/design-spec.md) and
[`docs/design-code.md`](../../docs/design-code.md), replacing the
Fyne-based GUI (`../fyne/cmd/app`) on these two platforms specifically.
Windows/Linux/Android keep using the Fyne app for now; nothing here
affects them.

The Xcode project exists and builds and runs (onboarding, chat list,
conversation, and all dialogs are wired to the real Go core). This
document describes what's here and how to keep it building after you
change either the Go side or the Swift side.

## What's here

```
app/xcode/
├── Package.swift                     OpenChatKit — a Swift package, not the app itself
├── Sources/OpenChatKit/
│   ├── Theme/                        Design tokens (color/type/spacing) from design-spec.md §1
│   ├── Components/                   Shared UI: buttons, PIN keypad, avatar, message bubble, ...
│   ├── Models/                       Contact, StoredMessage
│   ├── Core/                         OpenChatCore protocol + MockCore + LiveCore (gomobile) + provider
│   ├── Storage/                      KeychainWalletStore (wallet), ChatStore (contacts/history)
│   ├── ViewModels/                   AppModel, SessionModel, OnboardingModel
│   ├── Onboarding/                   A1–A5 screens + OnboardingCoordinator
│   ├── Main/                         B1/B2 chat list + conversation + RootView
│   └── Dialogs/                      B3–B7: Add/Edit contact, My code, Network settings, Log out
├── AppTargets/                       Historical reference only — see note below, not part of the build
├── OpenChatMobile.xcframework/       Legacy copy, unused by the build — see prepare-project.sh's comments
├── scripts/
│   └── prepare-project.sh            Rebuild OpenChatMobile.xcframework + verify OpenChatKit — run this
│                                      after changing app/golang/mobile or Sources/OpenChatKit
└── app/                              The actual Xcode project
    ├── app.xcodeproj/                Single unified target, builds for iOS/iPadOS + macOS
    ├── OpenChatMobile.xcframework/   The framework the project actually links (see below)
    └── app/
        ├── appApp.swift              @main entry point — window geometry only, all UI is RootView
        ├── app.entitlements          App Sandbox + network client + user-selected file read
        ├── Assets.xcassets/          App icon + accent color
        └── ContentView.swift         Unused Xcode-template leftover (RootView is what's actually shown)
```

`OpenChatKit` is intentionally *only* a library — everything platform-
UI-visible is here, screen-complete against design-spec.md §3 (A1–A5,
B1–B7). `app/app.xcodeproj` is the real, buildable Xcode project that
depends on it as a local Swift package.

## Architecture: where Swift ends and Go begins

- **Go core** (`app/golang/mobile`, bound via `gomobile bind` into
  `OpenChatMobile.xcframework`): wallet derivation, contact codes, and
  the network session (connect/send/listen/history-sync) — the same
  `pkg/client`/`pkg/crypto` logic already used by the CLI and the Fyne
  app, just behind a gomobile-bindable facade. See
  [`app/golang/mobile/doc.go`](../golang/mobile/doc.go) for the exact
  bind command and the reasoning for the split.
- **Swift shell** (this package): everything UI, plus the two things
  deliberately *not* pushed into Go — `KeychainWalletStore` (the
  recovery phrase, PIN-encrypted, in the system Keychain) and
  `ChatStore` (contacts/message history, a local JSON file). Both
  replace what the Fyne app does with its own file-based storage, with
  a more native/secure option instead of reimplementing Fyne's scheme
  a third time.
- **`OpenChatCore`** (`Core/OpenChatCore.swift`) is the seam between
  them: a protocol with two implementations — `MockCore` (in-memory, no
  network, used in SwiftUI previews and if the xcframework isn't
  embedded) and `LiveCore` (the real gomobile wrapper, compiled in only
  when `OpenChatMobile` can be imported). Nothing above `Core/` ever
  knows which one it's talking to.

## Diagnosing chat-update issues

`Core/Log.swift` defines two `os.Logger` categories (`session`,
`chatstore`), both under subsystem `network.openchat.app`, covering the
whole connect → history-sync → live-listen → `ChatStore` pipeline:
connection/sync/listen starting and finishing, every message stored (or
a history one skipped as an already-known duplicate), every status
change, every persisted-file load/write. Message *text* is deliberately
never logged, only metadata (address, tx hash, byte counts) — these are
end-to-end encrypted messages.

`Log.session` also carries `[go]`-prefixed lines forwarded from the Go
core itself (`LiveCore.swift`'s `LiveLogListener`, wired to
`mobile.LogListener`/`pkg/client`'s `Discovery`/`Client.OnEvent`) — this
is the only place the *per-candidate* gateway dial detail shows up
(which address, in what order, and why each one failed: DNS failure,
connection refused, TLS handshake, timeout). A bare Swift-side
`"failover exhausted candidates: context deadline exceeded"` only says
every candidate failed within the deadline; the `[go]` lines explain
which candidates were tried and why — the first thing to check if
history sync or sending is failing and it's not obvious whether that's
a real server-side outage or a client misconfiguration (wrong bootstrap
address in Network Settings, for instance).

To view them: Xcode's own console while running from Xcode, or from a
terminal —

```
log stream --predicate 'subsystem == "network.openchat.app"'
```

(works against the Simulator or a connected device; Console.app's
search field also accepts the same subsystem string).

## Building and running

Once the project exists (it already does — see `app/app.xcodeproj`),
day-to-day this is just:

1. If you changed `app/golang/mobile` (or anything it pulls in from
   `pkg/client`/`pkg/crypto`) or anything under `Sources/OpenChatKit`,
   run the prep script first:
   ```
   ./scripts/prepare-project.sh
   ```
   This regenerates `app/OpenChatMobile.xcframework` from the current Go
   source and runs `swift build` against `OpenChatKit` so Swift-side
   breakage shows up on the command line instead of only inside Xcode.
   See the script's own header comment for flags (`--skip-mobile`,
   `--skip-swift`) to run just one half.
2. Open `app/app.xcodeproj` in Xcode.
3. Pick a run destination (My Mac, or an iOS Simulator/device) and hit
   Run. The project is a single unified multiplatform target — one
   scheme builds for both.

If `gomobile bind` produced a different exported API shape than before
(new/renamed/reshaped methods on `MobileWallet`/`MobileSession`), Xcode
will show the new compile errors directly in
`Sources/OpenChatKit/Core/LiveCore.swift` — its own header comment
documents gomobile's Swift-mapping rules (Mobile-prefixed types,
`NSError**` auto-bridging to `throws` on instance methods, the one
exception found so far being `sendText`) as a starting point for fixing
them. Nothing else in the app needs to change either way, since
everything above `Core/` only ever talks to the `OpenChatCore` protocol.

### Project configuration notes (in case you're ever rebuilding `app.xcodeproj` from scratch)

These are hand-set in `project.pbxproj` and won't regenerate themselves
if the project is ever recreated from an Xcode template:

- **Deployment targets:** iOS 17.0 / macOS 14.0 (matches
  `Package.swift`'s `platforms:`). Xcode's default for a new project is
  whatever the latest SDK is — set these explicitly or the app will
  silently require a much newer OS than it needs to.
- **Supported platforms / device family:** iOS + iPadOS + macOS only —
  deliberately **not** visionOS. Including `xros`/`xrsimulator` produced
  a `cannot link directly with 'SwiftUICore'... not an allowed client of
  it` linker error, tied to `OpenChatKit`'s `Package.swift` not
  declaring visionOS support.
- **`OTHER_LDFLAGS = "-lresolv"`:** without it, linking fails with
  `Undefined symbol: _res_9_ninit` (and `_nclose`/`_nsearch`) — Go's
  net/DNS resolver code needs `libresolv` linked into the host app; it
  isn't pulled in automatically just by embedding the xcframework.
- **`app.entitlements`** (App Sandbox on): `com.apple.security.app-sandbox`,
  `com.apple.security.network.client` (required — without it a sandboxed
  macOS app can't make outbound connections at all, which silently
  breaks networking with no obvious error), and
  `com.apple.security.files.user-selected.read-only` (for B6's CA
  certificate file picker).
- **Embedding the framework:** `OpenChatMobile.xcframework` needs
  "Embed & Sign" in the target's *Frameworks, Libraries, and Embedded
  Content* — a plain "Do Not Embed" link will build but crash at launch
  with a dyld "image not found" error.
- **App icon / accent color:** `Assets.xcassets/AppIcon.appiconset` and
  `AccentColor.colorset` are already populated (not the Xcode-default
  blank/blue placeholder) — if you regenerate the asset catalog from a
  template, both need their `Contents.json` filled in explicitly with
  real filenames/colors or Xcode renders a blue placeholder icon.

### About `AppTargets/` and the legacy top-level `OpenChatMobile.xcframework/`

`AppTargets/` was written before the real Xcode project existed, as a
plan for *two* separate single-platform targets (macOS App + iOS App)
sharing one workspace. The project that actually got built instead uses
Xcode 16's unified multiplatform single-target format, and
`app/app/appApp.swift` is the merged, adapted version of what
`AppTargets/macOS/OpenChatApp.swift` sketched out. `AppTargets/` is kept
for historical reference but nothing copies from it anymore — don't
follow its old "two targets" framing if you're rebuilding the project.

Similarly, `app/xcode/OpenChatMobile.xcframework/` (top-level, as
opposed to `app/xcode/app/OpenChatMobile.xcframework/`) predates the
`app/app.xcodeproj` layout and isn't read by anything in the build
graph anymore. `scripts/prepare-project.sh` still refreshes it (so it
can't silently drift into something misleading) but if you're cleaning
house it's safe to delete.

## Status

Screen-complete against design-spec.md §3: onboarding (A1–A5), main app
(B1/B2), and all five dialogs (B3–B7) are implemented in
`Sources/OpenChatKit` and wired end-to-end through `RootView` against
the real Go core via `LiveCore`. The macOS and iOS UI/UX bug passes
against a real running build are done (field styling, sheet sizing,
sidebar toggle, keyboard input, PIN dot sizing, iOS chat navigation,
composer sizing, conversation header, CA-certificate picker) and
`docs/design-spec.md`/`docs/design-code.md` are kept in sync with the
current behavior. Real app icon and accent color are in place.
