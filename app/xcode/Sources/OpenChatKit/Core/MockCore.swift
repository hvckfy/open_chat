import Foundation

/// An in-memory, no-network `OpenChatCore` for SwiftUI previews, unit
/// tests, and building this package before the real `LiveCore` (Core/
/// LiveCore.swift) has an XCFramework to link against. Deterministic and
/// entirely local: no gomobile dependency at all.
enum MockCore: OpenChatCore {
    private static let mockWordlist = [
        "ripple", "cactus", "harbor", "velvet", "meadow", "comet", "granite", "willow",
        "amber", "quartz", "ember", "cedar", "tundra", "orbit", "flint", "coral",
        "birch", "canyon", "lumen", "delta", "marsh", "otter", "spruce", "vapor",
    ]

    static func generateMnemonic() throws -> String {
        mockWordlist.shuffled().joined(separator: " ")
    }

    static func createWallet() throws -> WalletHandle {
        try MockWallet(mnemonic: generateMnemonic())
    }

    static func importWallet(mnemonic: String) throws -> WalletHandle {
        let words = mnemonic.split(separator: " ")
        guard words.count == 12 || words.count == 24 else {
            throw OpenChatError(message: "Wrong number of words — a recovery phrase is 12 or 24 words.")
        }
        return try MockWallet(mnemonic: mnemonic)
    }

    static func parseContactCode(_ code: String) throws -> (address: String, x25519Hex: String) {
        guard code.hasPrefix("oc1:"), code.count > 8 else {
            throw OpenChatError(message: "That doesn't look like a valid contact code.")
        }
        let seed = String(code.dropFirst(4))
        return (address: Self.hexHash(seed, bytes: 32), x25519Hex: Self.hexHash(seed + "x", bytes: 32))
    }

    static func makeSession(wallet: WalletHandle, bootstrapCSV: String, caPEM: Data?, insecureSkipVerify: Bool) throws -> NetworkSession {
        MockSession(wallet: wallet)
    }

    fileprivate static func hexHash(_ s: String, bytes: Int) -> String {
        var hash: UInt64 = 1469598103934665603
        for b in s.utf8 {
            hash = (hash ^ UInt64(b)) &* 1099511628211
        }
        var out = ""
        var h = hash
        while out.count < bytes * 2 {
            out += String(format: "%016llx", h)
            h = h &* 6364136223846793005 &+ 1
        }
        return String(out.prefix(bytes * 2))
    }
}

private struct MockWallet: WalletHandle {
    let mnemonic: String
    let address: String
    let encryptionPublicHex: String

    init(mnemonic: String) throws {
        self.mnemonic = mnemonic
        self.address = MockCore.hexHash(mnemonic, bytes: 32)
        self.encryptionPublicHex = MockCore.hexHash(mnemonic + "x25519", bytes: 32)
    }

    var contactCode: String {
        "oc1:" + address.prefix(16) + encryptionPublicHex.prefix(16)
    }
}

/// A session that "connects" instantly and echoes nothing — enough for
/// previews and UI-flow testing without a real network.
private final class MockSession: NetworkSession {
    private let wallet: WalletHandle
    private var stopped = false

    init(wallet: WalletHandle) {
        self.wallet = wallet
    }

    func connect(timeoutSeconds: Int64) async throws {
        try? await Task.sleep(nanoseconds: 300_000_000)
    }

    var currentGateway: String { "mock.local:9090" }

    func sendText(toAddress: String, toX25519Hex: String, text: String) async throws -> String {
        try? await Task.sleep(nanoseconds: 150_000_000)
        return MockCore.hexHash(toAddress + text + "\(Date().timeIntervalSince1970)", bytes: 32)
    }

    func listen(onMessage: @escaping (IncomingMessagePayload) -> Bool) async throws {
        stopped = false
        while !stopped {
            try Task.checkCancellation()
            try? await Task.sleep(nanoseconds: 1_000_000_000)
        }
    }

    func stop() { stopped = true }

    func fetchHistory(fromHeight: Int64, knownX25519Hex: [String: String]) async throws -> (events: [HistoryEventPayload], nextHeight: Int64) {
        (events: [], nextHeight: fromHeight)
    }

    func close() { stop() }
}
