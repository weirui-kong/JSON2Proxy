import Shared
import SwiftUI

struct SettingsView: View {
    @EnvironmentObject private var model: AppModel
    @State private var draft = SettingsDraft()
    @State private var rulesError: String?
    @State private var loaded = false

    var body: some View {
        Form {
            Section {
                Picker("Mode", selection: $draft.mode) {
                    ForEach(RoutingMode.allCases, id: \.self) { Text($0.title).tag($0) }
                }
                if draft.mode == .custom {
                    TextEditor(text: $draft.rules)
                        .font(.system(.footnote, design: .monospaced))
                        .frame(minHeight: 160)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    if let rulesError {
                        Text("Invalid rules: \(rulesError)").foregroundStyle(.red).font(.footnote)
                    }
                }
            } header: {
                Text("Routing")
            } footer: {
                Text(draft.mode.footnote)
            }

            Section {
                TextField(draft.mode.defaultDoH, text: $draft.doh)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.URL)
                TextField(AppSettings.companion.DEFAULT_BOOTSTRAP, text: $draft.bootstrap)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
            } header: {
                Text("DNS")
            } footer: {
                Text("DoH resolves domains for routing. Bootstrap DNS only resolves the proxy server and the DoH host. Leave empty for the defaults.")
            }

            Section {
                Toggle("Kill switch", isOn: $draft.killSwitch)
                Toggle("Connect on demand", isOn: $draft.autoConnect)
                Toggle("Strict kill switch", isOn: $draft.strictKillSwitch)
            } header: {
                Text("Protection")
            } footer: {
                Text("The kill switch blocks traffic while the proxy is down. Strict mode routes all system traffic through the tunnel and can affect AirDrop and local network features. Changes apply on the next connection.")
            }
        }
        .navigationTitle("Settings")
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button(action: save) { Label("Save", systemImage: "checkmark") }
                    .labelStyle(.iconOnly)
            }
        }
        .onAppear {
            guard !loaded else { return }
            draft = SettingsDraft(model.settings)
            loaded = true
        }
    }

    private func save() {
        if draft.mode == .custom {
            rulesError = model.validateRules(draft.rules)
            guard rulesError == nil else { return }
        }
        model.update(draft.settings)
        ToastCenter.shared.show(String(localized: "Settings saved"))
    }
}

/// Editable copy of AppSettings; Kotlin data classes are immutable in Swift.
struct SettingsDraft {
    var mode: RoutingMode = .global
    var rules = ""
    var doh = ""
    var bootstrap = ""
    var killSwitch = true
    var autoConnect = false
    var strictKillSwitch = false

    init() {}

    init(_ settings: AppSettings) {
        mode = settings.routingMode
        rules = settings.customRules
        doh = settings.dohUrl
        bootstrap = settings.bootstrap
        killSwitch = settings.killSwitch
        autoConnect = settings.autoConnect
        strictKillSwitch = settings.strictKillSwitch
    }

    var settings: AppSettings {
        AppSettings(
            routingMode: mode,
            customRules: rules,
            dohUrl: ModelsKt.normalizeAscii(input: doh),
            bootstrap: ModelsKt.normalizeAscii(input: bootstrap),
            killSwitch: killSwitch,
            autoConnect: autoConnect,
            strictKillSwitch: strictKillSwitch
        )
    }
}

extension RoutingMode {
    var title: String {
        switch self {
        case .global: String(localized: "Global proxy")
        case .geoIpCn: String(localized: "GeoIP CN direct")
        case .custom: String(localized: "Imported rules")
        }
    }

    var footnote: String {
        switch self {
        case .global: String(localized: "All traffic goes through the proxy. Domains are sent to the server without resolving them on the device.")
        case .geoIpCn: String(localized: "Addresses in China connect directly; everything else goes through the proxy.")
        case .custom: String(localized: "One rule per line, ending with FINAL. Supported: DOMAIN, DOMAIN-SUFFIX, DOMAIN-KEYWORD, IP-CIDR, IP-CIDR6, GEOIP.")
        }
    }
}
