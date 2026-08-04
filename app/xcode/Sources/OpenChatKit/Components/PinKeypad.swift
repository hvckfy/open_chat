import SwiftUI

/// The dot-mask progress row above `PinKeypad` (design-spec.md A4/A5):
/// `filled` solid circles followed by `total - filled` outlined ones.
/// A5's error state passes `tint: Theme.Color.danger` and the caller
/// wraps this in the ±6pt/120ms shake described in the spec.
struct PinDotsRow: View {
    let filled: Int
    let total: Int
    var tint: Color = Theme.Color.textPrimary
    var dotSize: CGFloat = 14

    var body: some View {
        HStack(spacing: 14) {
            ForEach(0..<total, id: \.self) { i in
                Circle()
                    .strokeBorder(tint, lineWidth: 1.5)
                    .background(Circle().fill(i < filled ? tint : .clear))
                    .frame(width: dotSize, height: dotSize)
            }
        }
    }
}

/// The shared round numeric keypad used identically on A4 (Set a PIN)
/// and A5 (Unlock) and everywhere else PIN entry occurs, on every
/// platform (design-spec.md A4: "including macOS/Windows/Linux where it
/// replaces relying on the physical keyboard, so PIN entry reads
/// identically everywhere"). 3×4 grid: 1–9, blank, 0, delete.
///
/// This view is purely presentational — it reports taps via `onDigit`/
/// `onDelete` and holds no state of its own; the caller owns the current
/// PIN string and decides when it's "complete" (A4: after Confirm PIN
/// matches; A5: auto-submits once the expected length is reached, per
/// the spec's "standard native passcode-entry pattern").
struct PinKeypad: View {
    var keyDiameter: CGFloat = platformDefaultKeyDiameter
    let onDigit: (Int) -> Void
    let onDelete: () -> Void

    private static var platformDefaultKeyDiameter: CGFloat {
        #if os(macOS)
        Theme.Size.pinKeyDiameterRegular
        #else
        Theme.Size.pinKeyDiameterCompact
        #endif
    }

    private let rows: [[Key]] = [
        [.digit(1), .digit(2), .digit(3)],
        [.digit(4), .digit(5), .digit(6)],
        [.digit(7), .digit(8), .digit(9)],
        [.blank, .digit(0), .delete],
    ]

    private enum Key: Hashable {
        case digit(Int)
        case blank
        case delete
    }

    var body: some View {
        VStack(spacing: 20) {
            ForEach(rows.indices, id: \.self) { r in
                HStack(spacing: 26) {
                    ForEach(rows[r], id: \.self) { key in
                        keyView(key)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func keyView(_ key: Key) -> some View {
        switch key {
        case .digit(let d):
            Button {
                onDigit(d)
            } label: {
                Circle()
                    .fill(Theme.Color.bgSurface2)
                    .overlay(Circle().strokeBorder(Theme.Color.borderHairline, lineWidth: 1))
                    .overlay(
                        Text("\(d)")
                            .font(.system(size: keyDiameter * 0.4, weight: .medium))
                            .foregroundStyle(Theme.Color.textPrimary)
                    )
                    .frame(width: keyDiameter, height: keyDiameter)
            }
            .buttonStyle(.plain)

        case .blank:
            Color.clear.frame(width: keyDiameter, height: keyDiameter)

        case .delete:
            Button {
                onDelete()
            } label: {
                Image(systemName: "delete.left")
                    .font(.system(size: keyDiameter * 0.34))
                    .foregroundStyle(Theme.Color.textSecondary)
                    .frame(width: keyDiameter, height: keyDiameter)
            }
            .buttonStyle(.plain)
        }
    }
}

/// Lets a physical/external keyboard drive PIN entry alongside
/// `PinKeypad`'s on-screen taps — digit keys 0–9 and Delete/Backspace,
/// wired through the same `onDigit`/`onDelete` callbacks so a screen
/// using this only ever has one place that decides what a digit/delete
/// means. Mac users in particular expect the number row to just work;
/// the on-screen keypad stays the primary, always-present affordance
/// per design-spec.md A4 ("including macOS... where it replaces relying
/// on the physical keyboard") — this only adds the keyboard as a second,
/// optional path into the same state.
private struct PinKeyboardInput: ViewModifier {
    let onDigit: (Int) -> Void
    let onDelete: () -> Void

    @FocusState private var isFocused: Bool

    func body(content: Content) -> some View {
        content
            .focusable()
            .focused($isFocused)
            .onAppear { isFocused = true }
            .onKeyPress(characters: .decimalDigits) { press in
                guard let character = press.characters.first, let digit = character.wholeNumberValue else {
                    return .ignored
                }
                onDigit(digit)
                return .handled
            }
            .onKeyPress(.delete) {
                onDelete()
                return .handled
            }
    }
}

extension View {
    /// See `PinKeyboardInput`.
    func pinKeyboardInput(onDigit: @escaping (Int) -> Void, onDelete: @escaping () -> Void) -> some View {
        modifier(PinKeyboardInput(onDigit: onDigit, onDelete: onDelete))
    }
}
