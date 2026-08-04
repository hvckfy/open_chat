import SwiftUI

/// The circular initial-letter placeholder used in B1's contact list
/// (40×40pt) and B4's Edit contact dialog (64×64pt), tinted by a
/// deterministic hue derived from the contact's address so the same
/// contact always gets the same color across launches, without storing
/// one.
struct ContactAvatar: View {
    /// What to show inside the circle — a single letter for a named
    /// contact, or "0x" for one still known only by its address (see
    /// design-spec.md B1's list-row description).
    let label: String
    /// Used only to derive the fill hue — typically the contact's address.
    let seed: String
    var diameter: CGFloat = Theme.Size.avatarRow

    private var hue: Double {
        var hash: UInt32 = 2166136261
        for byte in seed.utf8 {
            hash = (hash ^ UInt32(byte)) &* 16777619
        }
        return Double(hash % 360) / 360.0
    }

    var body: some View {
        Circle()
            .fill(Color(hue: hue, saturation: 0.42, brightness: 0.72))
            .frame(width: diameter, height: diameter)
            .overlay(
                Text(label)
                    .font(.system(size: diameter * 0.4, weight: .semibold, design: label.count > 1 ? .monospaced : .default))
                    .foregroundStyle(.white)
            )
    }
}

extension String {
    /// The avatar `label` for a display name: its first character
    /// uppercased, or "0x" for a fallback-formatted address name (see
    /// ChatStore.shortAddr on the Go side — the same "0x…" truncation
    /// convention design-spec.md §1.3 asks every platform to share).
    var avatarInitial: String {
        if hasPrefix("0x") { return "0x" }
        guard let first = self.first else { return "?" }
        return String(first).uppercased()
    }
}
