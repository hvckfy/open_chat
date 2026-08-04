import SwiftUI
#if os(iOS)
import UIKit
#elseif os(macOS)
import AppKit
#endif

/// B1's detail/conversation column (design-spec.md §B1 "Conversation
/// column") + B2 message bubbles. Bottom-anchored scroll, composer bar,
/// empty-conversation state. The no-selection state (macOS/iPad only)
/// lives in `NoSelectionView` below, shown by MainSplitView instead of
/// this view when nothing is picked.
struct ConversationView: View {
    @ObservedObject var session: SessionModel
    /// See ChatListView's doc comment on `chatStore` — observed
    /// separately from `session` so new/changed messages re-render.
    @ObservedObject var chatStore: ChatStore
    let contact: Contact
    /// Edit-contact affordance. On iOS this lives in the conversation's
    /// own header (design-code.md feedback: "so I know who I'm talking
    /// to, put the edit button here") next to a compact avatar+name —
    /// on macOS the sidebar row's hover-pencil already covers it, so
    /// this simply isn't shown there.
    var onEdit: () -> Void = {}

    @State private var composerText = ""
    @State private var sendError: String?

    private var messages: [StoredMessage] {
        chatStore.messages(for: contact.address)
    }

    var body: some View {
        VStack(spacing: 0) {
            messageList
            composer
        }
        .background(Theme.Color.bgCanvas)
        #if os(macOS)
        .navigationTitle(contact.displayName)
        #else
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                HStack(spacing: 8) {
                    ContactAvatar(label: contact.displayName.avatarInitial, seed: contact.address, diameter: 28)
                    Text(contact.displayName)
                        .font(Theme.Font.headline)
                        .foregroundStyle(Theme.Color.textPrimary)
                        .lineLimit(1)
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button(action: onEdit) {
                    Image(systemName: "pencil")
                }
            }
        }
        #endif
    }

    private var messageList: some View {
        ScrollViewReader { proxy in
            ScrollView {
                if messages.isEmpty {
                    emptyConversation
                } else {
                    LazyVStack(alignment: .leading, spacing: 4) {
                        ForEach(Array(messages.enumerated()), id: \.element.id) { index, message in
                            let previous = index > 0 ? messages[index - 1] : nil
                            let directionChanged = previous?.isOutgoing != message.isOutgoing
                            MessageBubble(text: message.text, timestamp: message.date, isOutgoing: message.isOutgoing) {
                                #if os(iOS)
                                UIPasteboard.general.string = message.text
                                #elseif os(macOS)
                                NSPasteboard.general.clearContents()
                                NSPasteboard.general.setString(message.text, forType: .string)
                                #endif
                            }
                            .padding(.top, previous == nil ? 0 : (directionChanged ? 12 : 4))
                            .id(message.id)
                        }
                    }
                    .padding(16)
                }
            }
            .onChange(of: messages.count) { _, _ in
                if let last = messages.last {
                    withAnimation { proxy.scrollTo(last.id, anchor: .bottom) }
                }
            }
            .onAppear {
                if let last = messages.last {
                    proxy.scrollTo(last.id, anchor: .bottom)
                }
            }
        }
    }

    private var emptyConversation: some View {
        VStack {
            Spacer(minLength: 120)
            Text("No messages yet — say hello")
                .font(Theme.Font.subheadline)
                .foregroundStyle(Theme.Color.textSecondary)
            Spacer(minLength: 120)
        }
        .frame(maxWidth: .infinity)
    }

    private var composer: some View {
        VStack(spacing: 0) {
            Rectangle()
                .fill(Theme.Color.borderHairline)
                .frame(height: 1)

            HStack(alignment: .bottom, spacing: 8) {
                ZStack(alignment: .topLeading) {
                    if composerText.isEmpty {
                        Text("Message")
                            .font(Theme.Font.body)
                            .foregroundStyle(Theme.Color.textTertiary)
                            // Same fix as LabeledMultilineField (see its
                            // comment): TextEditor's built-in top inset is
                            // ~8pt on iOS (UITextView) but only ~1pt on
                            // macOS (NSTextView), so one hardcoded value
                            // put this well below the actual caret on
                            // macOS specifically.
                            #if os(macOS)
                            .padding(.top, 1)
                            #else
                            .padding(.top, 8)
                            #endif
                            .padding(.leading, 5)
                            .allowsHitTesting(false)
                    }
                    TextEditor(text: $composerText)
                        .font(Theme.Font.body)
                        // 28pt inner height + this ZStack's 8pt padding on
                        // each side (below) = 44pt total — level with the
                        // send button, per design; grows toward
                        // `maxHeight` as content wraps, and since the
                        // enclosing HStack aligns `.bottom`, growth reads
                        // as "expanding upward" rather than pushing the
                        // send button down.
                        //
                        // `.frame(minHeight:maxHeight:)` alone does NOT
                        // make this shrink back down to 28pt when empty —
                        // TextEditor's own ideal/proposed height (an
                        // NSScrollView/UIScrollView under the hood) is
                        // "as much as offered", so the surrounding layout
                        // just handed it the full 120pt every time,
                        // regardless of how little text was in it. Adding
                        // `.fixedSize(horizontal: false, vertical: true)`
                        // *after* the frame forces it to report its actual
                        // content height as its ideal size instead — the
                        // frame's min/max still clamp that between 28 and
                        // 120, so it grows/shrinks with the text and only
                        // starts internally scrolling once it hits the cap.
                        .frame(minHeight: 28, maxHeight: 120)
                        .fixedSize(horizontal: false, vertical: true)
                        .scrollContentBackground(.hidden)
                        .scrollIndicators(.hidden)
                        // `.onKeyPress(.return) { ... }` (the single-
                        // KeyEquivalent overload) hands back a closure with
                        // *no* parameters — there's no way to inspect
                        // modifiers from it, so Shift+Return couldn't be
                        // told apart from plain Return. `onKeyPress(keys:)`
                        // passes the full `KeyPress` (with `.modifiers`)
                        // into the closure instead.
                        .onKeyPress(keys: [.return]) { press in
                            guard !press.modifiers.contains(.shift) else { return .ignored }
                            send()
                            return .handled
                        }
                }
                .padding(8)
                .background(Theme.Color.bgSurface)
                .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))

                Button(action: send) {
                    Circle()
                        .fill(Theme.Color.accent.opacity(canSend ? 1 : 0.35))
                        .frame(width: 44, height: 44)
                        .overlay(
                            Image(systemName: "arrow.up")
                                .font(.system(size: 16, weight: .semibold))
                                .foregroundStyle(Theme.Color.onAccent)
                        )
                }
                .buttonStyle(.plain)
                .disabled(!canSend)
            }
            .padding(12)

            if let sendError {
                Text(sendError)
                    .font(Theme.Font.caption)
                    .foregroundStyle(Theme.Color.danger)
                    .padding(.horizontal, 12)
                    .padding(.bottom, 8)
            }
        }
        .background(Theme.Color.bgSurface2)
    }

    private var canSend: Bool {
        !composerText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private func send() {
        guard canSend else { return }
        let text = composerText.trimmingCharacters(in: .whitespacesAndNewlines)
        composerText = ""
        sendError = nil
        Task {
            do {
                try await session.sendText(text, to: contact)
            } catch {
                sendError = error.localizedDescription
                composerText = text
            }
        }
    }
}

/// macOS/iPad-only: shown by MainSplitView's detail column when no
/// contact is selected (design-spec.md §B1).
struct NoSelectionView: View {
    var body: some View {
        VStack(spacing: 8) {
            Image(systemName: "bubble.left.and.bubble.right")
                .font(.system(size: 32))
                .foregroundStyle(Theme.Color.textTertiary)
            Text("Select a conversation")
                .font(Theme.Font.subheadline)
                .foregroundStyle(Theme.Color.textSecondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Theme.Color.bgCanvas)
    }
}
