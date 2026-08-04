// Package mobile is the gomobile-bindable facade over OpenChat's core:
// wallet derivation, contact codes, and the resilient network session
// (connect/send/listen/history-sync) from pkg/client and pkg/crypto.
// Bind it with:
//
//	cd app/golang
//	gomobile bind -target=ios,iossimulator,macos -o OpenChatMobile.xcframework ./mobile
//
// (needs gomobile installed — `go install golang.org/x/mobile/cmd/gomobile@latest`
// — and, for the iOS/macOS targets, a Mac with Xcode; this cannot be
// produced from this sandbox). The output name matters: gomobile derives
// the generated Swift module name from it, and app/xcode's LiveCore.swift
// / CoreProvider.swift both gate on `canImport(OpenChatMobile)` — keep
// this exact name unless you also update those two `#if canImport`
// lines. The resulting .xcframework is a normal binary framework
// dependency for an Xcode project; see app/xcode/README.md for how it's
// wired into the app.
//
// # Why this package exists, separate from pkg/client
//
// gomobile bind only supports a constrained type surface in exported
// signatures: string, bool, the sized numeric types (but NOT unsigned
// ones besides byte), []byte, error, and named struct/interface types
// declared in a bound package. No fixed-size arrays ([32]byte), no maps,
// no slices of structs, no generics, no variadic functions. pkg/client's
// actual API (e.g. SendMessage's `toX25519Pub [32]byte`, FetchHistory's
// `map[string][32]byte`) doesn't fit that, and shouldn't be contorted to
// — it's also used unmodified by cmd/client (a normal Go binary) and
// app/fyne (a normal Go module), where those aren't constraints at all.
// This package is the thin adapter layer that exists only for the third
// consumer, native mobile/desktop shells via gomobile, converting to/from
// bindable types (hex strings for keys, JSON strings for bulk/structured
// data, callback interfaces instead of Go func values) at the boundary.
//
// # What's in scope here, and what isn't
//
// In scope: everything that needs the E2EE scheme or talks to the
// network — wallet creation/import, contact codes, connecting to a
// gateway with failover, sending, live listening, and chain-history sync.
// All of it is exactly pkg/client/pkg/crypto's existing, already-used-in-
// production-by-the-CLI-and-Fyne-app logic; nothing is reimplemented here.
//
// Deliberately NOT in scope, left to the native shell instead:
//
//   - At-rest storage of the mnemonic. app/fyne encrypts it into a local
//     JSON file with a PIN-derived key (see app/fyne/internal/app/
//     wallet_store.go) because Fyne has no better cross-platform option.
//     A real iOS/macOS app has a better one — the system Keychain, with
//     hardware-backed encryption and optional biometric gating — and
//     should use it instead of asking this package to reimplement
//     Fyne's PIN+scrypt scheme a third time. This package only ever
//     holds keys in process memory, exactly like pkg/client itself.
//   - Chat history / contact list persistence. Also app/fyne-specific
//     (internal/app/chat_store.go, a JSON file via Fyne's storage API).
//     A native app should use whatever's idiomatic there (SwiftData,
//     Core Data, or just a JSON file in the app's own sandbox) — this
//     package only ever hands back what it decrypted/sent in a given
//     call; it doesn't remember it.
//
// The dividing line, in one sentence: this package is "the network and
// the cryptography"; the native shell is "the UI and everything that
// needs to survive a relaunch."
package mobile
