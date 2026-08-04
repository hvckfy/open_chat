import SwiftUI

/// B3 — Add contact (design-spec.md §3 B3).
struct AddContactSheet: View {
    let core: OpenChatCore.Type
    let chatStore: ChatStore
    /// Called after a contact is successfully added, so the caller can
    /// select it immediately (design-code.md: adding a contact should
    /// drop you straight into the new conversation).
    var onAdded: (Contact) -> Void = { _ in }

    @Environment(\.dismiss) private var dismiss
    @State private var code = ""
    @State private var name = ""
    @State private var errorMessage: String?

    var body: some View {
        VStack(alignment: .leading, spacing: Theme.Spacing.group) {
            Text("Add contact")
                .font(Theme.Font.title)
                .foregroundStyle(Theme.Color.textPrimary)

            LabeledMultilineField(label: "Contact code", placeholder: "Paste a contact code…", text: $code, minHeight: 80, isError: errorMessage != nil)
            if let errorMessage {
                FieldError(message: errorMessage)
            }

            LabeledField(label: "Name (optional)", placeholder: "Name", text: $name)

            Spacer(minLength: 0)

            HStack(spacing: 12) {
                Spacer()
                PlainTextButton(title: "Cancel", color: Theme.Color.textPrimary) { dismiss() }
                PrimaryButtonCompact(title: "Add", isEnabled: !trimmedCode.isEmpty, action: submit)
            }
        }
        .padding(24)
        // A fixed width (rather than min/ideal) on macOS — a sheet sized
        // with `minWidth`/`idealWidth` visibly pops from its fitted
        // content size to the ideal one right after presenting instead
        // of appearing at its final size immediately. `.large`-only on
        // iOS for the same "arrives at final size, not mid-animation
        // resize" reason (the code field plus name field plus buttons
        // don't comfortably fit `.medium` anyway).
        #if os(macOS)
        // Two chained calls, not `.frame(width:minHeight:)` — see
        // EditContactSheet's note: SwiftUI has no overload mixing a fixed
        // `width:` with `minHeight:` in a single call.
        .frame(width: 440)
        .frame(minHeight: 320)
        #else
        .presentationDetents([.large])
        #endif
    }

    private var trimmedCode: String {
        code.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func submit() {
        errorMessage = nil
        do {
            let parsed = try core.parseContactCode(trimmedCode)
            let trimmedName = name.trimmingCharacters(in: .whitespaces)
            let contact = Contact(
                address: parsed.address,
                x25519Hex: parsed.x25519Hex,
                displayName: trimmedName.isEmpty ? Contact.placeholderName(for: parsed.address) : trimmedName
            )
            chatStore.upsertContact(contact)
            onAdded(contact)
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// A smaller, trailing-aligned accent button for dialog action pairs
/// (B3–B6's "Cancel" / "Add"/"Save") — PrimaryButton's 50pt emphasis
/// height is reserved for onboarding's single "start" action per
/// design-spec.md, so dialogs get this standard-height variant instead.
struct PrimaryButtonCompact: View {
    let title: String
    var isEnabled: Bool = true
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(Theme.Font.headline)
                .padding(.horizontal, 20)
                .frame(height: Theme.Size.control)
                .foregroundStyle(Theme.Color.onAccent)
                .background(Theme.Color.accent.opacity(isEnabled ? 1 : 0.35))
                .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))
        }
        .buttonStyle(.plain)
        .disabled(!isEnabled)
    }
}

#Preview {
    AddContactSheet(core: MockCore.self, chatStore: ChatStore(fileName: "preview.json"))
}
