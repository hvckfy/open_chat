import SwiftUI

/// B1's connection/history-sync status, shown as a colored dot + short
/// label in the sidebar's status strip (design-spec.md §B1). The only
/// ongoing-async indicator in the whole app by design — see
/// design-spec.md §5: "Loading states are narrow and explicit... no
/// blocking spinners."
enum ConnectionStatus: Equatable {
    case connecting
    case syncing
    case upToDate
    case error(String)

    var label: String {
        switch self {
        case .connecting: return "Connecting…"
        case .syncing: return "Syncing message history…"
        case .upToDate: return "Message history up to date"
        case .error(let message): return message
        }
    }

    var tint: Color {
        switch self {
        case .connecting, .syncing: return Theme.Color.warning
        case .upToDate: return Theme.Color.success
        case .error: return Theme.Color.danger
        }
    }
}

/// 32pt-tall status strip at the top of B1's sidebar/contact list.
struct StatusStrip: View {
    let status: ConnectionStatus

    var body: some View {
        HStack(spacing: 8) {
            Circle()
                .fill(status.tint)
                .frame(width: 8, height: 8)
            Text(status.label)
                .font(Theme.Font.caption)
                .foregroundStyle(Theme.Color.textSecondary)
                .lineLimit(1)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .frame(height: 32)
        .background(Theme.Color.bgSurface2)
    }
}
