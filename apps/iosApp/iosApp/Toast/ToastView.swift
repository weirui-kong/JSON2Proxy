import SwiftUI

struct ToastView: View {
    let toast: Toast

    var body: some View {
        Text(toast.message)
            .font(Typo.small)
            .multilineTextAlignment(.center)
            .foregroundStyle(Colors.textPrimary)
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(Colors.surface, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(Colors.border))
            .padding(.horizontal, 32)
    }
}

/// Root view of the toast window: the stack pinned near the bottom, newest
/// at the bottom. Toasts rise in with a spring and fade out in place.
struct ToastOverlay: View {
    @ObservedObject var center: ToastCenter

    var body: some View {
        VStack(spacing: 8) {
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
    VStack(spacing: 8) {
        ToastView(toast: Toast("Settings saved"))
        ToastView(toast: Toast("Connected to example.net:8443"))
    }
}
