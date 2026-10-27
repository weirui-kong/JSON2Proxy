import Foundation
import NetworkExtension
import Shared

/// Starts and stops the packet tunnel through NETunnelProviderManager and
/// reports its status to the shared controller. Kotlin calls start and stop
/// from any thread; all state is touched on the main actor.
final class TunnelBackend: VpnBackend, @unchecked Sendable {
    private var graph: AppGraph?
    private var manager: NETunnelProviderManager?
    private var statusObserver: NSObjectProtocol?
    private var noticeRelay: NoticeRelay?
    private var lastStatus: NEVPNStatus = .invalid

    @MainActor
    func attach(_ graph: AppGraph) {
        self.graph = graph
        statusObserver = NotificationCenter.default.addObserver(
            forName: .NEVPNStatusDidChange, object: nil, queue: .main
        ) { [weak self] note in
            guard let status = (note.object as? NEVPNConnection)?.status else { return }
            MainActor.assumeIsolated { self?.report(status) }
        }
        noticeRelay = NoticeRelay { kind, message in
            graph.controller.onEngineNotice(kind: kind, detail: message)
        }
        Task { await refreshStatus() }
    }

    func start(engineConfigJson: String) {
        Task { @MainActor in await self.startTunnel(engineConfigJson) }
    }

    func stop() {
        Task { @MainActor in await self.stopTunnel() }
    }

    @MainActor
    private func startTunnel(_ config: String) async {
        guard SharedKeychain.set(config, account: SharedConfig.engineConfigAccount) else {
            fail("Could not store the tunnel configuration in the keychain")
            return
        }
        do {
            let manager = try await loadManager()
            configure(manager)
            try await manager.saveToPreferences()
            // Required after saving, or the first start fails.
            try await manager.loadFromPreferences()
            try manager.connection.startVPNTunnel()
        } catch {
            fail(error.localizedDescription)
        }
    }

    @MainActor
    private func stopTunnel() async {
        guard let manager = try? await loadManager() else { return }
        if manager.isOnDemandEnabled {
            // Otherwise the system starts the tunnel again right away.
            manager.isOnDemandEnabled = false
            try? await manager.saveToPreferences()
        }
        manager.connection.stopVPNTunnel()
    }

    @MainActor
    private func loadManager() async throws -> NETunnelProviderManager {
        if let manager { return manager }
        let managers = try await NETunnelProviderManager.loadAllFromPreferences()
        let manager = managers.first ?? NETunnelProviderManager()
        self.manager = manager
        return manager
    }

    @MainActor
    private func configure(_ manager: NETunnelProviderManager) {
        let settings = graph?.settings.current
        let proto = (manager.protocolConfiguration as? NETunnelProviderProtocol) ?? NETunnelProviderProtocol()
        proto.providerBundleIdentifier = SharedConfig.tunnelBundleID
        proto.serverAddress = graph?.profiles.selected?.server ?? "JSON2Proxy"
        proto.includeAllNetworks = settings?.strictKillSwitch ?? false
        proto.excludeLocalNetworks = true
        proto.disconnectOnSleep = false
        manager.protocolConfiguration = proto
        manager.localizedDescription = "JSON2Proxy"
        manager.isEnabled = true
        let onDemand = settings?.autoConnect ?? false
        manager.onDemandRules = onDemand ? [NEOnDemandRuleConnect()] : nil
        manager.isOnDemandEnabled = onDemand
    }

    @MainActor
    private func refreshStatus() async {
        guard let manager = try? await loadManager() else { return }
        report(manager.connection.status)
    }

    @MainActor
    private func report(_ status: NEVPNStatus) {
        guard let controller = graph?.controller else { return }
        let previous = lastStatus
        lastStatus = status
        switch status {
        case .connecting, .reasserting:
            controller.onVpnStatus(status: .connecting, error: nil)
        case .connected:
            controller.onVpnStatus(status: .connected, error: nil)
        case .disconnecting:
            controller.onVpnStatus(status: .disconnecting, error: nil)
        case .disconnected, .invalid:
            guard previous == .connecting, let connection = manager?.connection else {
                controller.onVpnStatus(status: .disconnected, error: nil)
                return
            }
            // The tunnel never came up: surface the extension's error.
            connection.fetchLastDisconnectError { [weak self] error in
                let message = error?.localizedDescription
                Task { @MainActor in
                    self?.graph?.controller.onVpnStatus(status: .disconnected, error: message)
                }
            }
        @unknown default:
            break
        }
    }

    @MainActor
    private func fail(_ message: String) {
        graph?.controller.onVpnStatus(status: .disconnected, error: message)
    }
}

/// Receives engine notices that the extension stores in the App Group and
/// announces with a Darwin notification.
@MainActor
final class NoticeRelay {
    private let handler: (String, String) -> Void

    init(handler: @escaping (String, String) -> Void) {
        self.handler = handler
        CFNotificationCenterAddObserver(
            CFNotificationCenterGetDarwinNotifyCenter(),
            Unmanaged.passUnretained(self).toOpaque(),
            { _, observer, _, _, _ in
                guard let observer else { return }
                let relay = Unmanaged<NoticeRelay>.fromOpaque(observer).takeUnretainedValue()
                Task { @MainActor in relay.deliver() }
            },
            SharedConfig.noticeNotification as CFString,
            nil,
            .deliverImmediately
        )
    }

    deinit {
        CFNotificationCenterRemoveEveryObserver(
            CFNotificationCenterGetDarwinNotifyCenter(),
            Unmanaged.passUnretained(self).toOpaque()
        )
    }

    private func deliver() {
        let defaults = SharedConfig.groupDefaults
        guard let kind = defaults.string(forKey: SharedConfig.noticeKindKey),
              let message = defaults.string(forKey: SharedConfig.noticeMessageKey) else { return }
        handler(kind, message)
    }
}
