import UIKit

/// Shows toasts as a stack: each new toast enters at the bottom and pushes
/// the ones still on screen up. Every toast has its own timer.
@MainActor
final class ToastCenter: ObservableObject {
    static let shared = ToastCenter()

    /// Visible toasts, oldest first; the last one sits at the bottom.
    @Published private(set) var toasts: [Toast] = []

    private var timers: [UUID: Task<Void, Never>] = [:]
    private let maxVisible = 3

    func show(_ message: String, duration: Toast.Duration = .short) {
        show(Toast(message, duration: duration))
    }

    func show(_ toast: Toast) {
        // The same message again keeps the visible one up instead of stacking a copy.
        if let visible = toasts.first(where: { $0.message == toast.message }) {
            schedule(visible.id, after: toast.duration)
            return
        }
        toasts.append(toast)
        if toasts.count > maxVisible, let oldest = toasts.first {
            dismiss(oldest.id)
        }
        UIAccessibility.post(notification: .announcement, argument: toast.message)
        schedule(toast.id, after: toast.duration)
    }

    func dismiss(_ id: UUID) {
        timers.removeValue(forKey: id)?.cancel()
        toasts.removeAll { $0.id == id }
    }

    private func schedule(_ id: UUID, after duration: Toast.Duration) {
        timers[id]?.cancel()
        timers[id] = Task { [weak self] in
            try? await Task.sleep(nanoseconds: duration.nanoseconds)
            guard !Task.isCancelled else { return }
            self?.dismiss(id)
        }
    }
}
