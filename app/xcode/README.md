# app/xcode — native macOS/iOS app (not started)

This directory is a placeholder for a native Xcode project (Swift/
SwiftUI) implementing the OpenChat client for macOS and iOS, replacing
the current Fyne-based GUI (`../golang/cmd/app`) on those two platforms
specifically. Windows/Linux/Android keep using the Fyne app for now;
nothing here affects them.

## Why a separate native app

The Fyne app works everywhere but looks/feels native nowhere — it's a
single Go-rendered UI on every platform. The goal is: keep the Go module
(`../golang`) as the shared network/protocol/crypto core, and give each
OS a real native shell on top of it (Xcode here; something else for
Android later, "и так далее").

## What has to happen before code goes here

1. **Design.** [`../../docs/design-code.md`](../../docs/design-code.md)
   is the functional spec (what every screen/dialog needs to do); it's
   meant to be run through claude-design to produce the actual visual
   design system and page templates. That output is the input to this
   app's UI work.
2. **Decide how Swift talks to the Go core.** Not decided yet — three
   real options, each with a different amount of up-front work:
   - **gomobile bindings**: compile `app/golang/pkg/client` +
     `pkg/crypto` into an `.xcframework` via `gomobile bind`, call it
     directly from Swift. Reuses the exact same protocol/crypto code
     that's already written and (per `app/golang/README.md`'s "honest
     caveats") the least battle-tested part of this project — no
     reimplementation, no risk of the Swift and Go sides silently
     drifting apart.
   - **Embedded local daemon**: ship the `openchat-node`/client binary
     (or a small purpose-built local daemon) alongside the app, talk to
     it over localhost gRPC/a Unix socket from Swift. More moving parts,
     but keeps Swift code network/crypto-agnostic.
   - **Native reimplementation**: a Swift gRPC client talking directly to
     `SendSMS`/`StreamIncomingSMS`/etc., with the BIP-39/E2EE scheme
     (`app/golang/pkg/crypto`) reimplemented in Swift. Fastest to start
     with familiar native tooling, but duplicates security-sensitive
     code in two languages that then have to be kept in sync by hand.

   gomobile is the default recommendation (least duplication of
   security-critical code) but this should be confirmed before real work
   starts, since it constrains the whole app's architecture.
3. **Apple Developer account + signing setup**, same as already noted for
   the Fyne iOS build in `app/golang/README.md`.

## Status

Empty. Once the design templates above exist, the next step is scaffolding
an Xcode project here (workspace + target for macOS, target for iOS,
shared SwiftUI views where the platforms' UX doesn't diverge) and picking
one of the three integration options above.
