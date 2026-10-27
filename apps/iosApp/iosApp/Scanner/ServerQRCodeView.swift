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
            TerminalPage {
                PathTitle(String(localized: "servers"), String(localized: "qr"))
                TerminalCard {
                    HStack {
                        Text(verbatim: Glyph.qrFile)
                        Spacer()
                        Text(verbatim: Glyph.linkScheme)
                    }
                    .font(Typo.label)
                    .foregroundStyle(Colors.textSecondary)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)
                    .accessibilityHidden(true)
                    CardDivider()
                    VStack(spacing: 14) {
                        if let image = QRCodeImage.make(from: link) {
                            Image(uiImage: image)
                                .interpolation(.none)
                                .resizable()
                                .scaledToFit()
                                .padding(16)
                                // The white card is the code's quiet zone, not a palette color.
                                .background(.white, in: RoundedRectangle(cornerRadius: 6, style: .continuous))
                                .frame(maxWidth: 280)
                                .accessibilityLabel(Text("QR code for \(profile.name)"))
                        }
                        Text(verbatim: profile.name).font(Typo.bodyBold).foregroundStyle(Colors.textPrimary)
                        Text(verbatim: profile.server).font(Typo.label).foregroundStyle(Colors.textSecondary)
                    }
                    .padding(20)
                    .frame(maxWidth: .infinity)
                }
            } bottom: {
                Banner(
                    kind: .warning,
                    text: String(localized: "This QR code contains the server secret. Only let devices you trust scan it."),
                    systemImage: "exclamationmark.triangle.fill"
                )
            }
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
