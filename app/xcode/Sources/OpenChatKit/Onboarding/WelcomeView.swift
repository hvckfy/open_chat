import SwiftUI

/// A1 — Welcome (design-spec.md §3 A1). First screen; no back/skip.
struct WelcomeView: View {
    let onCreate: () -> Void
    let onImport: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            #if os(iOS)
            Spacer().frame(height: 120)
            #else
            Spacer()
            #endif

            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .fill(Theme.Color.accent)
                .frame(width: 56, height: 56)
                .padding(.bottom, 16)

            Text("OpenChat")
                .font(Theme.Font.display)
                .foregroundStyle(Theme.Color.textPrimary)

            Text("A private messenger with no company in the middle. End-to-end encrypted. Free to send, always.")
                .font(Theme.Font.subheadline)
                .foregroundStyle(Theme.Color.textSecondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 320)
                .padding(.top, 12)

            Spacer()

            VStack(spacing: 12) {
                PrimaryButton(title: "Create a new identity", action: onCreate)
                SecondaryButton(title: "I already have a recovery phrase", action: onImport)
            }
            .frame(maxWidth: buttonMaxWidth)
            .padding(.horizontal, horizontalPadding)
            .padding(.bottom, bottomPadding)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Theme.Color.bgCanvas)
    }

    #if os(iOS)
    private let buttonMaxWidth: CGFloat = .infinity
    private let horizontalPadding: CGFloat = Theme.Spacing.screenMarginCompact
    private let bottomPadding: CGFloat = 40
    #else
    private let buttonMaxWidth: CGFloat = 280
    private let horizontalPadding: CGFloat = 0
    private let bottomPadding: CGFloat = 48
    #endif
}

#Preview {
    WelcomeView(onCreate: {}, onImport: {})
}
