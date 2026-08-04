import SwiftUI

/// A4 — Set a PIN (design-spec.md §3 A4). Local-only PIN that encrypts
/// the recovery phrase at rest on this device (see KeychainWalletStore).
/// 4-digit, using the shared PinKeypad — see that type's doc comment and
/// design-spec.md §6's own open question about PIN length/character set;
/// 4 digits matches what the mockup itself shows and is what this
/// reference implementation commits to.
struct SetPINView: View {
    var errorMessage: String?
    var isBusy: Bool = false
    let onSave: (String) -> Void

    private enum Stage { case entering, confirming }
    @State private var stage: Stage = .entering
    @State private var pin = ""
    @State private var confirmPin = ""
    @State private var mismatch = false

    private let pinLength = 4

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Set a PIN")
                .font(Theme.Font.title)
                .foregroundStyle(Theme.Color.textPrimary)
                .padding(.bottom, 6)

            Text("Stays on this device only — not sent anywhere, can't be reset. Forgetting it means re-importing your recovery phrase.")
                .font(Theme.Font.caption)
                .foregroundStyle(Theme.Color.textSecondary)
                .padding(.bottom, 14)

            VStack(spacing: 6) {
                Text("PIN").font(Theme.Font.caption).foregroundStyle(Theme.Color.textSecondary)
                PinDotsRow(filled: pin.count, total: pinLength, dotSize: 16)
                    .padding(.vertical, 6)
            }
            .frame(maxWidth: .infinity)
            .padding(.bottom, 10)

            VStack(spacing: 6) {
                Text("Confirm PIN").font(Theme.Font.caption).foregroundStyle(Theme.Color.textSecondary)
                PinDotsRow(filled: confirmPin.count, total: pinLength, tint: mismatch ? Theme.Color.danger : Theme.Color.textPrimary, dotSize: 16)
                    .padding(.vertical, 6)
            }
            .frame(maxWidth: .infinity)

            if mismatch {
                FieldError(message: "PINs don't match — try again.")
                    .padding(.top, 4)
            } else if let errorMessage {
                FieldError(message: errorMessage).padding(.top, 4)
            }

            PrimaryButton(
                title: "Save & Continue",
                isEnabled: pin.count == pinLength && confirmPin.count == pinLength,
                isLoading: isBusy
            ) {
                submit()
            }
            .padding(.top, 14)

            Spacer(minLength: 0)

            PinKeypad(onDigit: handleDigit, onDelete: handleDelete)
                .frame(maxWidth: .infinity, alignment: .center)
                .padding(.bottom, 24)
        }
        .padding(screenMargin)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(Theme.Color.bgCanvas)
        .pinKeyboardInput(onDigit: handleDigit, onDelete: handleDelete)
    }

    private func handleDigit(_ d: Int) {
        mismatch = false
        switch stage {
        case .entering:
            guard pin.count < pinLength else { return }
            pin.append(String(d))
            if pin.count == pinLength { stage = .confirming }
        case .confirming:
            guard confirmPin.count < pinLength else { return }
            confirmPin.append(String(d))
        }
    }

    private func handleDelete() {
        mismatch = false
        switch stage {
        case .entering:
            if !pin.isEmpty { pin.removeLast() }
        case .confirming:
            if !confirmPin.isEmpty {
                confirmPin.removeLast()
            } else {
                stage = .entering
                if !pin.isEmpty { pin.removeLast() }
            }
        }
    }

    private func submit() {
        guard pin == confirmPin else {
            mismatch = true
            confirmPin = ""
            return
        }
        onSave(pin)
    }

    #if os(iOS)
    private let screenMargin: CGFloat = Theme.Spacing.screenMarginCompact
    #else
    private let screenMargin: CGFloat = Theme.Spacing.screenMarginRegular
    #endif
}

#Preview {
    SetPINView(onSave: { _ in })
}
