import Shared

// ProxyController keeps its state in StateFlow/SharedFlow, which are safe to
// use from any thread under the Kotlin/Native memory model.
extension ProxyController: @retroactive @unchecked Sendable {}
