import Shared
import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(model.state.label)
                    if let traffic = model.traffic {
                        TrafficSummary(traffic: traffic, unit: model.settings.trafficUnit)
                    }
                    if model.state.isActive {
                        Button("Disconnect", role: .destructive, action: model.disconnect)
                    } else {
                        Button("Connect", action: model.connect)
                            .disabled(model.selectedProfile == nil)
                    }
                }

                Section("Server") {
                    NavigationLink {
                        ServerListView()
                    } label: {
                        if let profile = model.selectedProfile {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(profile.name)
                                Text(profile.server)
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                            }
                        } else {
                            Text("Add a server")
                        }
                    }
                }

                Section {
                    NavigationLink("Settings") { SettingsView() }
                } footer: {
                    Text("Routing: \(model.settings.routingMode.title) · \(model.platform)")
                }
            }
            .navigationTitle("JSON2Proxy")
        }
    }
}

/// Overall upload and download speed, and the open TCP connections and UDP flows.
private struct TrafficSummary: View {
    let traffic: TrafficStats
    let unit: TrafficUnit

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 16) {
                Label {
                    Text(TrafficFormat.shared.rate(bytesPerSecond: traffic.uploadSpeed, unit: unit))
                } icon: {
                    Image(systemName: "arrow.up")
                }
                .accessibilityLabel(Text("Upload"))
                Label {
                    Text(TrafficFormat.shared.rate(bytesPerSecond: traffic.downloadSpeed, unit: unit))
                } icon: {
                    Image(systemName: "arrow.down")
                }
                .accessibilityLabel(Text("Download"))
            }
            .font(.subheadline.monospacedDigit())
            Text("TCP \(traffic.tcpConnections) · UDP \(traffic.udpConnections)")
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
    }
}
