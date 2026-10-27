import Shared
import SwiftUI
import UIKit

/// The shared color card (`Palette` in the shared module); screens take every color from here.
enum Colors {
    static let canvas = Color(argb: Palette.shared.canvas)
    static let surface = Color(argb: Palette.shared.surface)
    static let surfaceSelected = Color(argb: Palette.shared.surfaceSelected)
    static let border = Color(argb: Palette.shared.border)
    static let control = Color(argb: Palette.shared.control)
    static let gridline = Color(argb: Palette.shared.gridline)
    static let textPrimary = Color(argb: Palette.shared.textPrimary)
    static let textSecondary = Color(argb: Palette.shared.textSecondary)
    static let accent = Color(argb: Palette.shared.accent)
    static let onAccent = Color(argb: Palette.shared.onAccent)
    static let danger = Color(argb: Palette.shared.danger)
    static let warning = Color(argb: Palette.shared.warning)
    static let syntaxKey = Color(argb: Palette.shared.syntaxKey)
    static let syntaxString = Color(argb: Palette.shared.syntaxString)
    static let syntaxNumber = Color(argb: Palette.shared.syntaxNumber)
    static let infoBackground = Color(argb: Palette.shared.infoBackground)
    static let infoBorder = Color(argb: Palette.shared.infoBorder)
    static let infoText = Color(argb: Palette.shared.infoText)
    static let warningBackground = Color(argb: Palette.shared.warningBackground)
    static let warningBorder = Color(argb: Palette.shared.warningBorder)
    static let warningText = Color(argb: Palette.shared.warningText)
    static let dangerBackground = Color(argb: Palette.shared.dangerBackground)
    static let dangerText = Color(argb: Palette.shared.dangerText)
}

extension UIColor {
    convenience init(argb: Int64) {
        self.init(
            red: CGFloat((argb >> 16) & 0xFF) / 255,
            green: CGFloat((argb >> 8) & 0xFF) / 255,
            blue: CGFloat(argb & 0xFF) / 255,
            alpha: CGFloat((argb >> 24) & 0xFF) / 255
        )
    }
}

extension Color {
    init(argb: Int64) {
        self.init(uiColor: UIColor(argb: argb))
    }
}

/// JetBrains Mono NL (SIL OFL 1.1), scaled with Dynamic Type. Chinese falls back to the system font.
enum Mono {
    static let regular = "JetBrainsMonoNL-Regular"
    static let medium = "JetBrainsMonoNL-Medium"
    static let bold = "JetBrainsMonoNL-Bold"

    static func uiFont(_ name: String, size: CGFloat, style: UIFont.TextStyle) -> UIFont {
        let font = UIFont(name: name, size: size) ?? .monospacedSystemFont(ofSize: size, weight: .regular)
        return UIFontMetrics(forTextStyle: style).scaledFont(for: font)
    }
}

/// Type scale of the terminal UI spec.
enum Typo {
    static let title = Font.custom(Mono.bold, size: 20, relativeTo: .title3)
    static let section = Font.custom(Mono.bold, size: 15, relativeTo: .headline)
    static let body = Font.custom(Mono.regular, size: 15, relativeTo: .body)
    static let bodyMedium = Font.custom(Mono.medium, size: 15, relativeTo: .body)
    static let bodyBold = Font.custom(Mono.bold, size: 15, relativeTo: .body)
    static let code = Font.custom(Mono.regular, size: 14, relativeTo: .callout)
    static let stat = Font.custom(Mono.bold, size: 17, relativeTo: .headline)
    static let small = Font.custom(Mono.regular, size: 13, relativeTo: .subheadline)
    static let smallBold = Font.custom(Mono.bold, size: 13, relativeTo: .subheadline)
    static let label = Font.custom(Mono.regular, size: 12, relativeTo: .footnote)
    static let caption = Font.custom(Mono.regular, size: 11, relativeTo: .caption)
}

/// Technical notation of the terminal style: the same in every language, never translated.
enum Glyph {
    static let prompt = "$"
    static let command = "json2proxy"
    static let homeDir = "~/"
    static let pathSeparator = "/"
    static let section = "#"
    static let comment = "//"
    static let selected = ">"
    static let radioOn = "(•)"
    static let radioOff = "( )"
    static let error = "✕"
    static let external = "↗"
    static let next = "→"
    static let download = "↓"
    static let upload = "↑"
    static let none = "—"
    static let barFilled = "█"
    static let barEmpty = "░"
    static let barFailed = "·"
    static let sessionFile = "session.json"
    static let qrFile = "server.qr"
    static let linkScheme = "json2proxy://"
    static let live = "live"
    static let idle = "idle"
    static let keyState = "state"
    static let keyServer = "server"
    static let keyRouting = "routing"
    static let null = "null"
    static let stateConnected = "connected"
    static let stateConnecting = "connecting"
    static let stateDisconnecting = "disconnecting"
    static let stateDisconnected = "disconnected"
    static let stateFailed = "failed"
    static let tcp = "TCP"
    static let udp = "UDP"
    static let memory = "mem"
    static let footerSeparator = " · "

    static func connect(_ label: String) -> String { "> \(label)_" }
    static func disconnect(_ label: String) -> String { "[ \(label) ]" }
    static func add(_ label: String) -> String { "+ \(label)" }
}

/// Dark navigation bars over the canvas, with the monospaced title font.
enum Appearance {
    static func apply() {
        let bar = UINavigationBarAppearance()
        bar.configureWithOpaqueBackground()
        bar.backgroundColor = UIColor(argb: Palette.shared.canvas)
        bar.shadowColor = .clear
        bar.titleTextAttributes = [
            .foregroundColor: UIColor(argb: Palette.shared.textPrimary),
            .font: Mono.uiFont(Mono.bold, size: 17, style: .headline),
        ]
        let navigation = UINavigationBar.appearance()
        navigation.standardAppearance = bar
        navigation.scrollEdgeAppearance = bar
        navigation.compactAppearance = bar
        navigation.tintColor = UIColor(argb: Palette.shared.textPrimary)
    }
}
