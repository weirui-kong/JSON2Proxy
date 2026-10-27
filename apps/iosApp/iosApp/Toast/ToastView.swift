import SwiftUI
import TerminalUI

struct ToastView: View {
    let toast: Toast

    var body: some View {
        TerminalToast(toast.message)
            .padding(.horizontal, ToastLayout.sideInset)
    }
}

private enum ToastLayout {
    static let sideInset: CGFloat = 32
}

/// Root view of the toast window: the stack pinned near the bottom, newest
/// at the bottom. Toasts rise in with a spring and fade out in place.
struct ToastOverlay: View {
    @ObservedObject var center: ToastCenter

    var body: some View {
        VStack(spacing: TerminalSpacing.space8) {
            Spacer(minLength: 0)
            ForEach(center.toasts) { toast in
                ToastView(toast: toast)
                    .transition(
                        .asymmetric(
                            insertion: .move(edge: .bottom).combined(with: .opacity),
                            removal: .opacity.combined(with: .scale(scale: 0.9))
                        )
                    )
            }
        }
        .padding(.bottom, 56)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .animation(.spring(response: 0.42, dampingFraction: 0.78), value: center.toasts)
        .allowsHitTesting(false)
    }
}

#Preview {
    VStack(spacing: TerminalSpacing.space8) {
        ToastView(toast: Toast("Settings saved"))
        ToastView(toast: Toast("Connected to example.net:8443"))
    }
}
