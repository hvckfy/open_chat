import Foundation

/// One saved conversation partner — the Swift-side counterpart of the Go
/// backend's `internal/app.Contact` (app/fyne), except here *this* app
/// owns persistence (see the mobile package's doc comment on why: a
/// native app should use its own idiomatic local storage, not a
/// reimplementation of Fyne's). `Codable` so a native shell can persist
/// it however it likes (SwiftData, a JSON file, etc.) — persistence
/// itself is deliberately not part of this package.
struct Contact: Identifiable, Codable, Hashable {
    var address: String
    var x25519Hex: String
    var displayName: String

    var id: String { address }

    /// Falls back to a shortened address (design-code.md's "address-as-
    /// identity" convention) when no real name has been set yet — the
    /// same placeholder a first, unprompted message gets before the
    /// user renames the contact (see B4 Edit contact).
    static func placeholderName(for address: String) -> String {
        guard address.count > 10 else { return address }
        let start = address.prefix(6)
        let end = address.suffix(4)
        return "0x\(start)…\(end)"
    }
}
