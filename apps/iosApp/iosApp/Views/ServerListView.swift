import Shared
import SwiftUI

struct ServerListView: View {
    @EnvironmentObject private var model: AppModel
    @State private var editing: EditorTarget?
    @State private var sharing: SharedServer?

    var body: some View {
        // A List keeps the swipe actions; its rows are restyled as terminal cards.
        List {
            Section {
                ForEach(model.profiles, id: \.id) { profile in
                    Button {
                        model.select(profile.id)
                    } label: {
                        ServerRow(profile: profile, isSelected: profile.id == model.selectedId, ping: model.pings[profile.server])
                    }
                    .buttonStyle(.plain)
                    .listRowInsets(EdgeInsets())
                    .listRowBackground(profile.id == model.selectedId ? Colors.surfaceSelected : Colors.surface)
                    .listRowSeparatorTint(Colors.border)
                    .swipeActions {
                        // Icons only; the names are read by VoiceOver.
                        // The app-wide accent tint would otherwise turn the destructive action green.
                        Button(role: .destructive) {
                            model.delete(profile.id)
                        } label: {
                            Image(systemName: "trash")
                        }
                        .tint(Colors.danger)
                        .accessibilityLabel(Text("Delete"))
                        Button {
                            editing = EditorTarget(profile: profile)
                        } label: {
                            Image(systemName: "pencil")
                        }
                        .tint(Colors.syntaxKey)
                        .accessibilityLabel(Text("Edit"))
                        Button {
                            share(profile)
                        } label: {
                            Image(systemName: "qrcode")
                        }
                        .tint(Colors.control)
                        .accessibilityLabel(Text("Show QR Code"))
                    }
                    .contextMenu {
                        Button("Edit") { editing = EditorTarget(profile: profile) }
                        Button("Test latency") { model.ping(profile.id) }
                        Button("Show QR Code") { share(profile) }
                        Button("Delete", role: .destructive) { model.delete(profile.id) }
                    }
                }
            } header: {
                // In the header rather than a row, so the section's rounded clipping leaves it alone.
                VStack(alignment: .leading, spacing: 20) {
                    PathTitle(String(localized: "servers"))
                    actions
                    Summary(summary: ServerSummary.companion.of(profiles: model.profiles, pings: model.pings))
                }
                .textCase(nil)
                .padding(.bottom, 12)
                .listRowInsets(EdgeInsets())
            } footer: {
                VStack(alignment: .leading, spacing: 2) {
                    if model.profiles.isEmpty {
                        Note(String(localized: "No servers yet"))
                    } else {
                        Note(Glyph.selected + " " + String(localized: "Selected · tap a server to switch"))
                        Note(String(localized: "Latency is the TCP connect time"))
                    }
                }
                .padding(.top, 8)
                .listRowInsets(EdgeInsets())
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
        .background(Colors.canvas.ignoresSafeArea())
        .environment(\.defaultMinListRowHeight, 52)
        .navigationBarTitleDisplayMode(.inline)
        .sheet(item: $editing) { target in
            NavigationStack {
                ServerEditorView(profile: target.profile)
            }
        }
        .sheet(item: $sharing) { shared in
            ServerQRCodeView(profile: shared.profile, link: shared.link)
        }
    }

    private var actions: some View {
        HStack(spacing: 8) {
            Button {
                model.presentScanner()
            } label: {
                Text("Scan").frame(maxWidth: .infinity)
            }
            .buttonStyle(SecondaryButtonStyle())
            Button(action: model.pingAll) {
                Text("Test all").frame(maxWidth: .infinity)
            }
            .buttonStyle(SecondaryButtonStyle())
            .disabled(model.profiles.isEmpty)
            Button {
                editing = EditorTarget(profile: nil)
            } label: {
                Text(verbatim: Glyph.add(String(localized: "Add"))).frame(maxWidth: .infinity)
            }
            .buttonStyle(PrimaryButtonStyle())
        }
    }

    private func share(_ profile: ServerProfile) {
        guard let link = model.link(for: profile) else {
            ToastCenter.shared.show(FailureReason.missingSecret.localizedText(detail: profile.name, isToast: true), duration: .long)
            return
        }
        sharing = SharedServer(profile: profile, link: link)
    }
}

/// Total servers, the best latency and how many failed; `—` until there are results.
private struct Summary: View {
    let summary: ServerSummary

    var body: some View {
        TerminalCard {
            HStack(spacing: 0) {
                cell(String(localized: "Total"), "\(summary.total)", Colors.textPrimary)
                Rectangle().fill(Colors.border).frame(width: 1)
                cell(
                    String(localized: "Best"),
                    summary.bestMillis.map { $0.int64Value < 1 ? String(localized: "<1 ms") : String(localized: "\($0.int64Value) ms") } ?? Glyph.none,
                    summary.bestMillis != nil ? Colors.accent : Colors.textSecondary
                )
                Rectangle().fill(Colors.border).frame(width: 1)
                cell(
                    String(localized: "Unreachable"),
                    summary.failed.map { "\($0.intValue)" } ?? Glyph.none,
                    (summary.failed?.intValue ?? 0) > 0 ? Colors.danger : Colors.textSecondary
                )
            }
            .fixedSize(horizontal: false, vertical: true)
        }
    }

    private func cell(_ label: String, _ value: String, _ color: Color) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label).font(Typo.caption).foregroundStyle(Colors.textSecondary).lineLimit(1)
            Text(verbatim: value).font(Typo.stat).foregroundStyle(color).lineLimit(1)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .accessibilityElement(children: .combine)
    }
}

private struct ServerRow: View {
    let profile: ServerProfile
    let isSelected: Bool
    let ping: PingState?

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            GlyphText(text: isSelected ? Glyph.selected : " ", color: Colors.accent, font: Typo.bodyBold)
                .frame(width: 14, alignment: .leading)
            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 8) {
                    Text(verbatim: profile.name).font(Typo.bodyMedium).foregroundStyle(Colors.textPrimary).lineLimit(1)
                    Spacer(minLength: 0)
                    if let ping {
                        Text(ping.label).font(Typo.small).foregroundStyle(ping.grade.color).lineLimit(1)
                    }
                }
                Text(verbatim: profile.server).font(Typo.label).foregroundStyle(Colors.textSecondary).lineLimit(1)
                if let ping { LatencyBar(ping: ping) }
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .contentShape(Rectangle())
        .accessibilityElement(children: .combine)
        .accessibilityAddTraits(isSelected ? [.isSelected] : [])
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
