import SwiftUI
#if os(iOS)
import UIKit
#elseif os(macOS)
import AppKit
#endif

/// B5 — My contact code (design-spec.md §3 B5).
struct MyCodeSheet: View {
    let contactCode: String
    let address: String

    @Environment(\.dismiss) private var dismiss
    @State private var justCopiedCode = false
    @State private var justCopiedAddress = false

    var body: some View {
        VStack(alignment: .leading, spacing: Theme.Spacing.group) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("My code")
                        .font(Theme.Font.title)
                        .foregroundStyle(Theme.Color.textPrimary)
                    Text("Share this so others can message you.")
                        .font(Theme.Font.subheadline)
                        .foregroundStyle(Theme.Color.textSecondary)
                }
                Spacer()
                #if os(macOS)
                Button("Close") { dismiss() }
                #endif
            }

            Text(Self.grouped(contactCode))
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

            SecondaryButton(title: justCopiedCode ? "Copied ✓" : "Copy to clipboard", height: Theme.Size.control) {
                copy(contactCode)
                flashCopied($justCopiedCode)
            }

            Rectangle()
                .fill(Theme.Color.borderHairline)
                .frame(height: 1)
                .padding(.vertical, 4)

            VStack(alignment: .leading, spacing: 4) {
                Text("Your address")
                    .font(Theme.Font.caption)
                    .foregroundStyle(Theme.Color.textSecondary)
                HStack {
                    Text(address)
                        .font(Theme.Font.monoCaption)
                        .foregroundStyle(Theme.Color.textPrimary)
                        .textSelection(.enabled)
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Spacer(minLength: 8)
                    Button {
                        copy(address)
                        flashCopied($justCopiedAddress)
                    } label: {
                        Image(systemName: justCopiedAddress ? "checkmark" : "doc.on.doc")
                            .font(.system(size: 12))
                            .foregroundStyle(Theme.Color.textSecondary)
                    }
                    .buttonStyle(.plain)
                }
            }

            #if os(iOS)
            SecondaryButton(title: "Close", height: Theme.Size.control) { dismiss() }
            #endif
        }
        .padding(24)
        // See AddContactSheet's note on why this is a fixed macOS width
        // rather than min/ideal.
        #if os(macOS)
        // Two chained calls, not `.frame(width:minHeight:)` — see
        // EditContactSheet's note: SwiftUI has no overload mixing a fixed
        // `width:` with `minHeight:` in a single call.
        .frame(width: 440)
        .frame(minHeight: 380)
        #else
        .presentationDetents([.large])
        #endif
    }

    /// Breaks a one-long-string code into 4-char groups (design-spec.md
    /// B5: "break at fixed character intervals e.g. every 4 chars with a
    /// hairline-thin visual grouping, not raw unbroken wrap").
    private static func grouped(_ s: String) -> String {
        var out = ""
        for (i, ch) in s.enumerated() {
            if i > 0 && i % 4 == 0 { out += " " }
            out.append(ch)
        }
        return out
    }

    private func copy(_ text: String) {
        #if os(iOS)
        UIPasteboard.general.string = text
        #elseif os(macOS)
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
        #endif
    }

    /// Sets `flag` true, then back to false after 1.5s (design-spec.md
    /// B5: "label swaps to 'Copied' + checkmark for 1.5s, then reverts").
    private func flashCopied(_ flag: Binding<Bool>) {
        flag.wrappedValue = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { flag.wrappedValue = false }
    }
}

#Preview {
    MyCodeSheet(contactCode: "oc1:4f2a9e1c7b3d5a8f0e2c1b9a7d3f5e8c", address: "0x4f2a9e1c7b3d5a8f0e2c1b9a7d3f5e8c9a1b2c3d")
}
