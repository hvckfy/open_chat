import SwiftUI

/// B7 — Log out confirmation (design-spec.md §3 B7): "a system-native
/// confirmation pattern... not a custom full sheet." `.alert` gives that
/// on both platforms (macOS renders it as the NSAlert-style sheet the
/// spec asks for; iOS as its native alert) without a bespoke view.
extension View {
    func logoutConfirmation(isPresented: Binding<Bool>, onConfirm: @escaping () -> Void) -> some View {
        alert("Log out of OpenChat?", isPresented: isPresented) {
            Button("Cancel", role: .cancel) {}
            Button("Log Out", role: .destructive, action: onConfirm)
        } message: {
            Text("This deletes your wallet and message history from this device. Your recovery phrase is the only way back in — make sure you've saved it.")
        }
    }
}
