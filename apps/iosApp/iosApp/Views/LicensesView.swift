import Shared
import SwiftUI

/// Third-party components and data the app uses, grouped by kind, each with its license.
struct LicensesView: View {
    private let notices = OpenSourceNotices.shared.forPlatform(platform: .ios)

    var body: some View {
        TerminalPage {
            PathTitle(String(localized: "settings"), String(localized: "licenses"))
            Note(String(localized: "This app uses the following open-source components and data."))
            ForEach(NoticeCategory.allCases, id: \.self) { category in
                let group = notices.filter { $0.category == category }
                if !group.isEmpty {
                    TerminalSection(title: category.title) {
                        TerminalCard {
                            ForEach(Array(group.enumerated()), id: \.element.id) { index, notice in
                                if index > 0 { CardDivider() }
                                NoticeRow(notice: notice)
                            }
                        }
                    }
                }
            }
        }
        .navigationBarTitleDisplayMode(.inline)
    }
}

private struct NoticeRow: View {
    let notice: OpenSourceNotice
    @State private var expanded = false

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .firstTextBaseline) {
                Text(verbatim: notice.name).font(Typo.bodyMedium).foregroundStyle(Colors.textPrimary)
                Spacer(minLength: 8)
                Text(notice.license.title).font(Typo.label).foregroundStyle(Colors.accent)
            }
            Text(notice.id.purpose).font(Typo.small).foregroundStyle(Colors.textSecondary)
            Text(verbatim: notice.copyright).font(Typo.label).foregroundStyle(Colors.textSecondary)
            if let url = URL(string: notice.url) {
                Link(destination: url) {
                    Text(verbatim: notice.url).font(Typo.label).foregroundStyle(Colors.syntaxKey)
                }
                .frame(minHeight: 32)
            }
            if let text = notice.text {
                Button {
                    expanded.toggle()
                } label: {
                    Text(expanded ? "Hide license text" : "Show license text")
                        .font(Typo.small)
                        .foregroundStyle(Colors.textPrimary)
                        .frame(minHeight: 44)
                }
                .buttonStyle(.plain)
                if expanded {
                    Text(verbatim: text)
                        .font(Font.custom(Mono.regular, size: 10, relativeTo: .caption2))
                        .foregroundStyle(Colors.textSecondary)
                        .padding(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(Colors.canvas, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .textSelection(.enabled)
                }
            }
        }
        .padding(14)
    }
}

extension NoticeCategory {
    var title: String {
        switch self {
        case .library: String(localized: "Libraries")
        case .font: String(localized: "Fonts")
        case .icons: String(localized: "Icons")
        case .data: String(localized: "Data")
        }
    }
}

extension NoticeLicense {
    /// SPDX identifiers stay verbatim; public data has a localized name.
    var title: String {
        spdx ?? String(localized: "Public statistics")
    }
}

extension NoticeId {
    var purpose: String {
        switch self {
        case .goRuntime: String(localized: "Runtime of the networking core")
        case .goExtensions: String(localized: "DNS, system calls and mobile bindings")
        case .gvisor: String(localized: "TCP/IP stack inside the VPN tunnel")
        case .btree: String(localized: "Data structure used by gVisor")
        case .kotlinStdlib: String(localized: "Language runtime of the shared module")
        case .kotlinxCoroutines: String(localized: "Asynchronous work and state updates")
        case .kotlinxSerialization: String(localized: "JSON for settings and statistics")
        case .skie: String(localized: "Exposes the shared module to Swift")
        case .androidX: String(localized: "Android interface, lifecycle and camera")
        case .zxing: String(localized: "Generating and reading QR codes")
        case .jetBrainsMono: String(localized: "Monospaced typeface of the interface")
        case .materialIcons: String(localized: "Android interface icons")
        case .apnicStats: String(localized: "Source of the GeoIP CN address table")
        }
    }
}
