import SwiftUI

struct ToastView: View {
    let toast: Toast

    var body: some View {
        Text(toast.message)
            .font(.subheadline.weight(.medium))
            .multilineTextAlignment(.center)
            .foregroundStyle(.primary)
            .padding(.horizontal, 18)
            .padding(.vertical, 12)
            .modifier(ToastBackground())
            .padding(.horizontal, 32)
    }
}

/// Liquid Glass on iOS 26 and later, a frosted material below.
private struct ToastBackground: ViewModifier {
    private let shape = RoundedRectangle(cornerRadius: 22, style: .continuous)

    func body(content: Content) -> some View {
        if #available(iOS 26.0, *) {
            content.glassEffect(.regular, in: shape)
        } else {
            content
                .background(.regularMaterial, in: shape)
                .overlay(shape.strokeBorder(.white.opacity(0.2), lineWidth: 0.5))
                .shadow(color: .black.opacity(0.15), radius: 12, y: 4)
        }
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
