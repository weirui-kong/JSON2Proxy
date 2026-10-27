import Shared
import SwiftUI

struct ServerListView: View {
    @EnvironmentObject private var model: AppModel
    @State private var editing: EditorTarget?
    @State private var sharing: SharedServer?

    var body: some View {
        List {
            ForEach(model.profiles, id: \.id) { profile in
                Button {
                    model.select(profile.id)
                } label: {
                    row(profile)
                }
                .buttonStyle(.plain)
                .swipeActions {
                    Button("Delete", role: .destructive) { model.delete(profile.id) }
                    Button("Edit") { editing = EditorTarget(profile: profile) }
                        .tint(.blue)
                    Button {
                        share(profile)
                    } label: {
                        Label("Show QR Code", systemImage: "qrcode")
                    }
                    .tint(.indigo)
                }
                .contextMenu {
                    Button("Edit") { editing = EditorTarget(profile: profile) }
                    Button("Test latency") { model.ping(profile.id) }
                    Button("Show QR Code") { share(profile) }
                    Button("Delete", role: .destructive) { model.delete(profile.id) }
                }
            }
        }
        .overlay {
            if model.profiles.isEmpty {
                Text("No servers yet")
                    .foregroundStyle(.secondary)
            }
        }
        .navigationTitle("Servers")
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: model.pingAll) {
                    Label("Test all", systemImage: "speedometer")
                }
                .labelStyle(.iconOnly)
                .disabled(model.profiles.isEmpty)
            }
            ToolbarItemGroup(placement: .navigationBarTrailing) {
                Button {
                    model.presentScanner()
                } label: {
                    Label("Scan", systemImage: "qrcode.viewfinder")
                }
                .labelStyle(.iconOnly)
                Button {
                    editing = EditorTarget(profile: nil)
                } label: {
                    Label("Add", systemImage: "plus")
                }
                .labelStyle(.iconOnly)
            }
        }
        .sheet(item: $editing) { target in
            NavigationStack {
                ServerEditorView(profile: target.profile)
            }
        }
        .sheet(item: $sharing) { shared in
            ServerQRCodeView(profile: shared.profile, link: shared.link)
        }
    }

    private func share(_ profile: ServerProfile) {
        guard let link = model.link(for: profile) else {
            ToastCenter.shared.show(FailureReason.missingSecret.localizedText(detail: profile.name, isToast: true), duration: .long)
            return
        }
        sharing = SharedServer(profile: profile, link: link)
    }

    private func row(_ profile: ServerProfile) -> some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(profile.name)
                    .foregroundStyle(.primary)
                Text(profile.server)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            if let ping = model.pings[profile.server] {
                Text(ping.label)
                    .font(.footnote)
                    .foregroundStyle(ping.isFailure ? .red : .secondary)
                    .lineLimit(1)
            }
            if profile.id == model.selectedId {
                Image(systemName: "checkmark")
                    .foregroundStyle(.tint)
            }
        }
        .contentShape(Rectangle())
    }
}

struct SharedServer: Identifiable {
    let id = UUID()
    let profile: ServerProfile
    let link: String
}

/// Sheet item: nil profile means "add".
struct EditorTarget: Identifiable {
    let id = UUID()
    let profile: ServerProfile?
}
