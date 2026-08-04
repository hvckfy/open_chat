import Foundation

/// One incoming message delivered live by `NetworkSession.listen`.
struct IncomingMessagePayload {
    let fromAddress: String
    let fromX25519Hex: String
    let plaintext: Data
    let txHash: String
    let blockHeight: Int64
}

/// One event recovered by `NetworkSession.fetchHistory`.
struct HistoryEventPayload {
    let height: Int64
    let txHash: String
    let timestampMs: Int64
    let incoming: Bool
    let peerAddress: String
    let peerX25519Hex: String
    let text: String
}

/// A derived identity — the Swift-side counterpart of `mobile.Wallet`
/// (app/golang/mobile/wallet.go). `LiveWallet` (Core/LiveCore.swift)
/// wraps the real gomobile type; `MockWallet` (Core/MockCore.swift) is
/// an in-memory stand-in for previews/tests/pre-XCFramework builds.
protocol WalletHandle {
    var address: String { get }
    var encryptionPublicHex: String { get }
    /// Only meaningful right after creation, for the one-time backup
    /// screen (A2) — see mobile.Wallet.Mnemonic's doc comment. Never
    /// persist this; that's exactly the secret Keychain protects.
    var mnemonic: String { get }
    var contactCode: String { get }
}

/// A live connection to the OpenChat network for one wallet — the
/// Swift-side counterpart of `mobile.Session`. Every method (other than
/// `stop`/`close`) can throw a network/protocol error; surface it to the
/// UI per design-spec.md's per-action error convention rather than a
/// global banner.
protocol NetworkSession: AnyObject {
    /// Dials the bootstrap gateway list with failover (see
    /// pkg/client/discovery.go). Call once after creating the session.
    func connect(timeoutSeconds: Int64) async throws
    /// "host:port" of the currently connected gateway, or "" if never
    /// connected — for status display only.
    var currentGateway: String { get }
    /// Encrypts, signs and submits a text message; returns the
    /// committed transaction's hash.
    func sendText(toAddress: String, toX25519Hex: String, text: String) async throws -> String
    /// Opens a live subscription and suspends until `onMessage` returns
    /// false, `stop()` is called, or an unrecoverable error occurs.
    /// `onMessage` may be called from a background thread.
    func listen(onMessage: @escaping (IncomingMessagePayload) -> Bool) async throws
    /// Ends a running `listen` call (no-op if none is running).
    func stop()
    /// Recovers message history from `fromHeight` (0 for a full resync)
    /// up through whatever's currently committed. `knownX25519Hex` seeds
    /// (and, on return, should be updated from) the caller's saved
    /// contacts — see mobile.Session.FetchHistory's doc comment for why
    /// this wallet's own past outgoing messages need it.
    func fetchHistory(fromHeight: Int64, knownX25519Hex: [String: String]) async throws -> (events: [HistoryEventPayload], nextHeight: Int64)
    /// Stops listening and releases the connection. Call on logout.
    func close()
}

/// The single entry point the rest of this package (and the app target)
/// uses to reach the Go core — an abstract factory, so views/view-models
/// never import the generated gomobile framework directly and keep
/// working (against `MockCore`) in SwiftUI previews and in any build
/// made before that framework exists. `OpenChatCoreProvider.current` is
/// what actually selects an implementation — see Core/CoreProvider.swift.
protocol OpenChatCore {
    static func generateMnemonic() throws -> String
    static func createWallet() throws -> WalletHandle
    static func importWallet(mnemonic: String) throws -> WalletHandle
    static func parseContactCode(_ code: String) throws -> (address: String, x25519Hex: String)
    static func makeSession(wallet: WalletHandle, bootstrapCSV: String, caPEM: Data?, insecureSkipVerify: Bool) throws -> NetworkSession
}

/// A plain Swift error carrying whatever message the Go side (or a mock)
/// produced, so callers don't need to know whether it originated from
/// NSError-bridged Go error or a Swift-native one.
struct OpenChatError: LocalizedError {
    let message: String
    var errorDescription: String? { message }
}
