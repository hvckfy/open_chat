import Foundation

/// One already-decrypted message kept in local history — the Swift-side
/// counterpart of the Go backend's `internal/app.StoredMessage`. Only
/// plaintext already seen is ever stored here; no ciphertext or key
/// material crosses into this type.
struct StoredMessage: Identifiable, Codable, Hashable {
    enum Direction: String, Codable {
        case incoming
        case outgoing
    }

    /// The chain transaction hash when known (live/history-synced
    /// messages always have one — it's also how a history sync dedupes
    /// against what's already stored); a client-generated UUID string
    /// for anything that doesn't have one yet in principle.
    var id: String
    var direction: Direction
    var text: String
    var timestampMs: Int64

    var date: Date { Date(timeIntervalSince1970: Double(timestampMs) / 1000) }
    var isOutgoing: Bool { direction == .outgoing }
}
