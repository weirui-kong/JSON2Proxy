import Json2ProxyCore
import Network
import OSLog

/// Keeps this app's own Go sockets (TCP ping and its bootstrap DNS lookups)
/// on the current Wi-Fi or cellular interface. iOS does not exclude the app
/// from its own tunnel, so unbound sockets would enter the VPN.
final class PhysicalInterfaceMonitor {
    private let monitor = NWPathMonitor(prohibitedInterfaceTypes: [.other])
    private static let logger = Logger(subsystem: "com.json2proxy.app", category: "network")

    init() {
        monitor.pathUpdateHandler = { path in
            let physical: [NWInterface.InterfaceType] = [.wifi, .cellular, .wiredEthernet]
            let name = path.availableInterfaces.first { physical.contains($0.type) }?.name ?? ""
            var error: NSError?
            if !MobileSetPhysicalInterface(name, &error), let error {
                Self.logger.error("Cannot bind to \(name, privacy: .public): \(error.localizedDescription, privacy: .public)")
            }
        }
        monitor.start(queue: DispatchQueue(label: "com.json2proxy.interface"))
    }

    deinit {
        monitor.cancel()
    }
}
