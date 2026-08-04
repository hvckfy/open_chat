import SwiftUI

/// Owns onboarding's `OnboardingModel` and switches between A1–A5 based
/// on its `step` — the one `SetContent`-equivalent place for the whole
/// flow (design-code.md: "A linear flow, not a freely-navigable set of
/// tabs"). See RootView for how this is instantiated when
/// `AppModel.stage == .onboarding`.
// Internal, not public — see MainSplitView's identical note. KeychainWalletStore/
// OpenChatCore are themselves internal; only RootView.swift constructs this.
struct OnboardingCoordinator: View {
    @StateObject private var model: OnboardingModel

    init(keychain: KeychainWalletStore, core: OpenChatCore.Type, onReady: @escaping (String) throws -> Void) {
        _model = StateObject(wrappedValue: OnboardingModel(keychain: keychain, core: core, onReady: onReady))
    }

    var body: some View {
        Group {
            switch model.step {
            case .welcome:
                WelcomeView(onCreate: model.createIdentity, onImport: model.chooseImport)
            case .backup(let mnemonic):
                BackupPhraseView(mnemonic: mnemonic) {
                    model.confirmedBackup(mnemonic: mnemonic)
                }
            case .importPhrase:
                ImportPhraseView(errorMessage: model.errorMessage, onImport: model.submitImport, onBack: model.backToWelcome)
            case .setPIN(let mnemonic):
                SetPINView(errorMessage: model.errorMessage, isBusy: model.isBusy) { pin in
                    model.savePIN(pin, mnemonic: mnemonic)
                }
            case .unlock:
                UnlockView(errorMessage: model.errorMessage, isBusy: model.isBusy, onUnlock: model.unlock, onUseDifferentPhrase: model.logOutAndImportInstead)
            }
        }
    }
}
