import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var model: AppModel

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(model.state.label)
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
