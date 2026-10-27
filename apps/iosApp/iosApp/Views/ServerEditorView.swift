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

    /// Parts of the form that a validation problem points at, in form order.
    private enum Field: Int, Comparable {
        case name, server, clientId, secret, obfs, participation, transport

        static func < (lhs: Field, rhs: Field) -> Bool { lhs.rawValue < rhs.rawValue }
    }

    private func field(of problem: ProfileProblem) -> Field {
        switch problem {
        case .nameRequired: .name
        case .invalidServer: .server
        case .clientIdRequired: .clientId
        case .secretNotBase64Url, .secretTooShort: .secret
        case .unknownObfsProvider: .obfs
        case .invalidParticipation: .participation
        case .unknownPerformanceMode: .transport
        }
    }

    private func error(_ field: Field) -> String? {
        problems.first { self.field(of: $0) == field }?.localizedText
    }

    var body: some View {
        ScrollViewReader { reader in
            TerminalPage {
                PathTitle(String(localized: "servers"), existing == nil ? String(localized: "new") : String(localized: "edit"))
                if isImport {
                    Banner(kind: .info, text: String(localized: "Review the imported server, then save."))
                }

                VStack(alignment: .leading, spacing: 14) {
                    TerminalField(label: String(localized: "Name"), error: error(.name)) {
                        TextField(text: $name) { Text("Name") }
                    }
                    .id(Field.name)
                    TerminalField(label: String(localized: "Server (host:port)"), error: error(.server)) {
                        TextField(text: $server) { Text("Server (host:port)") }
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .keyboardType(.URL)
                    }
                    .id(Field.server)
                    TerminalField(label: String(localized: "Client ID"), error: error(.clientId)) {
                        TextField(text: $clientId) { Text("Client ID") }
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                    }
                    .id(Field.clientId)
                    let secretLabel = existing == nil ? String(localized: "Secret") : String(localized: "Secret (leave empty to keep)")
                    TerminalField(label: secretLabel, error: error(.secret)) {
                        // The label sits above the field; an empty prompt keeps it from repeating inside.
                        SecureField(text: $secret, prompt: Text(verbatim: "")) { Text(secretLabel) }
                    }
                    .id(Field.secret)
                }

                TerminalSection(title: String(localized: "Obfuscation")) {
                    SegmentedChoice(options: ObfsProviders.shared.all, selection: $obfsProvider, label: obfsLabel)
                        .onChange(of: obfsProvider) { provider in
                            participation = provider == ObfsProviders.shared.NONE
                                ? "0" : String(ObfsProviders.shared.DEFAULT_PARTICIPATION)
                        }
                    if let message = error(.obfs) { FieldError(text: message) }
                    // Participation only matters once a provider is chosen.
                    if obfsProvider != ObfsProviders.shared.NONE || error(.participation) != nil {
                        TerminalField(label: String(localized: "Participation (0–1)"), error: error(.participation)) {
                            TextField(text: $participation) { Text("Participation (0–1)") }
                                .keyboardType(.decimalPad)
                        }
                        .id(Field.participation)
                    }
                }
                .id(Field.obfs)

                TerminalSection(title: String(localized: "Transport")) {
                    ChoiceList(options: PerformanceModes.shared.all, selection: $performanceMode, title: performanceLabel, detail: { $0 })
                    if let message = error(.transport) { FieldError(text: message) }
                    SwitchRow(label: String(localized: "UDP"), isOn: $udpEnabled)
                }
                .id(Field.transport)
            }
            .onChange(of: problems) { problems in
                guard let first = problems.map(field(of:)).min() else { return }
                withAnimation { reader.scrollTo(first, anchor: .top) }
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button { dismiss() } label: { Label("Cancel", systemImage: "xmark") }
                    .labelStyle(.iconOnly)
            }
            ToolbarItem(placement: .confirmationAction) {
                Button("Save", action: save)
                    .font(Typo.bodyBold)
                    .foregroundStyle(Colors.accent)
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
