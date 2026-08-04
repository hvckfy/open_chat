import SwiftUI
import UniformTypeIdentifiers

/// B6 — Network settings (design-spec.md §3 B6). Saving doesn't apply
/// settings live (the underlying `NetworkSession` is built once at
/// login — see SessionModel.init) — surfaces the "restart to apply"
/// banner the spec calls for instead of pretending otherwise.
struct NetworkSettingsSheet: View {
    let initial: NetworkSettings
    let onSave: (NetworkSettings) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var bootstrapText: String
    @State private var caCertPath: String
    @State private var insecureSkipVerify: Bool
    @State private var showFileImporter = false
    @State private var showRestartBanner = false

    init(initial: NetworkSettings, onSave: @escaping (NetworkSettings) -> Void) {
        self.initial = initial
        self.onSave = onSave
        _bootstrapText = State(initialValue: initial.bootstrapCSV)
        _caCertPath = State(initialValue: initial.caCertPath)
        _insecureSkipVerify = State(initialValue: initial.insecureSkipVerify)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Network settings")
                .font(Theme.Font.title)
                .foregroundStyle(Theme.Color.textPrimary)
                .padding(.bottom, Theme.Spacing.group)

            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    section("Gateways") {
                        LabeledMultilineField(label: "Bootstrap gateways", placeholder: "host:port, host2:port2", text: $bootstrapText, minHeight: 80)
                        Text("Comma-separated host:port list. Leave blank to use the built-in defaults.")
                            .font(Theme.Font.caption)
                            .foregroundStyle(Theme.Color.textSecondary)
                    }

                    section("Certificate") {
                        #if os(macOS)
                        // Typing a path is reasonable on Mac (Terminal-
                        // adjacent audience, real filesystem paths).
                        HStack(alignment: .bottom, spacing: 8) {
                            LabeledField(label: "CA certificate path", placeholder: "/path/to/ca.pem", text: $caCertPath)
                            Button("Choose file…") { showFileImporter = true }
                        }
                        #else
                        // iOS has no meaningful typed "path" a user could
                        // enter — files only ever come from the picker —
                        // so there's no text field here at all, just the
                        // picked filename (or "No file selected") and the
                        // button that's the only way to change it.
                        VStack(alignment: .leading, spacing: 4) {
                            Text("CA certificate").font(Theme.Font.caption).foregroundStyle(Theme.Color.textSecondary)
                            HStack {
                                Text(caCertPath.isEmpty ? "No file selected" : (caCertPath as NSString).lastPathComponent)
                                    .font(Theme.Font.body)
                                    .foregroundStyle(caCertPath.isEmpty ? Theme.Color.textTertiary : Theme.Color.textPrimary)
                                    .lineLimit(1)
                                    .truncationMode(.middle)
                                Spacer(minLength: 8)
                                Button("Choose file…") { showFileImporter = true }
                            }
                        }
                        #endif
                    }

                    dangerZone

                    if showRestartBanner {
                        restartBanner
                    }
                }
            }

            Spacer(minLength: 0)

            HStack(spacing: 12) {
                Spacer()
                PlainTextButton(title: "Cancel", color: Theme.Color.textPrimary) { dismiss() }
                PrimaryButtonCompact(title: "Save", action: save)
            }
            .padding(.top, Theme.Spacing.group)
        }
        .padding(24)
        // See AddContactSheet's note on why this is a fixed macOS width
        // rather than min/ideal.
        #if os(macOS)
        // Two chained calls, not `.frame(width:minHeight:)` — see
        // EditContactSheet's note: SwiftUI has no overload mixing a fixed
        // `width:` with `minHeight:` in a single call.
        .frame(width: 480)
        .frame(minHeight: 480)
        #else
        .presentationDetents([.large])
        #endif
        .fileImporter(isPresented: $showFileImporter, allowedContentTypes: [.item], allowsMultipleSelection: false) { result in
            if case .success(let urls) = result, let url = urls.first {
                caCertPath = url.path
            }
        }
    }

    private func section<Content: View>(_ title: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Theme.Spacing.tight) {
            Text(title.uppercased())
                .font(Theme.Font.caption)
                .foregroundStyle(Theme.Color.textSecondary)
            content()
        }
    }

    /// Section 3, deliberately visually distinct (design-spec.md B6:
    /// "must look distinct from sections 1–2, not just another row").
    private var dangerZone: some View {
        VStack(alignment: .leading, spacing: 8) {
            Toggle("Skip TLS verification", isOn: $insecureSkipVerify)
                .font(Theme.Font.body)
                .foregroundStyle(Theme.Color.textPrimary)
            Text("Only for local/dev networks. Leave off otherwise.")
                .font(Theme.Font.caption)
                .foregroundStyle(Theme.Color.warning)
        }
        .padding(12)
        .background(Theme.Color.warning.opacity(0.12))
        .overlay(
            RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous)
                .strokeBorder(Theme.Color.warning.opacity(0.35), lineWidth: 1)
        )
        .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))
    }

    private var restartBanner: some View {
        Text("Restart OpenChat to apply network changes")
            .font(Theme.Font.caption)
            .foregroundStyle(Theme.Color.textPrimary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(10)
            .background(Theme.Color.warning.opacity(0.18))
            .clipShape(RoundedRectangle(cornerRadius: Theme.Radius.small, style: .continuous))
    }

    private func save() {
        let gateways = bootstrapText.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
        let settings = NetworkSettings(bootstrapGateways: gateways, caCertPath: caCertPath, insecureSkipVerify: insecureSkipVerify)
        onSave(settings)
        showRestartBanner = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 4) { showRestartBanner = false }
    }
}

#Preview {
    NetworkSettingsSheet(initial: NetworkSettings(), onSave: { _ in })
}
