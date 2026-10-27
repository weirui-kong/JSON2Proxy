import Foundation

/// A short, non-interactive message, matching Android's `Toast`.
struct Toast: Identifiable, Equatable {
    enum Duration {
        /// Android `LENGTH_SHORT`.
        case short
        /// Android `LENGTH_LONG`.
        case long

        var nanoseconds: UInt64 {
            switch self {
            case .short: 2_000_000_000
            case .long: 3_500_000_000
            }
        }
    }

    let id = UUID()
    let message: String
    let duration: Duration

    init(_ message: String, duration: Duration = .short) {
        self.message = message
        self.duration = duration
    }
}
