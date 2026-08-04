import Foundation

/// The local (device-only) address book + message history — the Swift-
/// side counterpart of the Go backend's `internal/app.ChatStore`
/// (app/fyne), reimplemented here as plain local storage since the
/// mobile core deliberately doesn't cover persistence (see app/golang/
/// mobile's package doc comment). A single JSON file in the app's
/// Application Support directory; every mutating call persists
/// immediately so a killed app never loses a message.
///
/// `@MainActor` + `ObservableObject`: every screen (B1's list, the
/// conversation view) observes this directly rather than going through
/// a separate view-model layer for what's fundamentally just state.
@MainActor
final class ChatStore: ObservableObject {
    @Published private(set) var contacts: [Contact] = []
    @Published private(set) var messagesByAddress: [String: [StoredMessage]] = [:]
    @Published private(set) var syncedHeight: Int64 = 0

    private struct FileFormat: Codable {
        var contacts: [Contact]
        var messages: [String: [StoredMessage]]
        var syncedHeight: Int64
    }

    private let fileURL: URL

    init(fileName: String = "chats.v1.json") {
        let dir = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        self.fileURL = dir.appendingPathComponent(fileName)
        load()
    }

    private func load() {
        guard let data = try? Data(contentsOf: fileURL) else {
            Log.chatStore.info("load(): no existing file at \(self.fileURL.path, privacy: .public) — starting empty")
            return
        }
        guard let decoded = try? JSONDecoder().decode(FileFormat.self, from: data) else {
            Log.chatStore.error("load(): file exists at \(self.fileURL.path, privacy: .public) but failed to decode (\(data.count, privacy: .public) byte(s)) — starting empty, existing data NOT overwritten until the next persist()")
            return
        }
        contacts = decoded.contacts
        messagesByAddress = decoded.messages
        syncedHeight = decoded.syncedHeight
        let totalMessages = decoded.messages.values.reduce(0) { $0 + $1.count }
        Log.chatStore.info("load(): loaded \(decoded.contacts.count, privacy: .public) contact(s), \(totalMessages, privacy: .public) message(s) total, syncedHeight=\(decoded.syncedHeight, privacy: .public)")
    }

    private func persist() {
        let snapshot = FileFormat(contacts: contacts, messages: messagesByAddress, syncedHeight: syncedHeight)
        guard let data = try? JSONEncoder().encode(snapshot) else {
            Log.chatStore.error("persist(): failed to encode snapshot — nothing written")
            return
        }
        do {
            try data.write(to: fileURL, options: .atomic)
        } catch {
            Log.chatStore.error("persist(): failed to write to \(self.fileURL.path, privacy: .public): \(String(describing: error), privacy: .public)")
        }
    }

    func contact(for address: String) -> Contact? {
        contacts.first { $0.address == address }
    }

    func upsertContact(_ contact: Contact) {
        let isNew = !contacts.contains { $0.address == contact.address }
        if let i = contacts.firstIndex(where: { $0.address == contact.address }) {
            contacts[i] = contact
        } else {
            contacts.append(contact)
        }
        if messagesByAddress[contact.address] == nil {
            messagesByAddress[contact.address] = []
        }
        persist()
        Log.chatStore.info("upsertContact: \(isNew ? "added" : "updated", privacy: .public) \(contact.address, privacy: .public)")
    }

    func messages(for address: String) -> [StoredMessage] {
        messagesByAddress[address] ?? []
    }

    /// Records a new message (sent or received), auto-creating a
    /// placeholder contact if one doesn't exist yet. `senderX25519Hex`
    /// is the encryption key the *other side* used — pass "" for
    /// outgoing messages; for an incoming one it's the sender's real
    /// X25519 key (see the Go core's `IncomingMessagePayload`), which
    /// lets a reply go out immediately without a separate contact-code
    /// exchange the first time a stranger messages this wallet.
    func appendMessage(address: String, message: StoredMessage, senderX25519Hex: String) {
        ensureContact(address: address, x25519Hex: senderX25519Hex)
        messagesByAddress[address, default: []].append(message)
        persist()
        Log.chatStore.info("appendMessage: \(message.direction == .incoming ? "incoming" : "outgoing", privacy: .public) for \(address, privacy: .public), id=\(message.id, privacy: .public) — now \(self.messagesByAddress[address]?.count ?? 0, privacy: .public) message(s) in this conversation")
    }

    /// `AppendMessage`'s counterpart for a message recovered by a
    /// history scan rather than a live send/receive — dedupes by
    /// message id (tx hash) since a resumed sync can overlap what's
    /// already stored. Returns whether it was actually added.
    @discardableResult
    func appendHistoryMessage(address: String, message: StoredMessage, peerX25519Hex: String) -> Bool {
        if messagesByAddress[address]?.contains(where: { $0.id == message.id }) == true {
            Log.chatStore.debug("appendHistoryMessage: skipping already-known id=\(message.id, privacy: .public) for \(address, privacy: .public)")
            return false
        }
        ensureContact(address: address, x25519Hex: peerX25519Hex)
        messagesByAddress[address, default: []].append(message)
        persist()
        Log.chatStore.info("appendHistoryMessage: stored \(message.direction == .incoming ? "incoming" : "outgoing", privacy: .public) for \(address, privacy: .public), id=\(message.id, privacy: .public)")
        return true
    }

    private func ensureContact(address: String, x25519Hex: String) {
        if let i = contacts.firstIndex(where: { $0.address == address }) {
            if !x25519Hex.isEmpty && contacts[i].x25519Hex.isEmpty {
                contacts[i].x25519Hex = x25519Hex
            }
        } else {
            contacts.append(Contact(address: address, x25519Hex: x25519Hex, displayName: Contact.placeholderName(for: address)))
        }
    }

    func setSyncedHeight(_ height: Int64) {
        syncedHeight = height
        persist()
    }

    /// Erases everything — contacts, messages, sync checkpoint. The
    /// local-storage half of "Log out" (design-code.md B7); pair with
    /// `KeychainWalletStore.delete()` for the wallet half.
    func wipe() {
        contacts = []
        messagesByAddress = [:]
        syncedHeight = 0
        try? FileManager.default.removeItem(at: fileURL)
    }
}
