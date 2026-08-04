import SwiftUI

/// A5 — Unlock (design-spec.md §3 A5). Returning-user re-entry: lower
/// ceremony than A1, auto-submits once the PIN reaches its expected
/// length (standard native passcode-entry pattern) — no separate
/// "Unlock" button.
struct UnlockView: View {
    var errorMessage: String?
    var isBusy: Bool = false
    let onUnlock: (String) -> Void
    let onUseDifferentPhrase: () -> Void

    @State private var pin = ""
    @State private var shakeTrigger = 0

    private let pinLength = 4

    var body: some View {
        VStack(spacing: 0) {
            #if os(iOS)
            Spacer().frame(height: 120)
            #else
            Spacer()
            #endif

            RoundedRectangle(cornerRadius: 9, style: .continuous)
                .fill(Theme.Color.accent)
                .frame(width: 32, height: 32)
                .padding(.bottom, 20)

            PinDotsRow(filled: pin.count, total: pinLength, tint: errorMessage != nil ? Theme.Color.danger : Theme.Color.textPrimary)
                .modifier(ShakeEffect(trigger: shakeTrigger))

            if let errorMessage {
                Text(errorMessage)
                    .font(Theme.Font.caption)
                    .foregroundStyle(Theme.Color.danger)
                    .padding(.top, 8)
            } else if isBusy {
                Text(" ")
                    .font(Theme.Font.caption)
                    .padding(.top, 8)
            }

            Spacer()

            PinKeypad(onDigit: handleDigit, onDelete: handleDelete)
                .opacity(isBusy ? 0.5 : 1)
                .allowsHitTesting(!isBusy)
                .padding(.bottom, 24)

            PlainTextButton(title: "Log out and use a different recovery phrase", color: Theme.Color.textSecondary, action: onUseDifferentPhrase)
                .font(Theme.Font.caption)
                .padding(.bottom, 16)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Theme.Color.bgCanvas)
        .onChange(of: errorMessage) { _, newValue in
            if newValue != nil {
                pin = ""
                shakeTrigger += 1
            }
        }
        .pinKeyboardInput(onDigit: handleDigit, onDelete: handleDelete)
    }

    private func handleDigit(_ d: Int) {
        guard pin.count < pinLength, !isBusy else { return }
        pin.append(String(d))
        if pin.count == pinLength {
            onUnlock(pin)
        }
    }

    private func handleDelete() {
        guard !isBusy, !pin.isEmpty else { return }
        pin.removeLast()
    }
}

/// A2's shake: ±6pt, 120ms, retriggered by changing `trigger`.
private struct ShakeEffect: ViewModifier {
    let trigger: Int
    @State private var offset: CGFloat = 0

    func body(content: Content) -> some View {
        content
            .offset(x: offset)
            .onChange(of: trigger) { _, _ in
                withAnimation(.easeInOut(duration: 0.06).repeatCount(4, autoreverses: true)) {
                    offset = offset == 0 ? 6 : 0
                }
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.12) { offset = 0 }
            }
    }
}

#Preview {
    UnlockView(errorMessage: nil, onUnlock: { _ in }, onUseDifferentPhrase: {})
}
