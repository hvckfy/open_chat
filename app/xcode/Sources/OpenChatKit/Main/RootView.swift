import SwiftUI

/// The single entry point both app targets (macOS/iOS — see
/// ../../README.md "Setting up the Xcode project") mount: owns the one
/// `AppModel` for the process and switches between onboarding and the
/// main app per its `stage` (design-code.md "Global structure"). Each
/// platform's `App.swift` is just `WindowGroup { RootView() }` (macOS
/// additionally sets the window size — see the reference App.swift
/// files under app/xcode/AppTargets/).
public struct RootView: View {
    @StateObject private var appModel: AppModel

    public init() {
        _appModel = StateObject(wrappedValue: AppModel())
    }

    public var body: some View {
        // `appModel.stage` is decided synchronously in AppModel.init, so
        // this never actually renders `.checking` today — no `.onAppear`
        // deciding it after the fact anymore either (that used to
        // mutate `stage` from inside this view's own first update pass,
        // triggering SwiftUI's "Publishing changes from within view
        // updates is not allowed" warning; see AppModel.init's comment).
        // `.checking` stays handled here only so a possible future async
        // startup check has somewhere to render while it works.
        switch appModel.stage {
        case .checking:
            Theme.Color.bgCanvas.ignoresSafeArea()
        case .onboarding:
            OnboardingCoordinator(
                keychain: appModel.keychain,
                core: appModel.core,
                onReady: { mnemonic in try appModel.completeOnboarding(mnemonic: mnemonic) }
            )
        case .main:
            if let session = appModel.session {
                MainSplitView(appModel: appModel, session: session)
            } else {
                // Unreachable in practice (completeOnboarding sets
                // `session` and `stage` together), kept as a safe
                // fallback rather than force-unwrapping.
                Theme.Color.bgCanvas.ignoresSafeArea()
            }
        }
    }
}
