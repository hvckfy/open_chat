import Foundation

/// The main app's live state once a wallet is unlocked: the network
/// session, connection/status, and orchestration between the Go core
/// (network + crypto) and ChatStore (local persistence) — the Swift-side
/// counterpart of the Go backend's `internal/app.Service` (app/fyne),
/// adapted to this package's Core/ChatStore split.
@MainActor
final class SessionModel: ObservableObject {
    let wallet: WalletHandle
    let chatStore: ChatStore

    @Published private(set) var status: ConnectionStatus = .connecting
    @Published var selectedContact: Contact?

    private let session: NetworkSession
    private var listenTask: Task<Void, Never>?

    var myAddress: String { wallet.address }
    var myContactCode: String { wallet.contactCode }

    init(wallet: WalletHandle, core: OpenChatCore.Type, chatStore: ChatStore, networkSettings: NetworkSettings) throws {
        self.wallet = wallet
        self.chatStore = chatStore
        // caCertPath is a *path* in NetworkSettings (matching the Fyne
        // app's existing "Network settings" field, which uses a native
        // file picker per design-spec.md B6) — read it here rather than
        // asking the Go core to open files itself, since Session.
        // makeSession takes PEM bytes (see mobile/session.go's doc
        // comment on why: a mobile app shouldn't need to hand a Go
        // function a sandboxed file path).
        var caPEM: Data?
        if !networkSettings.caCertPath.isEmpty {
            caPEM = try? Data(contentsOf: URL(fileURLWithPath: networkSettings.caCertPath))
        }
        self.session = try core.makeSession(
            wallet: wallet,
            bootstrapCSV: networkSettings.bootstrapCSV,
            caPEM: caPEM,
            insecureSkipVerify: networkSettings.insecureSkipVerify
        )
    }

    /// Connects, recovers missed history, then starts listening live —
    /// mirrors `internal/app.Service.Start`'s ordering and its accepted
    /// small race window (documented there) between "history sync says
    /// caught up" and "the live subscription actually opens".
    func start() {
        Log.session.info("start() — connecting")
        Task { [weak self] in
            guard let self else { return }
            do {
                self.status = .connecting
                try await self.session.connect(timeoutSeconds: 20)
                Log.session.info("connect() succeeded")
                await self.syncHistory()
                self.beginListening()
            } catch {
                Log.session.error("connect() failed: \(String(describing: error), privacy: .public)")
                self.status = .error(error.localizedDescription)
            }
        }
    }

    private func syncHistory() async {
        status = .syncing
        var known = Dictionary(uniqueKeysWithValues: chatStore.contacts.compactMap { c in
            c.x25519Hex.isEmpty ? nil : (c.address, c.x25519Hex)
        })
        Log.session.info("syncHistory() starting — fromHeight=\(self.chatStore.syncedHeight, privacy: .public)")
        do {
            let (events, next) = try await session.fetchHistory(fromHeight: chatStore.syncedHeight, knownX25519Hex: known)
            Log.session.info("syncHistory() fetched \(events.count, privacy: .public) event(s), nextHeight=\(next, privacy: .public)")
            var stored = 0
            var skippedDupes = 0
            for event in events {
                // peerAddress is already "sender's address if incoming,
                // recipient's otherwise" (see HistoryEventPayload) — the
                // conversation this event belongs to either way.
                let address = event.peerAddress
                let message = StoredMessage(
                    id: event.txHash,
                    direction: event.incoming ? .incoming : .outgoing,
                    text: event.text,
                    timestampMs: event.timestampMs
                )
                if chatStore.appendHistoryMessage(address: address, message: message, peerX25519Hex: event.peerX25519Hex) {
                    stored += 1
                } else {
                    skippedDupes += 1
                }
                if event.incoming { known[event.peerAddress] = event.peerX25519Hex }
            }
            Log.session.info("syncHistory() stored \(stored, privacy: .public), skipped \(skippedDupes, privacy: .public) already-known dupe(s)")
            chatStore.setSyncedHeight(next)
            status = .upToDate
        } catch {
            // Non-fatal: still proceed to live listening (matches
            // internal/app.Service.Start's "will still receive new
            // messages live" behavior on a history-sync failure).
            Log.session.error("syncHistory() failed: \(String(describing: error), privacy: .public)")
            status = .error("history sync failed: \(error.localizedDescription)")
        }
    }

    private func beginListening() {
        listenTask?.cancel()
        Log.session.info("beginListening() — starting live listen loop")
        listenTask = Task { [weak self] in
            guard let self else { return }
            do {
                try await self.session.listen { [weak self] incoming in
                    guard let self else { return false }
                    Log.session.info("listen(): incoming message from \(incoming.fromAddress, privacy: .public), txHash=\(incoming.txHash, privacy: .public), \(incoming.plaintext.count, privacy: .public) byte(s)")
                    Task { @MainActor in
                        let message = StoredMessage(
                            id: incoming.txHash,
                            direction: .incoming,
                            text: String(data: incoming.plaintext, encoding: .utf8) ?? "",
                            // Live delivery carries no wall-clock timestamp of
                            // its own (see IncomingMessagePayload) — use
                            // receipt time, same as the Go/Fyne backend's
                            // Service.Start does for a live StreamIncomingSMS
                            // arrival (internal/app/service.go: NowMillis()).
                            timestampMs: Int64(Date().timeIntervalSince1970 * 1000)
                        )
                        self.chatStore.appendMessage(address: incoming.fromAddress, message: message, senderX25519Hex: incoming.fromX25519Hex)
                        if self.status != .upToDate { self.status = .upToDate }
                    }
                    return true
                }
                Log.session.info("listen() returned normally (stream ended without error)")
            } catch {
                if !Task.isCancelled {
                    Log.session.error("listen() stopped: \(String(describing: error), privacy: .public)")
                    self.status = .error("listen stopped: \(error.localizedDescription)")
                } else {
                    Log.session.info("listen() cancelled")
                }
            }
        }
    }

    /// Sends `text` to `contact` and records it in local history.
    /// Clears/restores composer state is the caller's (view's)
    /// responsibility — this just does the network call + persistence.
    func sendText(_ text: String, to contact: Contact) async throws {
        guard !contact.x25519Hex.isEmpty else {
            Log.session.error("sendText() rejected: no known encryption key for \(contact.address, privacy: .public)")
            throw OpenChatError(message: "This contact has no known encryption key yet.")
        }
        Log.session.info("sendText() to \(contact.address, privacy: .public), \(text.utf8.count, privacy: .public) byte(s)")
        do {
            let txHash = try await session.sendText(toAddress: contact.address, toX25519Hex: contact.x25519Hex, text: text)
            Log.session.info("sendText() succeeded, txHash=\(txHash, privacy: .public)")
            let message = StoredMessage(id: txHash, direction: .outgoing, text: text, timestampMs: Int64(Date().timeIntervalSince1970 * 1000))
            chatStore.appendMessage(address: contact.address, message: message, senderX25519Hex: "")
        } catch {
            Log.session.error("sendText() failed: \(String(describing: error), privacy: .public)")
            throw error
        }
    }

    func stop() {
        Log.session.info("stop() — cancelling listen task and closing session")
        listenTask?.cancel()
        listenTask = nil
        session.close()
    }
}
