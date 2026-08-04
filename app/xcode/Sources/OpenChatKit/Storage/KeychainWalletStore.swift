import Foundation
import CryptoKit
#if canImport(Security)
import Security
#endif

/// Persists exactly one wallet's recovery phrase, encrypted, in the
/// system Keychain — the native-shell half of the boundary described in
/// app/golang/mobile's package doc comment ("a real iOS/macOS app has a
/// better [option than Fyne's PIN+scrypt file] — the system Keychain").
///
/// Two layers of protection, deliberately: the Keychain item itself is
/// already encrypted at rest by the OS (`.whenUnlockedThisDeviceOnly` —
/// excluded from unencrypted backups, inaccessible before first unlock,
/// never leaves this device even via iCloud Keychain), and on top of
/// that the mnemonic is AES-256-GCM sealed with a key derived from the
/// user's local app PIN (PBKDF2-HMAC-SHA256, 100k rounds) before it's
/// ever written — matching design-spec.md's A4/A5 PIN-gated flow, and
/// meaning a compromised/jailbroken-device Keychain dump alone still
/// isn't enough without the PIN too.
final class KeychainWalletStore {
    private let service = "network.openchat.app.wallet"
    private let account = "wallet"
    private let pbkdf2Iterations = 100_000

    private struct Blob: Codable {
        let salt: Data
        let nonce: Data
        let ciphertext: Data
    }

    enum StoreError: LocalizedError {
        case wrongPIN
        case keychain(OSStatus)

        var errorDescription: String? {
            switch self {
            case .wrongPIN: return "Incorrect PIN or corrupted wallet data."
            case .keychain(let status): return "Keychain error (\(status))."
            }
        }
    }

    init() {}

    /// Whether a wallet has already been onboarded on this device.
    func exists() -> Bool {
        (try? readRaw()) != nil
    }

    /// Encrypts `mnemonic` with a key derived from `pin` and writes it to
    /// the Keychain, replacing any existing entry.
    func save(mnemonic: String, pin: String) throws {
        var salt = Data(count: 16)
        _ = salt.withUnsafeMutableBytes { SecRandomCopyBytes(kSecRandomDefault, 16, $0.baseAddress!) }

        let key = try deriveKey(pin: pin, salt: salt)
        let nonce = AES.GCM.Nonce()
        let sealed = try AES.GCM.seal(Data(mnemonic.utf8), using: key, nonce: nonce)

        let blob = Blob(salt: salt, nonce: Data(nonce), ciphertext: sealed.ciphertext + sealed.tag)
        let data = try JSONEncoder().encode(blob)
        try writeRaw(data)
    }

    /// Decrypts and returns the stored mnemonic. Throws `.wrongPIN` if
    /// `pin` doesn't match (barring data corruption).
    func load(pin: String) throws -> String {
        let data = try readRaw()
        let blob = try JSONDecoder().decode(Blob.self, from: data)
        let key = try deriveKey(pin: pin, salt: blob.salt)

        guard blob.ciphertext.count > 16 else { throw StoreError.wrongPIN }
        let tag = blob.ciphertext.suffix(16)
        let ct = blob.ciphertext.dropLast(16)
        let nonce = try AES.GCM.Nonce(data: blob.nonce)

        do {
            let sealedBox = try AES.GCM.SealedBox(nonce: nonce, ciphertext: ct, tag: tag)
            let plain = try AES.GCM.open(sealedBox, using: key)
            guard let mnemonic = String(data: plain, encoding: .utf8) else { throw StoreError.wrongPIN }
            return mnemonic
        } catch {
            throw StoreError.wrongPIN
        }
    }

    /// Permanently erases the on-disk (Keychain) wallet entry, if one
    /// exists — the core of "Log out" (see design-code.md B7).
    func delete() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw StoreError.keychain(status)
        }
    }

    // MARK: - PBKDF2-HMAC-SHA256 (RFC 8018) key derivation

    private func deriveKey(pin: String, salt: Data) throws -> SymmetricKey {
        let password = SymmetricKey(data: Data(pin.utf8))
        var derived = Data()
        var blockIndex: UInt32 = 1
        while derived.count < 32 {
            var beIndex = blockIndex.bigEndian
            let indexData = withUnsafeBytes(of: &beIndex) { Data($0) }
            var u = Data(HMAC<SHA256>.authenticationCode(for: salt + indexData, using: password))
            var result = u
            if pbkdf2Iterations > 1 {
                for _ in 1..<pbkdf2Iterations {
                    u = Data(HMAC<SHA256>.authenticationCode(for: u, using: password))
                    for i in 0..<result.count { result[i] ^= u[i] }
                }
            }
            derived.append(result)
            blockIndex += 1
        }
        return SymmetricKey(data: derived.prefix(32))
    }

    // MARK: - Keychain plumbing

    private func readRaw() throws -> Data {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let data = item as? Data else {
            throw StoreError.keychain(status)
        }
        return data
    }

    private func writeRaw(_ data: Data) throws {
        let base: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        if exists() {
            let update: [String: Any] = [kSecValueData as String: data]
            let status = SecItemUpdate(base as CFDictionary, update as CFDictionary)
            guard status == errSecSuccess else { throw StoreError.keychain(status) }
        } else {
            var add = base
            add[kSecValueData as String] = data
            add[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
            let status = SecItemAdd(add as CFDictionary, nil)
            guard status == errSecSuccess else { throw StoreError.keychain(status) }
        }
    }
}
