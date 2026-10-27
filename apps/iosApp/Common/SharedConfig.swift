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
    /// Go heap limit for the extension; iOS stops packet tunnels near 50 MB.
    static let extensionMemoryLimitMB: Int32 = 40

    static var groupDefaults: UserDefaults {
        UserDefaults(suiteName: appGroup) ?? .standard
    }
}
