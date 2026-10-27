import UIKit

/// Queues toasts and shows them one at a time, like Android's toast queue.
@MainActor
final class ToastCenter: ObservableObject {
    static let shared = ToastCenter()

    @Published private(set) var current: Toast?

    private var pending: [Toast] = []
    private var timer: Task<Void, Never>?
    private let maxPending = 3
    /// Gap between toasts so the exit animation finishes before the next one.
    private let gapNanoseconds: UInt64 = 250_000_000

    func show(_ message: String, duration: Toast.Duration = .short) {
        show(Toast(message, duration: duration))
    }

    func show(_ toast: Toast) {
        // Drop repeats of a message that is already on screen or waiting.
        if current?.message == toast.message || pending.contains(where: { $0.message == toast.message }) {
            return
        }
        if current == nil && timer == nil {
            present(toast)
        } else {
            pending.append(toast)
            if pending.count > maxPending {
                pending.removeFirst()
            }
        }
    }

    private func present(_ toast: Toast) {
        current = toast
        UIAccessibility.post(notification: .announcement, argument: toast.message)
        timer = Task { [weak self] in
            try? await Task.sleep(nanoseconds: toast.duration.nanoseconds)
            self?.advance()
        }
    }

    private func advance() {
        current = nil
        guard !pending.isEmpty else {
            timer = nil
            return
        }
        let next = pending.removeFirst()
        timer = Task { [weak self, gapNanoseconds] in
            try? await Task.sleep(nanoseconds: gapNanoseconds)
            self?.present(next)
        }
    }
}
