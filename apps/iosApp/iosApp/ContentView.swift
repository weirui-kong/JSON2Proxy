import Shared
import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        NavigationStack {
            TerminalPage {
                HStack {
                    PromptTitle()
                    Spacer()
                    Button {
                        SettingsPresenter.present(model: model)
                    } label: {
                        SquareIcon(systemName: "slider.horizontal.3")
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(Text("Settings"))
                }
                SessionCard(
                    state: model.state,
                    server: model.selectedProfile?.name,
                    routing: model.settings.routingMode,
                    traffic: model.traffic,
                    history: model.trafficHistory,
                    unit: model.settings.trafficUnit
                )
                if case .failed = onEnum(of: model.state) {
                    Banner(kind: .danger, text: model.state.label)
                }
                connectButton
            } bottom: {
                TerminalCard {
                    NavigationLink {
                        ServerListView()
                    } label: {
                        NavRow(
                            label: String(localized: "Server"),
                            title: model.selectedProfile?.name ?? String(localized: "Add a server"),
                            subtitle: model.selectedProfile?.server,
                            trailing: model.selectedProfile.flatMap { model.pings[$0.server] }.map { ($0.label, $0.grade.color) }
                        )
                    }
                    CardDivider()
                    Button {
                        SettingsPresenter.present(model: model)
                    } label: {
                        NavRow(label: String(localized: "Routing"), title: model.settings.routingMode.title)
                    }
                    CardDivider()
                    Button {
                        SettingsPresenter.present(model: model)
                    } label: {
                        NavRow(
                            label: String(localized: "Kill switch"),
                            title: nil,
                            trailing: model.settings.killSwitch
                                ? (String(localized: "On"), Colors.accent)
                                : (String(localized: "Off"), Colors.textSecondary)
                        )
                    }
                }
                .buttonStyle(.plain)
            }
            .toolbar(.hidden, for: .navigationBar)
        }
    }

    private var isDisconnecting: Bool {
        if case .disconnecting = onEnum(of: model.state) { return true }
        return false
    }

    @ViewBuilder private var connectButton: some View {
        if model.state.isActive {
            let label = String(localized: "Disconnect")
            Button(action: model.disconnect) {
                Text(verbatim: Glyph.disconnect(label)).frame(maxWidth: .infinity)
            }
            .buttonStyle(SecondaryButtonStyle(color: Colors.danger, borderColor: Colors.danger, background: .clear, height: 52))
            .disabled(isDisconnecting)
            .accessibilityLabel(Text(label))
        } else {
            let label = String(localized: "Connect")
            Button(action: model.connect) {
                Text(verbatim: Glyph.connect(label)).frame(maxWidth: .infinity)
            }
            .buttonStyle(PrimaryButtonStyle(height: 52))
            .disabled(model.selectedProfile == nil)
            .accessibilityLabel(Text(label))
        }
    }
}

/// `session.json`: the connection as a small JSON object, then live traffic.
private struct SessionCard: View {
    let state: ConnectionState
    let server: String?
    let routing: RoutingMode
    let traffic: TrafficStats?
    let history: [TrafficPoint]
    let unit: TrafficUnit

    private var isLive: Bool {
        if case .connected = onEnum(of: state) { return true }
        return false
    }

    private var stateValue: (String, Color) {
        switch onEnum(of: state) {
        case .connected: (Glyph.stateConnected, Colors.accent)
        case .connecting: (Glyph.stateConnecting, Colors.warning)
        case .disconnecting: (Glyph.stateDisconnecting, Colors.warning)
        case .failed: (Glyph.stateFailed, Colors.danger)
        case .disconnected: (Glyph.stateDisconnected, Colors.textSecondary)
        }
    }

    var body: some View {
        TerminalCard {
            HStack {
                Text(verbatim: Glyph.sessionFile)
                Spacer()
                let dot = isLive ? Colors.accent : Colors.textSecondary
                Circle().fill(dot).frame(width: 7, height: 7)
                Text(verbatim: isLive ? Glyph.live : Glyph.idle).foregroundStyle(dot)
            }
            .font(Typo.label)
            .foregroundStyle(Colors.textSecondary)
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .accessibilityHidden(true)
            CardDivider()
            json
                .font(Typo.code)
                .lineSpacing(6)
                .padding(.horizontal, 14)
                .padding(.vertical, 12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityElement()
                .accessibilityLabel(Text("\(state.label), routing: \(routing.title)"))
            CardDivider()
            if isLive, let traffic {
                TrafficPanel(traffic: traffic, history: history, unit: unit)
            } else {
                Note(String(localized: "Live traffic appears once connected")).padding(14)
            }
        }
    }

    private var json: Text {
        let punctuation = Colors.textSecondary
        func key(_ name: String) -> Text {
            Text(verbatim: "  ") + Text(verbatim: "\"\(name)\"").foregroundColor(Colors.syntaxKey) + Text(verbatim: ": ").foregroundColor(punctuation)
        }
        let (value, color) = stateValue
        let serverValue = server.map { Text(verbatim: "\"\($0)\"").foregroundColor(Colors.syntaxString) }
            ?? Text(verbatim: Glyph.null).foregroundColor(Colors.syntaxNumber)
        return Text(verbatim: "{\n").foregroundColor(punctuation)
            + key(Glyph.keyState) + Text(verbatim: "\"\(value)\"").foregroundColor(color) + Text(verbatim: ",\n").foregroundColor(punctuation)
            + key(Glyph.keyServer) + serverValue + Text(verbatim: ",\n").foregroundColor(punctuation)
            + key(Glyph.keyRouting) + Text(verbatim: "\"\(routing.wire)\"").foregroundColor(Colors.syntaxString)
            + Text(verbatim: "\n}").foregroundColor(punctuation)
    }
}

private struct TrafficPanel: View {
    let traffic: TrafficStats
    let history: [TrafficPoint]
    let unit: TrafficUnit

    var body: some View {
        let download = TrafficFormat.shared.rate(bytesPerSecond: traffic.downloadSpeed, unit: unit)
        let upload = TrafficFormat.shared.rate(bytesPerSecond: traffic.uploadSpeed, unit: unit)
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Note(String(localized: "traffic · 60 s"))
                Spacer()
                Legend(color: Colors.syntaxKey, label: String(localized: "Download"))
                Legend(color: Colors.syntaxNumber, label: String(localized: "Upload"))
            }
            TrafficChart(history: history, description: String(localized: "Download \(download), upload \(upload)"))
            Grid(horizontalSpacing: 8, verticalSpacing: 8) {
                GridRow {
                    StatCell(glyph: Glyph.download, label: String(localized: "Download"), value: download, color: Colors.syntaxKey)
                    StatCell(glyph: Glyph.upload, label: String(localized: "Upload"), value: upload, color: Colors.syntaxNumber)
                }
                GridRow {
                    StatCell(glyph: nil, label: Glyph.tcp, value: String(traffic.tcpConnections), color: Colors.textPrimary)
                    StatCell(glyph: nil, label: Glyph.udp, value: String(traffic.udpConnections), color: Colors.textPrimary)
                }
            }
        }
        .padding(.horizontal, 14)
        .padding(.top, 12)
        .padding(.bottom, 14)
    }
}

private struct Legend: View {
    let color: Color
    let label: String

    var body: some View {
        HStack(spacing: 4) {
            Rectangle().fill(color).frame(width: 10, height: 2)
            Text(label).font(Typo.caption).foregroundStyle(Colors.textSecondary)
        }
    }
}

/// Label over a value; a rate's unit is smaller and dimmed.
private struct StatCell: View {
    let glyph: String?
    let label: String
    let value: String
    let color: Color

    var body: some View {
        let (number, unit) = splitRate(value)
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 0) {
                if let glyph { GlyphText(text: glyph + " ", font: Typo.caption) }
                Text(label).font(Typo.caption).foregroundStyle(Colors.textSecondary)
            }
            (Text(verbatim: number).foregroundColor(color) + Text(verbatim: unit.isEmpty ? "" : " \(unit)").foregroundColor(Colors.textSecondary))
                .font(Typo.stat)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(Colors.border))
        .accessibilityElement(children: .combine)
    }
}

/// A row of the home card: a short label, a value, and an optional colored status.
private struct NavRow: View {
    let label: String
    let title: String?
    var subtitle: String?
    var trailing: (String, Color)?

    var body: some View {
        HStack(spacing: 12) {
            Text(label).font(Typo.small).foregroundStyle(Colors.textSecondary).frame(width: 96, alignment: .leading)
            VStack(alignment: .leading, spacing: 2) {
                if let title {
                    Text(title).font(Typo.bodyMedium).foregroundStyle(Colors.textPrimary).lineLimit(1)
                }
                if let subtitle {
                    Text(verbatim: subtitle).font(Typo.label).foregroundStyle(Colors.textSecondary).lineLimit(1)
                }
            }
            Spacer(minLength: 0)
            if let trailing {
                Text(trailing.0).font(Typo.small).foregroundStyle(trailing.1).lineLimit(1)
            }
            GlyphText(text: Glyph.next)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .frame(minHeight: 52)
        .contentShape(Rectangle())
    }
}

/// 44×44 outlined icon, used as the label of buttons and links.
struct SquareIcon: View {
    let systemName: String
    var tint = Colors.textPrimary

    var body: some View {
        Image(systemName: systemName)
            .font(.system(size: 17, weight: .medium))
            .foregroundStyle(tint)
            .frame(width: 44, height: 44)
            .background(Colors.surface, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(Colors.border))
    }
}
