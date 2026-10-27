import Foundation
import Shared

@MainActor
final class ConnectionViewModel: ObservableObject {
    @Published private(set) var state: ConnectionState
    @Published var server = ""
    let platform: String

    private let controller: ProxyController
    private let toasts: ToastCenter

    init(toasts: ToastCenter = .shared) {
        let controller = ProxyController()
        self.controller = controller
        self.toasts = toasts
        self.state = controller.state.value
        self.platform = controller.platform
    }

    /// SKIE exposes the Kotlin StateFlow as an AsyncSequence.
    func observeState() async {
        for await state in controller.state {
            self.state = state
        }
    }

    func observeNotices() async {
        for await notice in controller.notices {
            toasts.show(Toast(notice: notice))
        }
    }

    func connect() {
        let server = server
        Task {
            try? await controller.connect(server: server)
        }
    }

    func disconnect() {
        controller.disconnect()
    }
}

extension Toast {
    init(notice: any Notice) {
        switch notice.level {
        case .info: self.init(notice.message, duration: .short)
        case .error: self.init(notice.message, duration: .long)
        }
    }
}

extension ConnectionState {
    var isActive: Bool {
        switch onEnum(of: self) {
        case .connecting, .connected: true
        case .disconnected, .failed: false
        }
    }

    var label: String {
        switch onEnum(of: self) {
        case .disconnected: "Disconnected"
        case .connecting(let state): "Connecting to \(state.server)…"
        case .connected(let state): "Connected to \(state.server)"
        case .failed(let state): "Failed: \(state.message)"
        }
    }
}
