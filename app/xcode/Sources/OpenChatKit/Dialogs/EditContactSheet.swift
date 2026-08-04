import SwiftUI
#if os(iOS)
import UIKit
#elseif os(macOS)
import AppKit
#endif

/// B4 — Edit contact (design-spec.md §3 B4).
struct EditContactSheet: View {
    let contact: Contact
    let chatStore: ChatStore

    @Environment(\.dismiss) private var dismiss
    @State private var name: String
    @FocusState private var nameFocused: Bool

    init(contact: Contact, chatStore: ChatStore) {
        self.contact = contact
        self.chatStore = chatStore
        _name = State(initialValue: contact.displayName)
    }

    var body: some View {
        VStack(spacing: Theme.Spacing.group) {
            Text("Edit contact")
                .font(Theme.Font.title)
                .foregroundStyle(Theme.Color.textPrimary)

            ContactAvatar(label: name.avatarInitial, seed: contact.address, diameter: Theme.Size.avatarDialog)

            LabeledField(label: "Name", placeholder: "Name", text: $name)
                .focused($nameFocused)

            addressBlock

            Spacer(minLength: 0)

            HStack(spacing: 12) {
                Spacer()
                PlainTextButton(title: "Cancel", color: Theme.Color.textPrimary) { dismiss() }
                PrimaryButtonCompact(title: "Save", isEnabled: !name.trimmingCharacters(in: .whitespaces).isEmpty, action: save)
            }
        }
        .padding(24)
        // See AddContactSheet's note on why this is a fixed macOS width
        // rather than min/ideal.
        #if os(macOS)
        // Two chained calls, not `.frame(width:minHeight:)` — SwiftUI has
        // no overload mixing a fixed `width:` with `minHeight:` in one
        // call (that combination only exists alongside minWidth/maxWidth
        // too); the single-call version fails with "Extra argument
        // 'minHeight' in call".
        .frame(width: 360)
        .frame(minHeight: 320)
        #else
        .presentationDetents([.medium])
        #endif
        .onAppear { nameFocused = true }
    }

    private var addressBlock: some View {
        HStack {
            Text(Contact.placeholderName(for: contact.address))
                .font(Theme.Font.monoCaption)
                .foregroundStyle(Theme.Color.textPrimary)
                .lineLimit(1)
            Spacer(minLength: 8)
            Button {
                copyAddress()
            } label: {
                Image(systemName: "doc.on.doc")
                    .font(.system(size: 13))
                    .foregroundStyle(Theme.Color.textSecondary)
            }
            .buttonStyle(.plain)
        }
        .padding(12)
        .background(Theme.Color.bgSurface2)
        .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))
    }

    private func copyAddress() {
        #if os(iOS)
        UIPasteboard.general.string = contact.address
        #elseif os(macOS)
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(contact.address, forType: .string)
        #endif
    }

    private func save() {
        var updated = contact
        updated.displayName = name.trimmingCharacters(in: .whitespaces)
        chatStore.upsertContact(updated)
        dismiss()
    }
}

#Preview {
    EditContactSheet(contact: Contact(address: "0x4f2a9e1c" + String(repeating: "a", count: 32), x25519Hex: "", displayName: "Alex"), chatStore: ChatStore(fileName: "preview.json"))
}
