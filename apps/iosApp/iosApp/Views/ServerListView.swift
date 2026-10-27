import Shared
import SwiftUI
import TerminalUI

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
                    .listRowBackground(profile.id == model.selectedId ? TerminalColors.surfaceSelected : TerminalColors.surface)
                    .listRowSeparatorTint(TerminalColors.border)
                    .swipeActions {
                        // Icons only; the names are read by VoiceOver.
                        // The app-wide accent tint would otherwise turn the destructive action green.
                        Button(role: .destructive) {
                            model.delete(profile.id)
                        } label: {
                            TerminalIcon.delete.image
                        }
                        .tint(TerminalColors.danger)
                        .accessibilityLabel(Text("Delete"))
                        Button {
                            editing = EditorTarget(profile: profile)
                        } label: {
                            TerminalIcon.edit.image
                        }
                        .tint(TerminalColors.syntaxKey)
                        .accessibilityLabel(Text("Edit"))
                        Button {
                            share(profile)
                        } label: {
                            TerminalIcon.qrCode.image
                        }
                        .tint(TerminalColors.control)
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
                    TerminalPathTitle(String(localized: "servers"))
                    actions
                    Summary(summary: ServerSummary.companion.of(profiles: model.profiles, pings: model.pings))
                }
                .textCase(nil)
                .padding(.bottom, 12)
                .listRowInsets(EdgeInsets())
            } footer: {
                VStack(alignment: .leading, spacing: 2) {
                    if model.profiles.isEmpty {
                        TerminalNote(String(localized: "No servers yet"))
                    } else {
                        TerminalNote(TerminalGlyph.selected + " " + String(localized: "Selected · tap a server to switch"))
                        TerminalNote(String(localized: "Latency is the TCP connect time"))
                    }
                }
                .padding(.top, 8)
                .listRowInsets(EdgeInsets())
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
        .background(TerminalColors.canvas.ignoresSafeArea())
        .environment(\.defaultMinListRowHeight, TerminalSize.row)
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
        HStack(spacing: TerminalSpacing.space8) {
            TerminalButton(String(localized: "Scan"), block: true) {
                model.presentScanner()
            }
            TerminalButton(String(localized: "Test all"), block: true, action: model.pingAll)
                .disabled(model.profiles.isEmpty)
            TerminalButton(String(localized: "Add"), variant: .primary, block: true, decorate: .add) {
                editing = EditorTarget(profile: nil)
            }
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
                cell(String(localized: "Total"), "\(summary.total)", TerminalColors.textPrimary)
                TerminalCardDivider(.vertical)
                cell(
                    String(localized: "Best"),
                    summary.bestMillis.map { $0.int64Value < 1 ? String(localized: "<1 ms") : String(localized: "\($0.int64Value) ms") } ?? TerminalGlyph.none,
                    summary.bestMillis != nil ? TerminalColors.accentText : TerminalColors.textSecondary
                )
                TerminalCardDivider(.vertical)
                cell(
                    String(localized: "Unreachable"),
                    summary.failed.map { "\($0.intValue)" } ?? TerminalGlyph.none,
                    (summary.failed?.intValue ?? 0) > 0 ? TerminalColors.danger : TerminalColors.textSecondary
                )
            }
            .fixedSize(horizontal: false, vertical: true)
        }
    }

    private func cell(_ label: String, _ value: String, _ color: Color) -> some View {
        VStack(alignment: .leading, spacing: TerminalSpacing.space2) {
            Text(label).font(TerminalType.caption).foregroundStyle(TerminalColors.textSecondary).lineLimit(1)
            Text(verbatim: value).font(TerminalType.stat).foregroundStyle(color).lineLimit(1)
        }
        .padding(.horizontal, TerminalSpacing.space12)
        .padding(.vertical, TerminalSpacing.space10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .accessibilityElement(children: .combine)
    }
}

private struct ServerRow: View {
    let profile: ServerProfile
    let isSelected: Bool
    let ping: PingState?

    var body: some View {
        HStack(alignment: .top, spacing: TerminalSpacing.space10) {
            TerminalGlyphText(isSelected ? TerminalGlyph.selected : " ", color: TerminalColors.accentText, font: TerminalType.bodyBold)
                .frame(width: TerminalSpacing.space14, alignment: .leading)
            VStack(alignment: .leading, spacing: TerminalSpacing.space4) {
                HStack(spacing: TerminalSpacing.space8) {
                    Text(verbatim: profile.name).font(TerminalType.bodyMedium).foregroundStyle(TerminalColors.textPrimary).lineLimit(1)
                    Spacer(minLength: 0)
                    if let ping {
                        Text(ping.label).font(TerminalType.small).foregroundStyle(ping.grade.color).lineLimit(1)
                    }
                }
                Text(verbatim: profile.server).font(TerminalType.label).foregroundStyle(TerminalColors.textSecondary).lineLimit(1)
                if let ping {
                    // Decorative: the latency text above says the same.
                    TerminalLatencyBar(
                        ms: ping.millis,
                        failed: ping.grade == .failed,
                        goodBelowMs: Int(Latency.shared.GOOD_BELOW_MS),
                        cells: Int(Latency.shared.BAR_CELLS)
                    )
                    .accessibilityHidden(true)
                }
            }
        }
        .padding(.horizontal, TerminalSpacing.space14)
        .padding(.vertical, TerminalSpacing.space12)
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

extension LatencyGrade {
    /// Color of the latency text.
    var color: Color {
        switch self {
        case .testing: TerminalColors.textSecondary
        case .good: TerminalColors.accentText
        case .slow: TerminalColors.warning
        case .failed: TerminalColors.danger
        }
    }
}

extension PingState {
    var grade: LatencyGrade { Latency.shared.grade(ping: self) }

    /// The measured latency, nil while testing or after a failure.
    var millis: Int? {
        if case let .success(success) = onEnum(of: self) { return Int(success.millis) }
        return nil
    }
}
