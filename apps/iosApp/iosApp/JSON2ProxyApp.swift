import SwiftUI

@main
struct JSON2ProxyApp: App {
    @StateObject private var model = AppModel()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(model)
                .task { await model.observe() }
                // Ask for network access right after installation (see NetworkAccess).
                .task { await model.networkAccess.prime() }
                .onChange(of: scenePhase) { phase in
                    if phase == .active { Task { await model.networkAccess.prime() } }
                }
                .networkAccessAlert(model.networkAccess)
                .onOpenURL { model.open($0) }
                .sheet(item: $model.pendingImport) { imported in
                    NavigationStack {
                        ServerEditorView(profile: nil, imported: imported.link)
                    }
                    .environmentObject(model)
                }
                .toastWindow()
        }
    }
}
