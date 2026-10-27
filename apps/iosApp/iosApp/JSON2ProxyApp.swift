import SwiftUI

@main
struct JSON2ProxyApp: App {
    @StateObject private var model = AppModel()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(model)
                .task { await model.observe() }
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
