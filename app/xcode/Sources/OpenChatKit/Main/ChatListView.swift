import SwiftUI

/// B1's sidebar/list column (design-spec.md §B1 "Sidebar / list column"):
/// status strip, 64pt contact rows, empty state. Selection is a binding
/// so both the macOS/iPad split-view and the iPhone push-navigation
/// shells (see MainSplitView) can drive the same list.
struct ChatListView: View {
    @ObservedObject var session: SessionModel
    /// Observed separately from `session` — `ChatStore` is its own
    /// `ObservableObject` (see its doc comment), so this view needs to
    /// subscribe to it directly for contact/message changes to trigger
    /// a re-render; `session`'s own `@Published` properties (status,
    /// selectedContact) don't forward chatStore's changes.
    @ObservedObject var chatStore: ChatStore
    @Binding var selection: Contact?
    /// True on iPhone's `NavigationStack` (MainSplitView.stackLayout):
    /// rows become `NavigationLink(value:)` so iOS gets a real disclosure
    /// chevron and a reliable push, driven by `.navigationDestination(for:)`
    /// at the call site. False on macOS's `HSplitView` and iPad's
    /// `NavigationSplitView`, where `List`'s own `selection` binding
    /// alone drives the detail column — no push, no chevron, that's
    /// already correct there.
    var asNavigationLinks: Bool = false
    let onAddContact: () -> Void
    let onEditContact: (Contact) -> Void

    var body: some View {
        VStack(spacing: 0) {
            StatusStrip(status: session.status)

            if chatStore.contacts.isEmpty {
                emptyState
            } else if asNavigationLinks {
                // Plain `List`, deliberately with no `selection:` binding
                // here — `List(selection:)` combined with
                // `NavigationLink(value:)` rows in the same list is a
                // known bad combination: the List's own single-selection
                // tap handling competes with the NavigationLink's, and in
                // practice the push just silently never fires (the row
                // only shows its selected/highlighted tint, same as a
                // `List(selection:)` row would after being tapped,
                // instead of actually navigating). Navigation here is
                // driven entirely by `NavigationLink(value:)` +
                // `.navigationDestination(for:)` at the call site
                // (MainSplitView.stackLayout), which doesn't need a
                // selection binding at all.
                List {
                    rows
                }
                .listStyle(.plain)
            } else {
                List(selection: $selection) {
                    rows
                }
                .listStyle(.plain)
            }
        }
        .background(Theme.Color.bgCanvas)
    }

    @ViewBuilder
    private var rows: some View {
        ForEach(chatStore.contacts) { contact in
            Group {
                if asNavigationLinks {
                    NavigationLink(value: contact) {
                        ChatListRow(contact: contact, onEdit: { onEditContact(contact) })
                    }
                } else {
                    ChatListRow(contact: contact, onEdit: { onEditContact(contact) })
                        .tag(contact)
                }
            }
            .listRowInsets(EdgeInsets())
            .listRowSeparator(.hidden)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Spacer()
            Image(systemName: "person.2")
                .font(.system(size: 32))
                .foregroundStyle(Theme.Color.textTertiary)
            Text("No contacts yet")
                .font(Theme.Font.headline)
                .foregroundStyle(Theme.Color.textPrimary)
            Text("Add someone to start chatting")
                .font(Theme.Font.subheadline)
                .foregroundStyle(Theme.Color.textSecondary)
            SecondaryButton(title: "Add contact", height: Theme.Size.control, action: onAddContact)
                .frame(maxWidth: 220)
                .padding(.top, 8)
            Spacer()
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.horizontal, Theme.Spacing.group)
    }
}

/// One 64pt row: 40pt avatar + name, truncating-middle for fallback
/// address names. Edit affordance is platform-native rather than one
/// shared control: hover-reveal pencil on macOS (design-spec.md B1),
/// swipe-to-edit on iOS (the row itself is a `NavigationLink` there —
/// see `ChatListView.asNavigationLinks` — so it also needs to stay free
/// of a permanently-visible trailing button to get its disclosure
/// chevron and tap-to-push back).
private struct ChatListRow: View {
    let contact: Contact
    let onEdit: () -> Void

    @State private var isHovering = false

    var body: some View {
        HStack(spacing: 12) {
            ContactAvatar(label: contact.displayName.avatarInitial, seed: contact.address)

            Text(contact.displayName)
                .font(Theme.Font.headline)
                .foregroundStyle(Theme.Color.textPrimary)
                .lineLimit(1)
                .truncationMode(.middle)

            Spacer(minLength: 0)

            #if os(macOS)
            Button(action: onEdit) {
                Image(systemName: "pencil")
                    .font(.system(size: 14))
                    .foregroundStyle(Theme.Color.textSecondary)
                    .frame(width: 44, height: 44)
            }
            .buttonStyle(.plain)
            .opacity(isHovering ? 1 : 0)
            #endif
        }
        .padding(.leading, 16)
        .padding(.trailing, 4)
        .frame(height: 64)
        .contentShape(Rectangle())
        .onHover { isHovering = $0 }
        #if os(iOS)
        .swipeActions(edge: .trailing) {
            Button(action: onEdit) {
                Label("Edit", systemImage: "pencil")
            }
            .tint(Theme.Color.accent)
        }
        #endif
    }
}
