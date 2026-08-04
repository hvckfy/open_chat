import SwiftUI

/// The shared text-field chrome used throughout onboarding/dialogs:
/// `bg.surface2` fill, 10pt radius, hairline border that turns `danger`
/// (1.5pt) when `isError` — design-spec.md's field error treatment,
/// applied consistently rather than per-screen.
struct FieldBackground: ViewModifier {
    var isError: Bool = false

    func body(content: Content) -> some View {
        content
            .padding(12)
            .background(Theme.Color.bgSurface2)
            .overlay(
                RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous)
                    .strokeBorder(isError ? Theme.Color.danger : Theme.Color.borderHairline, lineWidth: isError ? 1.5 : 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))
    }
}

extension View {
    func fieldBackground(isError: Bool = false) -> some View {
        modifier(FieldBackground(isError: isError))
    }
}

/// The inline field-error caption (`danger`, directly under the
/// offending field) used everywhere instead of one combined error
/// banner — design-spec.md §3 (A3/A4) and §5.
struct FieldError: View {
    let message: String

    var body: some View {
        Text(message)
            .font(Theme.Font.caption)
            .foregroundStyle(Theme.Color.danger)
    }
}

/// A labeled single-line field (name, "Confirm PIN", etc.) — plain
/// `TextField` with `FieldBackground`, sized to its own content rather
/// than a forced fixed height.
///
/// Two deliberate fixes here vs. a naive `TextField`:
/// `.textFieldStyle(.plain)` removes AppKit's own bezeled-border chrome
/// on macOS — without it, the platform draws *its own* box around the
/// field in addition to `fieldBackground`'s, which is exactly the
/// "extra frame around the field" / cursor-vs-text misalignment bug
/// (a `TextField` stretched taller than its intrinsic size via a fixed
/// `.frame(height:)` doesn't recenter its text within that frame,
/// visibly desyncing the caret from the glyphs — so this also drops
/// that fixed height and lets `fieldBackground`'s padding alone define
/// the box, which ends up "slightly bigger than the text" as intended.
struct LabeledField: View {
    var label: String? = nil
    var placeholder: String = ""
    @Binding var text: String
    var isError: Bool = false
    var isSecure: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let label {
                Text(label)
                    .font(Theme.Font.caption)
                    .foregroundStyle(Theme.Color.textSecondary)
            }
            Group {
                if isSecure {
                    SecureField(placeholder, text: $text)
                } else {
                    TextField(placeholder, text: $text)
                }
            }
            .textFieldStyle(.plain)
            .font(Theme.Font.body)
            .fieldBackground(isError: isError)
            #if os(iOS)
            .textInputAutocapitalization(.never)
            #endif
            .autocorrectionDisabled()
        }
    }
}

/// A multi-line field for longer paste-friendly input (contact codes,
/// recovery-phrase import, bootstrap gateway list) — design-spec.md's
/// "min height 80–120pt, Mono text" fields.
struct LabeledMultilineField: View {
    var label: String? = nil
    var placeholder: String = ""
    @Binding var text: String
    var minHeight: CGFloat = 100
    var isError: Bool = false
    var monospaced: Bool = true

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let label {
                Text(label)
                    .font(Theme.Font.caption)
                    .foregroundStyle(Theme.Color.textSecondary)
            }
            ZStack(alignment: .topLeading) {
                if text.isEmpty {
                    Text(placeholder)
                        .font(monospaced ? Theme.Font.mono : Theme.Font.body)
                        .foregroundStyle(Theme.Color.textTertiary)
                        // TextEditor wraps a plain UITextView on iOS and an
                        // NSTextView on macOS, and the two platforms ship
                        // very different default text-container insets
                        // (UITextView: ~8pt top; NSTextView: ~0-1pt top —
                        // SwiftUI's TextEditor doesn't unify them, and
                        // there's no public modifier to query or override
                        // it). Matching only one platform's number here is
                        // exactly what put the placeholder well below the
                        // actual blinking caret on macOS while looking
                        // fine on iOS — needs its own value per platform.
                        #if os(macOS)
                        .padding(.top, 1)
                        #else
                        .padding(.top, 8)
                        #endif
                        .padding(.leading, 5)
                        .allowsHitTesting(false)
                        // Without this, the placeholder inherits whatever
                        // transaction/animation is in flight when it first
                        // appears (e.g. a sheet's presentation spring) and
                        // visibly lags behind the now-focused text editor
                        // before "catching up" — it should just be there,
                        // at rest, from the first frame.
                        .transaction { $0.animation = nil }
                }
                TextEditor(text: $text)
                    .font(monospaced ? Theme.Font.mono : Theme.Font.body)
                    .frame(minHeight: minHeight)
                    .scrollContentBackground(.hidden)
            }
            .fieldBackground(isError: isError)
        }
    }
}
