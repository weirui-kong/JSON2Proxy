import Shared
import SwiftUI
import TerminalUI

struct SettingsView: View {
    @EnvironmentObject private var model: AppModel
    @Environment(\.dismiss) private var dismiss
    /// Closes the flip presentation (see SettingsPresenter); nil when pushed in a navigation stack.
    var onClose: (() -> Void)?
    @State private var draft = SettingsDraft()
    @State private var rulesError: String?
    @State private var loaded = false

    var body: some View {
        TerminalPage {
            TerminalPathTitle(String(localized: "settings"))

            TerminalSection(String(localized: "Routing")) {
                TerminalChoiceList(
                    selection: $draft.mode,
                    options: RoutingMode.allCases.map { TerminalOption($0, title: $0.title, detail: $0.wire) }
                )
                if draft.mode == .custom {
                    TerminalTextField(
                        String(localized: "Rules"),
                        text: $draft.rules,
                        error: rulesError.map { String(localized: "Invalid rules: \($0)") },
                        lines: Self.rulesLines
                    )
                }
                TerminalNote(draft.mode.footnote)
            }

            TerminalSection(String(localized: "DNS")) {
                TerminalTextField(String(localized: "DoH URL"), text: $draft.doh, placeholder: draft.mode.defaultDoH)
                    .keyboardType(.URL)
                TerminalTextField(String(localized: "Bootstrap DNS"), text: $draft.bootstrap, placeholder: AppSettings.companion.DEFAULT_BOOTSTRAP)
                TerminalNote(String(localized: "DoH resolves domains for routing. Bootstrap DNS only resolves the proxy server and the DoH host. Leave empty for the defaults."))
            }

            TerminalSection(String(localized: "Protection")) {
                TerminalSwitchRow(String(localized: "Kill switch"), isOn: $draft.killSwitch)
                TerminalSwitchRow(String(localized: "Connect on demand"), isOn: $draft.autoConnect)
                TerminalSwitchRow(String(localized: "Strict kill switch"), isOn: $draft.strictKillSwitch)
                TerminalNote(String(localized: "The kill switch blocks traffic while the proxy is down. Strict mode routes all system traffic through the tunnel and can affect AirDrop and local network features. Changes apply on the next connection."))
            }

            TerminalSection(String(localized: "Traffic units")) {
                TerminalChoiceList(
                    selection: $draft.trafficUnit,
                    options: TrafficUnit.allCases.map { TerminalOption($0, title: $0.title) }
                )
            }
        } footer: {
            // A small, quiet entry at the very end of the page.
            NavigationLink {
                LicensesView()
            } label: {
                Text("Open-source licenses").foregroundColor(TerminalColors.textSecondary)
            }
            .buttonStyle(TerminalButtonStyle(.secondary))
            .frame(maxWidth: .infinity)
            // Keeps the gap above it once the page scrolls.
            .padding(.top, TerminalSpacing.space20)
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if let onClose {
                ToolbarItem(placement: .cancellationAction) {
                    Button(action: onClose) { Label { Text("Close") } icon: { TerminalIcon.close.image } }
                        .labelStyle(.iconOnly)
                }
            }
            ToolbarItem(placement: .confirmationAction) {
                Button("Save", action: save)
                    .font(TerminalType.bodyBold)
                    .foregroundStyle(TerminalColors.accentText)
            }
        }
        .onAppear {
            guard !loaded else { return }
            draft = SettingsDraft(model.settings)
            loaded = true
        }
    }

    /// Height of the custom rules box, in lines.
    private static let rulesLines = 7

    private func close() {
        if let onClose { onClose() } else { dismiss() }
    }

    private func save() {
        if draft.mode == .custom {
            rulesError = model.validateRules(draft.rules)
            guard rulesError == nil else { return }
        }
        model.update(draft.settings)
        ToastCenter.shared.show(String(localized: "Settings saved"))
        close()
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
        case .bytes: String(localized: "Bytes")
        case .bits: String(localized: "Bits")
        }
    }
}
