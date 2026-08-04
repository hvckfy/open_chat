import SwiftUI

/// A1's primary CTA spec, reused everywhere a screen has one prominent
/// action (A1 "Create a new identity", A2/A4 "Continue"/"Save & Continue",
/// A3 "Import"): full accent fill, 50pt height (taller than the standard
/// 44pt control — this is the one button per screen that should read as
/// "start"), 14pt radius. `isEnabled` false renders per design-spec.md
/// A2: `accent` at 35% opacity and no tap feedback, not a desaturated
/// gray — the button stays identifiable, just inert.
struct PrimaryButton: View {
    let title: String
    var isEnabled: Bool = true
    var isLoading: Bool = false
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            ZStack {
                if isLoading {
                    ProgressView().tint(Theme.Color.onAccent)
                } else {
                    Text(title)
                        .font(Theme.Font.headline)
                }
            }
            .frame(maxWidth: .infinity)
            .frame(height: Theme.Size.primaryButton)
            .foregroundStyle(Theme.Color.onAccent)
            .background(Theme.Color.accent.opacity(isEnabled ? 1 : 0.35))
            .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.medium, style: .continuous))
        }
        .buttonStyle(.plain)
        .disabled(!isEnabled || isLoading)
    }
}

/// The bordered/secondary button style (A1's "I already have a recovery
/// phrase", B5's "Copy to clipboard"): same footprint as PrimaryButton,
/// `bg.surface` fill with a hairline border instead of an accent fill —
/// a supporting action, not the screen's primary CTA.
struct SecondaryButton: View {
    let title: String
    var height: CGFloat = Theme.Size.primaryButton
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(Theme.Font.headline)
                .frame(maxWidth: .infinity)
                .frame(height: height)
                .foregroundStyle(Theme.Color.textPrimary)
                .background(Theme.Color.bgSurface)
                .overlay(
                    RoundedRectangle(cornerRadius: Theme.Radius.medium, style: .continuous)
                        .strokeBorder(Theme.Color.borderHairline, lineWidth: 1)
                )
                .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.medium, style: .continuous))
        }
        .buttonStyle(.plain)
    }
}

/// A plain text link-style button (A3/A5 "Back" / "Log out and use a
/// different recovery phrase"), accent-tinted, no background.
struct PlainTextButton: View {
    let title: String
    var color: SwiftUI.Color = Theme.Color.accent
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(Theme.Font.subheadline.weight(.medium))
                .foregroundStyle(color)
        }
        .buttonStyle(.plain)
    }
}
