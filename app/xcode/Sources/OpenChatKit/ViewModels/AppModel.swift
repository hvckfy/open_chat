import Foundation

/// Top-level app state: which of the two mutually-exclusive top-level
/// states (design-code.md "Global structure") is showing right now.
@MainActor
final class AppModel: ObservableObject {
    enum Stage: Equatable {
        case checking // reserved for a possible future async startup check — see init
        case onboarding
        case main
    }

    // No default value: always set explicitly in init (never left at a
    // placeholder that a view could render for a frame before the real
    // decision lands).
    @Published private(set) var stage: Stage
    @Published private(set) var session: SessionModel?

    let keychain = KeychainWalletStore()
    let chatStore = ChatStore()
    let core: OpenChatCore.Type

    /// Network settings (design-code.md B6), loaded once at init from
    /// wherever the host app persists simple preferences — pass in a
    /// loader/saver so this package doesn't dictate UserDefaults vs.
    /// something else.
    @Published var networkSettings: NetworkSettings

    private let settingsStore: NetworkSettingsStore

    init(core: OpenChatCore.Type = OpenChatCoreProvider.current, settingsStore: NetworkSettingsStore = UserDefaultsNetworkSettingsStore()) {
        self.core = core
        self.settingsStore = settingsStore
        self.networkSettings = settingsStore.load()
        // Decided right here in init, not via a separate
        // decideInitialStage() called from RootView's `.onAppear` (how
        // this used to work) — that mutated `stage` (a `@Published`
        // property this same view is switching on) from inside the root
        // view's very first update pass, which is exactly what SwiftUI's
        // "Publishing changes from within view updates is not allowed,
        // this will cause undefined behavior" warning is about: fixing
        // it here means `stage` already has its real value before
        // `RootView` ever renders a first frame, so there's no follow-up
        // publish to trigger it. `.checking` is kept as an enum case for
        // a possible future *actually async* startup check, but nothing
        // currently produces it.
        self.stage = .onboarding
        // (Both would-be branches of the old decideInitialStage() landed
        // on `.onboarding` — OnboardingModel itself picks Unlock vs.
        // Welcome as its first step based on `keychain.exists()`, see
        // OnboardingModel.init — so this is intentionally not
        // conditioned on `keychain.exists()` here too.)
    }

    /// Called once onboarding produces an unlocked mnemonic (either a
    /// fresh identity that just finished backup+PIN, or a returning
    /// user's successful Unlock). Derives the wallet, opens a session,
    /// and switches to the main app.
    func completeOnboarding(mnemonic: String) throws {
        let wallet = try core.importWallet(mnemonic: mnemonic)
        let session = try SessionModel(wallet: wallet, core: core, chatStore: chatStore, networkSettings: networkSettings)
        self.session = session
        self.stage = .main
        session.start()
    }

    /// design-code.md B7 — wipes both halves of local state (Keychain +
    /// chat history) and returns to onboarding. Never fails outward: a
    /// user who asked to log out should always end up back at Welcome,
    /// even if one deletion step hit an error (best-effort, same
    /// reasoning as the Go/Fyne app's wipeSessionData).
    func logout() {
        session?.stop()
        try? keychain.delete()
        chatStore.wipe()
        session = nil
        stage = .onboarding
    }

    func saveNetworkSettings(_ settings: NetworkSettings) {
        networkSettings = settings
        settingsStore.save(settings)
    }
}

/// design-code.md B6's Network settings fields.
struct NetworkSettings: Equatable {
    var bootstrapGateways: [String] = []
    var caCertPath: String = ""
    var insecureSkipVerify: Bool = false

    var bootstrapCSV: String { bootstrapGateways.joined(separator: ",") }
}

protocol NetworkSettingsStore {
    func load() -> NetworkSettings
    func save(_ settings: NetworkSettings)
}

/// Default `NetworkSettingsStore`: `UserDefaults` (no secrets here — the
/// CA cert is a path/public data, matching what app/fyne's
/// `settings.go` already treats as ordinary preferences via Fyne's
/// Preferences API).
final class UserDefaultsNetworkSettingsStore: NetworkSettingsStore {
    private let bootstrapKey = "network.bootstrap"
    private let caPathKey = "network.ca_path"
    private let insecureKey = "network.insecure_skip_verify"
    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    func load() -> NetworkSettings {
        let csv = defaults.string(forKey: bootstrapKey) ?? ""
        let gateways = csv.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
        return NetworkSettings(
            bootstrapGateways: gateways,
            caCertPath: defaults.string(forKey: caPathKey) ?? "",
            insecureSkipVerify: defaults.bool(forKey: insecureKey)
        )
    }

    func save(_ settings: NetworkSettings) {
        defaults.set(settings.bootstrapCSV, forKey: bootstrapKey)
        defaults.set(settings.caCertPath, forKey: caPathKey)
        defaults.set(settings.insecureSkipVerify, forKey: insecureKey)
    }
}
