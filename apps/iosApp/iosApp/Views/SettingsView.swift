import Shared
import SwiftUI

struct SettingsView: View {
    @EnvironmentObject private var model: AppModel
    @Environment(\.dismiss) private var dismiss
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

            Section("Traffic units") {
                Picker("Traffic units", selection: $draft.trafficUnit) {
                    ForEach(TrafficUnit.allCases, id: \.self) { Text($0.title).tag($0) }
                }
                .pickerStyle(.inline)
                .labelsHidden()
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
        dismiss()
    }
}

/// Editable copy of AppSettings; Kotlin data classes are immutable in Swift.
struct SettingsDraft {
    struct DNS {
        var doh = ""
        var bootstrap = ""
    }

    var mode: RoutingMode = .global
    var rules = ""
    /// Each routing mode keeps its own DNS options; switching modes only changes which ones are shown.
    var dns: [RoutingMode: DNS] = [:]
    var killSwitch = true
    var autoConnect = false
    var strictKillSwitch = false
    var trafficUnit: TrafficUnit = .bytes

    init() {}

    init(_ settings: AppSettings) {
        mode = settings.routingMode
        rules = settings.customRules
        for mode in RoutingMode.allCases {
            let options = settings.options(mode: mode)
            dns[mode] = DNS(doh: options.dohUrl, bootstrap: options.bootstrap)
        }
        killSwitch = settings.killSwitch
        autoConnect = settings.autoConnect
        strictKillSwitch = settings.strictKillSwitch
        trafficUnit = settings.trafficUnit
    }

    /// DoH of the selected mode.
    var doh: String {
        get { dns[mode]?.doh ?? "" }
        set { dns[mode, default: DNS()].doh = newValue }
    }

    /// Bootstrap DNS of the selected mode.
    var bootstrap: String {
        get { dns[mode]?.bootstrap ?? "" }
        set { dns[mode, default: DNS()].bootstrap = newValue }
    }

    var settings: AppSettings {
        AppSettings(
            routingMode: mode,
            customRules: rules,
            modeOptions: dns.reduce(into: [String: ModeOptions]()) { result, entry in
                result[entry.key.wire] = ModeOptions(
                    dohUrl: ModelsKt.normalizeAscii(input: entry.value.doh),
                    bootstrap: ModelsKt.normalizeAscii(input: entry.value.bootstrap)
                )
            },
            killSwitch: killSwitch,
            autoConnect: autoConnect,
            strictKillSwitch: strictKillSwitch,
            trafficUnit: trafficUnit
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

extension TrafficUnit {
    var title: String {
        switch self {
        case .bytes: String(localized: "Bytes (KB/s, MB/s, GB/s)")
        case .bits: String(localized: "Bits (Kb/s, Mb/s, Gb/s)")
        }
    }
}
