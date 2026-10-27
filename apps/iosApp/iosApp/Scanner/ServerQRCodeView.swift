import CoreImage.CIFilterBuiltins
import Shared
import SwiftUI
import TerminalUI

/// Shows a saved server as a QR code for another device to scan.
struct ServerQRCodeView: View {
    let profile: ServerProfile
    let link: String
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            TerminalPage {
                TerminalPathTitle(String(localized: "servers"), String(localized: "qr"))
                TerminalCard {
                    VStack(spacing: TerminalSpacing.space14) {
                        if let image = QRCodeImage.make(from: link) {
                            Image(uiImage: image)
                                .interpolation(.none)
                                .resizable()
                                .scaledToFit()
                                .padding(TerminalSpacing.space16)
                                // The white card is the code's quiet zone, not a palette color.
                                .background(.white, in: RoundedRectangle(cornerRadius: TerminalRadius.sm, style: .continuous))
                                .frame(maxWidth: 280)
                                .accessibilityLabel(Text("QR code for \(profile.name)"))
                        }
                        Text(verbatim: profile.name).font(TerminalType.bodyBold).foregroundStyle(TerminalColors.textPrimary)
                        Text(verbatim: profile.server).font(TerminalType.label).foregroundStyle(TerminalColors.textSecondary)
                    }
                    .padding(TerminalSpacing.space20)
                    .frame(maxWidth: .infinity)
                } header: {
                    TerminalCardHeader(Glyph.qrFile) { Text(verbatim: Glyph.linkScheme) }
                        .accessibilityHidden(true)
                }
            } footer: {
                // The warning sign is the banner's default icon.
                TerminalBanner(
                    String(localized: "This QR code contains the server secret. Only let devices you trust scan it."),
                    kind: .warning
                )
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button { dismiss() } label: { Label { Text("Done") } icon: { TerminalIcon.check.image } }
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
