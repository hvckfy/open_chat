import SwiftUI

/// A2 — Back up your recovery phrase (design-spec.md §3 A2). Highest
/// visual weight screen in onboarding — the account's only backup,
/// shown exactly once. No skip.
struct BackupPhraseView: View {
    let mnemonic: String
    let onContinue: () -> Void

    @State private var confirmed = false

    private var words: [String] { mnemonic.split(separator: " ").map(String.init) }

    var body: some View {
        VStack(alignment: .leading, spacing: Theme.Spacing.group) {
            Text("Back up your recovery phrase")
                .font(Theme.Font.title)
                .foregroundStyle(Theme.Color.textPrimary)

            warningBlock

            RecoveryPhraseCard(words: words, columns: columnCount)

            Button {
                confirmed.toggle()
            } label: {
                HStack(spacing: 10) {
                    Image(systemName: confirmed ? "checkmark.square.fill" : "square")
                        .font(.system(size: 20))
                        .foregroundStyle(confirmed ? Theme.Color.accent : Theme.Color.textSecondary)
                    Text("I've written down my recovery phrase.")
                        .font(Theme.Font.body)
                        .foregroundStyle(Theme.Color.textPrimary)
                    Spacer(minLength: 0)
                }
            }
            .buttonStyle(.plain)

            Spacer(minLength: 0)

            PrimaryButton(title: "Continue", isEnabled: confirmed, action: onContinue)
        }
        .padding(screenMargin)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(Theme.Color.bgCanvas)
    }

    private var warningBlock: some View {
        HStack(alignment: .top, spacing: 10) {
            Circle()
                .fill(Theme.Color.danger)
                .frame(width: 20, height: 20)
                .padding(.top, 2)
            Text("Write these words down. Anyone with them can read your messages and impersonate you. This is the only way back into your account — OpenChat cannot reset it.")
                .font(Theme.Font.caption)
                .foregroundStyle(Theme.Color.textPrimary)
        }
        .padding(14)
        .background(Theme.Color.bgSurface2)
        .overlay(
            RoundedRectangle(cornerRadius: Theme.Radius.medium, style: .continuous)
                .strokeBorder(Theme.Color.danger.opacity(0.3), lineWidth: 1)
        )
        .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.medium, style: .continuous))
    }

    #if os(iOS)
    private let screenMargin: CGFloat = Theme.Spacing.screenMarginCompact
    private let columnCount = 2
    #else
    private let screenMargin: CGFloat = Theme.Spacing.screenMarginRegular
    private let columnCount = 3
    #endif
}

#Preview {
    BackupPhraseView(mnemonic: (1...24).map { "word\($0)" }.joined(separator: " "), onContinue: {})
}
