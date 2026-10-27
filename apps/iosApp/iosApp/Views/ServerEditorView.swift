import Shared
import SwiftUI

struct ServerEditorView: View {
    @EnvironmentObject private var model: AppModel
    @Environment(\.dismiss) private var dismiss

    private let existing: ServerProfile?
    private let isImport: Bool
    @State private var name: String
    @State private var server: String
    @State private var clientId: String
    @State private var secret: String
    @State private var obfsProvider: String
    @State private var participation: String
    @State private var performanceMode: String
    @State private var udpEnabled: Bool
    @State private var problems: [ProfileProblem] = []

    /// Edits `profile`, or adds a server; `imported` pre-fills a new server from a scanned link.
    init(profile: ServerProfile?, imported: ServerLink? = nil) {
        existing = profile
        isImport = imported != nil
        let profile = imported?.profile ?? profile
        _secret = State(initialValue: imported?.secret ?? "")
        _name = State(initialValue: profile?.name ?? "")
        _server = State(initialValue: profile?.server ?? "")
        _clientId = State(initialValue: profile?.clientId ?? "")
        _obfsProvider = State(initialValue: profile?.obfsProvider ?? ObfsProviders.shared.NONE)
        _participation = State(initialValue: String(profile?.obfsParticipation ?? 0))
        _performanceMode = State(initialValue: profile?.performanceMode ?? PerformanceModes.shared.INTERACTIVE)
        _udpEnabled = State(initialValue: profile?.udpEnabled ?? true)
    }

    var body: some View {
        Form {
            if isImport {
                Section {
                    Label("Review the imported server, then save.", systemImage: "qrcode")
                        .foregroundStyle(.secondary)
                }
            }
            Section("Server") {
                TextField("Name", text: $name)
                TextField("host:port", text: $server)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.URL)
                TextField("Client ID", text: $clientId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                SecureField(existing == nil ? "Secret" : "Secret (leave empty to keep)", text: $secret)
            }

            Section("Obfuscation") {
                Picker("Provider", selection: $obfsProvider) {
                    ForEach(ObfsProviders.shared.all, id: \.self) { Text(obfsLabel($0)).tag($0) }
                }
                .onChange(of: obfsProvider) { provider in
                    participation = provider == ObfsProviders.shared.NONE
                        ? "0" : String(ObfsProviders.shared.DEFAULT_PARTICIPATION)
                }
                LabeledContent("Participation") {
                    TextField("0–1", text: $participation)
                        .keyboardType(.decimalPad)
                        .multilineTextAlignment(.trailing)
                }
            }

            Section("Transport") {
                Picker("Performance", selection: $performanceMode) {
                    ForEach(PerformanceModes.shared.all, id: \.self) { Text(performanceLabel($0)).tag($0) }
                }
                Toggle("UDP", isOn: $udpEnabled)
            }

            if !problems.isEmpty {
                Section {
                    ForEach(problems, id: \.self) { Text($0.localizedText).foregroundStyle(.red) }
                }
            }
        }
        .navigationTitle(existing == nil ? "Add Server" : "Edit Server")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button { dismiss() } label: { Label("Cancel", systemImage: "xmark") }
                    .labelStyle(.iconOnly)
            }
            ToolbarItem(placement: .confirmationAction) {
                Button(action: save) { Label("Save", systemImage: "checkmark") }
                    .labelStyle(.iconOnly)
            }
        }
    }

    private func save() {
        let profile = ServerProfile(
            id: existing?.id ?? model.newProfileId(),
            name: name.trimmingCharacters(in: .whitespaces),
            server: ModelsKt.normalizeAscii(input: server),
            clientId: ModelsKt.normalizeAscii(input: clientId),
            obfsProvider: obfsProvider,
            obfsParticipation: Double(ModelsKt.normalizeAscii(input: participation)) ?? .nan,
            performanceMode: performanceMode,
            udpEnabled: udpEnabled
        )
        // Editing with an empty field keeps the stored secret.
        let newSecret: String? = (existing != nil && secret.isEmpty) ? nil : ModelsKt.normalizeAscii(input: secret)
        let result = model.save(profile, secret: newSecret)
        problems = result.problems
        if result.saved { dismiss() }
    }
}
