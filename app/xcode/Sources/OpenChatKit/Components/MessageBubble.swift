import SwiftUI

/// One message in the conversation view (design-spec.md §B2). The one
/// asymmetric-corner detail — radius 14pt everywhere except a 4pt "tail"
/// corner (bottom-trailing when sent, bottom-leading when received) — is
/// what reads as direction at a glance, on top of alignment/color.
struct MessageBubble: View {
    let text: String
    let timestamp: Date
    let isOutgoing: Bool
    /// Called when the user picks "Copy" from the bubble's context menu
    /// (long-press on iOS, right-click/hover on macOS) — never a
    /// permanent visible icon per bubble, to keep the transcript quiet.
    var onCopy: () -> Void = {}

    private static let timeFormatter: DateFormatter = {
        let f = DateFormatter()
        f.timeStyle = .short
        f.dateStyle = .none
        return f
    }()

    private var shape: UnevenRoundedRectangle {
        UnevenRoundedRectangle(
            topLeadingRadius: Theme.Radius.medium,
            bottomLeadingRadius: isOutgoing ? Theme.Radius.medium : Theme.Radius.bubbleTail,
            bottomTrailingRadius: isOutgoing ? Theme.Radius.bubbleTail : Theme.Radius.medium,
            topTrailingRadius: Theme.Radius.medium,
            style: .continuous
        )
    }

    var body: some View {
        VStack(alignment: isOutgoing ? .trailing : .leading, spacing: 2) {
            Text(text)
                .font(Theme.Font.body)
                .foregroundStyle(isOutgoing ? Theme.Color.onAccent : Theme.Color.textPrimary)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(isOutgoing ? Theme.Color.accent : Theme.Color.bgSurface)
                .overlay {
                    if !isOutgoing {
                        shape.strokeBorder(Theme.Color.borderHairline, lineWidth: 1)
                    }
                }
                .clipShape(shape)
                .contextMenu {
                    Button {
                        onCopy()
                    } label: {
                        Label("Copy", systemImage: "doc.on.doc")
                    }
                }

            Text(Self.timeFormatter.string(from: timestamp))
                .font(Theme.Font.caption)
                .foregroundStyle((isOutgoing ? Theme.Color.onAccent : Theme.Color.textPrimary).opacity(0.6))
        }
        .frame(maxWidth: .infinity, alignment: isOutgoing ? .trailing : .leading)
    }
}
