import SwiftUI

/// A3 — Import your recovery phrase (design-spec.md §3 A3).
struct ImportPhraseView: View {
    var errorMessage: String?
    let onImport: (String) -> Void
    let onBack: () -> Void

    @State private var phrase = ""

    var body: some View {
        VStack(alignment: .leading, spacing: Theme.Spacing.tight) {
            Text("Import your recovery phrase")
                .font(Theme.Font.title)
                .foregroundStyle(Theme.Color.textPrimary)
            Text("Enter your 12 or 24-word phrase, separated by spaces.")
                .font(Theme.Font.subheadline)
                .foregroundStyle(Theme.Color.textSecondary)
                .padding(.bottom, Theme.Spacing.tight)

            LabeledMultilineField(
                placeholder: "Enter your 12 or 24-word phrase…",
                text: $phrase,
                minHeight: 120,
                isError: errorMessage != nil
            )
            if let errorMessage {
                FieldError(message: errorMessage)
            }

            Spacer(minLength: 0)

            PrimaryButton(title: "Import", isEnabled: !phrase.trimmingCharacters(in: .whitespaces).isEmpty) {
                onImport(phrase)
            }
            HStack {
                Spacer()
                PlainTextButton(title: "Back", action: onBack)
                Spacer()
            }
        }
        .padding(screenMargin)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(Theme.Color.bgCanvas)
    }

    #if os(iOS)
    private let screenMargin: CGFloat = Theme.Spacing.screenMarginCompact
    #else
    private let screenMargin: CGFloat = Theme.Spacing.screenMarginRegular
    #endif
}

#Preview {
    ImportPhraseView(errorMessage: "Wrong number of words — a recovery phrase is 12 or 24 words.", onImport: { _ in }, onBack: {})
}
