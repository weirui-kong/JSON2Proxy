import SwiftUI

struct ToastView: View {
    let toast: Toast

    var body: some View {
        Text(toast.message)
            .font(.subheadline)
            .multilineTextAlignment(.center)
            .foregroundStyle(Color(uiColor: .systemBackground))
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .fill(Color(uiColor: .label).opacity(0.88))
            )
            .padding(.horizontal, 32)
    }
}

/// Root view of the toast window: the current toast pinned near the bottom.
struct ToastOverlay: View {
    @ObservedObject var center: ToastCenter

    var body: some View {
        ZStack(alignment: .bottom) {
            Color.clear
            if let toast = center.current {
                ToastView(toast: toast)
                    .id(toast.id)
                    .transition(.opacity)
                    .padding(.bottom, 56)
            }
        }
        .animation(.easeInOut(duration: 0.2), value: center.current)
        .allowsHitTesting(false)
    }
}

#Preview {
    ToastView(toast: Toast("Connected to example.net:8443"))
}
