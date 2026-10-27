import SwiftUI

struct ContentView: View {
    @StateObject private var model = ConnectionViewModel()

    var body: some View {
        NavigationStack {
            Form {
                Section("Server") {
                    TextField("host:port", text: $model.server)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                }

                Section {
                    Text(model.state.label)
                    if model.state.isActive {
                        Button("Disconnect", role: .destructive, action: model.disconnect)
                    } else {
                        Button("Connect", action: model.connect)
                    }
                }

                Section {
                    Text(model.platform)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle("JSON2Proxy")
        }
        .task { await model.observeState() }
        .task { await model.observeNotices() }
    }
}

#Preview {
    ContentView()
}
