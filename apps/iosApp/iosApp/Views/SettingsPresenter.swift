import Shared
import SwiftUI
import UIKit

/// Opens Settings full screen with UIKit's horizontal flip, which SwiftUI's
/// `sheet` and `fullScreenCover` cannot do; closing flips it back.
@MainActor
enum SettingsPresenter {
    static func present(model: AppModel) {
        guard let host = UIApplication.shared.topViewController else { return }
        weak var presented: UIViewController?
        let root = NavigationStack {
            SettingsView(onClose: { presented?.dismiss(animated: true) })
        }
        .environmentObject(model)
        .preferredColorScheme(.dark)
        .tint(Colors.accent)
        let controller = UIHostingController(rootView: root)
        // The flip only works with a full-screen presentation.
        controller.modalPresentationStyle = .fullScreen
        controller.modalTransitionStyle = .flipHorizontal
        controller.view.backgroundColor = UIColor(argb: Palette.shared.canvas)
        presented = controller
        host.present(controller, animated: true)
    }
}
