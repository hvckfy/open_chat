import Foundation

/// Drives the onboarding flow (design-code.md "A. Onboarding flow"):
/// which screen is showing and the transient state that crosses between
/// them (a freshly generated mnemonic on its way to being backed up and
/// PIN-protected). Owns no long-lived state — once `onReady` fires,
/// AppModel takes over and this object is discarded.
@MainActor
final class OnboardingModel: ObservableObject {
    enum Step: Equatable {
        case welcome
        case backup(mnemonic: String)
        case importPhrase
        case setPIN(mnemonic: String)
        case unlock
    }

    @Published private(set) var step: Step
    @Published var isBusy = false
    @Published var errorMessage: String?

    private let keychain: KeychainWalletStore
    private let core: OpenChatCore.Type
    /// Called once with the unlocked mnemonic — AppModel.completeOnboarding.
    private let onReady: (String) throws -> Void

    init(keychain: KeychainWalletStore, core: OpenChatCore.Type, onReady: @escaping (String) throws -> Void) {
        self.keychain = keychain
        self.core = core
        self.onReady = onReady
        self.step = keychain.exists() ? .unlock : .welcome
    }

    // MARK: - A1 Welcome

    func createIdentity() {
        errorMessage = nil
        do {
            let mnemonic = try core.generateMnemonic()
            step = .backup(mnemonic: mnemonic)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func chooseImport() {
        errorMessage = nil
        step = .importPhrase
    }

    // MARK: - A2 Backup

    /// Continue tapped with the confirmation checkbox checked.
    func confirmedBackup(mnemonic: String) {
        step = .setPIN(mnemonic: mnemonic)
    }

    // MARK: - A3 Import

    func submitImport(phrase: String) {
        errorMessage = nil
        let trimmed = phrase.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            // Validate by actually deriving — surfaces the same
            // word-count/checksum errors design-spec.md A3 expects,
            // straight from the one real implementation of that check.
            _ = try core.importWallet(mnemonic: trimmed)
            step = .setPIN(mnemonic: trimmed)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func backToWelcome() {
        errorMessage = nil
        step = .welcome
    }

    // MARK: - A4 Set PIN

    func savePIN(_ pin: String, mnemonic: String) {
        errorMessage = nil
        isBusy = true
        // See `unlock(pin:)`'s comment just below — `keychain.save` runs
        // the same deliberately-slow PBKDF2 derivation `.load` does, so
        // it needs the same off-main-thread treatment for the same
        // reason, even though this one's triggered by an explicit button
        // tap rather than an auto-submitting last keystroke.
        Task {
            do {
                let keychain = keychain
                try await Task.detached(priority: .userInitiated) {
                    try keychain.save(mnemonic: mnemonic, pin: pin)
                }.value
                isBusy = false
                try onReady(mnemonic)
            } catch {
                isBusy = false
                errorMessage = error.localizedDescription
            }
        }
    }

    // MARK: - A5 Unlock

    func unlock(pin: String) {
        errorMessage = nil
        isBusy = true
        // `keychain.load` runs PBKDF2-HMAC-SHA256 at 100k rounds
        // (deliberately slow, by design — see KeychainWalletStore's doc
        // comment) entirely synchronously. Calling it directly here,
        // on this @MainActor method, blocked the main thread for the
        // whole derivation with no suspension point in between — so the
        // SwiftUI re-render for the 4th PIN dot filling in (triggered by
        // `pin.append` in UnlockView, immediately followed by this call)
        // never got a chance to actually draw before the thread froze,
        // reading as "only 3 dots appeared" even though the 4th digit
        // was recorded fine. Running the derivation on a detached task
        // and hopping back to the main actor only to update `@Published`
        // state lets that frame render normally.
        Task {
            do {
                let keychain = keychain
                let mnemonic = try await Task.detached(priority: .userInitiated) {
                    try keychain.load(pin: pin)
                }.value
                isBusy = false
                try onReady(mnemonic)
            } catch {
                isBusy = false
                errorMessage = "Incorrect PIN"
            }
        }
    }

    /// "Log out and use a different recovery phrase" (A5) — wipes the
    /// existing wallet so the user can re-onboard via A3 Import. A
    /// destructive escape hatch for a genuinely PIN-locked-out user;
    /// see design-spec.md A5.
    func logOutAndImportInstead() {
        try? keychain.delete()
        errorMessage = nil
        step = .welcome
    }
}
