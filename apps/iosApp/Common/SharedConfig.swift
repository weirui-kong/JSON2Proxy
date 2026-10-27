import Foundation

/// Identifiers shared by the app and the packet tunnel extension.
enum SharedConfig {
    static let appGroup = "group.com.json2proxy.app"
    static let tunnelBundleID = "com.json2proxy.app.PacketTunnel"
    static let keychainService = "com.json2proxy"
    /// Keychain account holding the engine configuration for the extension.
    static let engineConfigAccount = "engine-config"
    /// Darwin notification the extension posts after storing an engine notice.
    static let noticeNotification = "com.json2proxy.app.notice"
    static let noticeKindKey = "notice.kind"
    static let noticeMessageKey = "notice.message"
    /// Provider messages the app sends to read the engine's traffic stats.
    static let statsMessage = "stats"
    static let statsDetailMessage = "stats-detail"
    /// Provider message asking the extension to restart the engine with the
    /// configuration now in the keychain, keeping the tunnel up.
    static let reloadMessage = "reload"
    static let reloadSucceeded = "ok"
    /// Go heap limit for the extension; iOS stops packet tunnels near 50 MB.
    static let extensionMemoryLimitMB: Int32 = 40

    /// File in the App Group container where the extension's engine keeps
    /// the traffic totals across runs; empty when the container is missing.
    static var trafficTotalsPath: String {
        FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: appGroup)?
            .appendingPathComponent("traffic-totals.json").path ?? ""
    }

    static var groupDefaults: UserDefaults {
        UserDefaults(suiteName: appGroup) ?? .standard
    }
}
