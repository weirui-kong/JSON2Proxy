import SwiftUI
import UIKit

extension View {
    /// Installs a toast window above this view's scene so toasts also appear
    /// over sheets and alerts. Apply once to each scene's root view.
    func toastWindow(_ center: ToastCenter = .shared) -> some View {
        background(ToastWindowInstaller(center: center))
    }
}

private struct ToastWindowInstaller: UIViewRepresentable {
    let center: ToastCenter

    func makeUIView(context: Context) -> SceneProbeView {
        let view = SceneProbeView()
        view.onScene = { scene in context.coordinator.install(in: scene, center: center) }
        return view
    }

    func updateUIView(_ uiView: SceneProbeView, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator() }

    @MainActor
    final class Coordinator {
        private var window: UIWindow?

        func install(in scene: UIWindowScene, center: ToastCenter) {
            guard window?.windowScene !== scene else { return }
            let host = UIHostingController(rootView: ToastOverlay(center: center))
            host.view.backgroundColor = .clear

            let window = UIWindow(windowScene: scene)
            window.windowLevel = .alert + 1
            window.backgroundColor = .clear
            // Toasts are non-interactive, so every touch falls through.
            window.isUserInteractionEnabled = false
            window.rootViewController = host
            window.isHidden = false
            self.window = window
        }
    }
}

/// Reports the window scene once the SwiftUI view lands in a window.
private final class SceneProbeView: UIView {
    var onScene: ((UIWindowScene) -> Void)?

    override func didMoveToWindow() {
        super.didMoveToWindow()
        if let scene = window?.windowScene {
            onScene?(scene)
        }
    }
}
