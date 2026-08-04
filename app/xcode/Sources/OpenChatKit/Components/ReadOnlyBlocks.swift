import SwiftUI

/// The recurring "important read-only text" treatment (recovery phrase,
/// contact code, addresses): `bg.surface2` panel, monospaced,
/// `text.primary` at full contrast — **never** a platform "disabled"
/// style, per design-code.md's contrast requirement (a disabled-looking
/// control was a real bug here once). `.textSelection(.enabled)` gives
/// native copy/selection for free without a fake editable field.

/// A2's recovery-phrase card: 24 words in a 3×8 grid (2×12 on iPhone —
/// pass `columns: 2`), each prefixed with its 1-based index.
struct RecoveryPhraseCard: View {
    let words: [String]
    var columns: Int = 3

    private var gridColumns: [GridItem] {
        Array(repeating: GridItem(.flexible(), alignment: .leading), count: columns)
    }

    var body: some View {
        LazyVGrid(columns: gridColumns, alignment: .leading, spacing: 10) {
            ForEach(Array(words.enumerated()), id: \.offset) { index, word in
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text("\(index + 1).")
                        .font(Theme.Font.caption)
                        .foregroundStyle(Theme.Color.textSecondary)
                        .frame(width: 18, alignment: .trailing)
                    Text(word)
                        .font(Theme.Font.mono)
                        .foregroundStyle(Theme.Color.textPrimary)
                }
            }
        }
        .padding(Theme.Spacing.group + 2) // 18pt per spec's phrase-card padding
        .background(Theme.Color.bgSurface2)
        .overlay(
            RoundedRectangle(cornerRadius: Theme.Radius.large, style: .continuous)
                .strokeBorder(Theme.Color.borderHairline, lineWidth: 1)
        )
        .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.large, style: .continuous))
        .textSelection(.enabled)
    }
}

/// B5's contact-code card: one long string, broken into fixed-width
/// groups (a hairline-thin visual grouping, not a raw unbroken wrap —
/// design-spec.md §B5) rather than shown as a single run-on line.
struct ReadOnlyCodeCard: View {
    let code: String
    var groupEvery: Int = 4

    private var grouped: String {
        var out = ""
        for (i, ch) in code.enumerated() {
            if i > 0 && i % groupEvery == 0 { out += " " }
            out.append(ch)
        }
        return out
    }

    var body: some View {
        Text(grouped)
            .font(Theme.Font.mono)
            .foregroundStyle(Theme.Color.textPrimary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
            .background(Theme.Color.bgSurface2)
            .overlay(
                RoundedRectangle(cornerRadius: Theme.Radius.large, style: .continuous)
                    .strokeBorder(Theme.Color.borderHairline, lineWidth: 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.large, style: .continuous))
            .textSelection(.enabled)
    }
}

/// B4's read-only address block: smaller/quieter than the two cards
/// above (12pt padding, 10pt radius), truncated-middle, with a small
/// trailing copy icon.
struct ReadOnlyAddressBlock: View {
    let address: String
    var onCopy: (() -> Void)? = nil

    var body: some View {
        HStack(spacing: 8) {
            Text(address)
                .font(Theme.Font.monoCaption)
                .foregroundStyle(Theme.Color.textPrimary)
                .lineLimit(1)
                .truncationMode(.middle)
                .textSelection(.enabled)
            if let onCopy {
                Spacer(minLength: 8)
                Button(action: onCopy) {
                    Image(systemName: "doc.on.doc")
                        .font(.caption)
                        .foregroundStyle(Theme.Color.textSecondary)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(12)
        .background(Theme.Color.bgSurface2)
        .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))
    }
}
