import Json2ProxyCore
import Network
@preconcurrency import NetworkExtension

/// Runs the Go engine on the utun interface. The app stores the engine
/// configuration in the shared keychain before starting the tunnel, so the
/// system can also start it on demand without the app.
final class PacketTunnelProvider: NEPacketTunnelProvider {
    private let platform = TunnelPlatform()
    private var pathMonitor: NWPathMonitor?

    override func startTunnel(options: [String: NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        guard let config = SharedKeychain.get(SharedConfig.engineConfigAccount) else {
            completionHandler(TunnelError.missingConfiguration)
            return
        }
        setTunnelNetworkSettings(Self.networkSettings()) { error in
            if let error {
                completionHandler(error)
                return
            }
            completionHandler(self.startEngine(config))
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        pathMonitor?.cancel()
        pathMonitor = nil
        MobileStopEngine()
        completionHandler()
    }

    /// Starts the Go engine on the utun descriptor; returns the failure, if any.
    private func startEngine(_ config: String) -> Error? {
        MobileSetPlatform(platform)
        var fd = 0
        var error: NSError?
        guard MobileTunnelFD(&fd, &error) else {
            return error ?? TunnelError.noTunnelDescriptor
        }
        guard MobileStartEngine(fd, config, &error) else {
            return error ?? TunnelError.engineFailed
        }
        // Wi-Fi and cellular changes invalidate cached DNS answers.
        let monitor = NWPathMonitor()
        monitor.pathUpdateHandler = { _ in MobileNetworkChanged() }
        monitor.start(queue: DispatchQueue(label: "com.json2proxy.path"))
        pathMonitor = monitor
        return nil
    }

    /// Addresses must match the Android VpnService builder; the engine
    /// answers DNS sent to any address.
    private static func networkSettings() -> NEPacketTunnelNetworkSettings {
        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "127.0.0.1")
        let ipv4 = NEIPv4Settings(addresses: ["172.19.0.1"], subnetMasks: ["255.255.255.252"])
        ipv4.includedRoutes = [NEIPv4Route.default()]
        settings.ipv4Settings = ipv4
        let ipv6 = NEIPv6Settings(addresses: ["fdfe:dcba:9876::1"], networkPrefixLengths: [126])
        ipv6.includedRoutes = [NEIPv6Route.default()]
        settings.ipv6Settings = ipv6
        let dns = NEDNSSettings(servers: ["172.19.0.2"])
        dns.matchDomains = [""]
        settings.dnsSettings = dns
        settings.mtu = 1500
        return settings
    }
}

enum TunnelError: LocalizedError {
    case missingConfiguration
    case noTunnelDescriptor
    case engineFailed

    var errorDescription: String? {
        switch self {
        case .missingConfiguration: String(localized: "No tunnel configuration; connect from the app first")
        case .noTunnelDescriptor: String(localized: "The tunnel interface was not found")
        case .engineFailed: String(localized: "The network engine failed to start")
        }
    }
}
