import CoreImage.CIFilterBuiltins
import Shared
import SwiftUI

/// Shows a saved server as a QR code for another device to scan.
struct ServerQRCodeView: View {
    let profile: ServerProfile
    let link: String
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                if let image = QRCodeImage.make(from: link) {
                    Image(uiImage: image)
                        .interpolation(.none)
                        .resizable()
                        .scaledToFit()
                        .padding(16)
                        .background(.white, in: RoundedRectangle(cornerRadius: 24, style: .continuous))
                        .frame(maxWidth: 300)
                        .accessibilityLabel(Text("QR code for \(profile.name)"))
                }
                Text(profile.name)
                    .font(.headline)
                Label("This QR code contains the server secret. Only let devices you trust scan it.", systemImage: "lock.fill")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding(24)
            .navigationTitle("QR Code")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button { dismiss() } label: { Label("Done", systemImage: "checkmark") }
                        .labelStyle(.iconOnly)
                }
            }
        }
    }
}

enum QRCodeImage {
    static func make(from text: String) -> UIImage? {
        let filter = CIFilter.qrCodeGenerator()
        filter.message = Data(text.utf8)
        filter.correctionLevel = "M"
        // Scale the module grid up so the image stays sharp when resized.
        guard let output = filter.outputImage?.transformed(by: CGAffineTransform(scaleX: 12, y: 12)),
              let cgImage = CIContext().createCGImage(output, from: output.extent) else {
            return nil
        }
        return UIImage(cgImage: cgImage)
    }
}
