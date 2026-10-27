import Shared
import SwiftUI

/// A full page on the canvas: scrolling content, with `bottom` resting on the
/// bottom edge when the content is shorter than the screen.
///
/// `footer` is not part of the page: it follows the content that fills the
/// safe area, in the bottom inset. On Face ID iPhones it sits just above the
/// home indicator; where there is no inset it only shows once scrolled to.
struct TerminalPage<Content: View, Bottom: View, Footer: View>: View {
    @ViewBuilder var content: Content
    @ViewBuilder var bottom: Bottom
    @ViewBuilder var footer: Footer

    var body: some View {
        GeometryReader { geometry in
            let inset = geometry.safeAreaInsets.bottom
            ScrollView {
                VStack(spacing: 0) {
                    VStack(alignment: .leading, spacing: 20) {
                        content
                        Spacer(minLength: 0)
                        bottom
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 16)
                    .frame(maxWidth: .infinity, minHeight: geometry.size.height, alignment: .topLeading)
                    footer
                        .padding(.horizontal, 20)
                        .frame(maxWidth: .infinity, minHeight: Footer.self == EmptyView.self ? 0 : TerminalPageFooter.height)
                    // Keeps the end of the page above the home indicator when scrolled down.
                    Color.clear.frame(height: max(inset - (Footer.self == EmptyView.self ? 0 : TerminalPageFooter.height), 0))
                }
            }
            .ignoresSafeArea(.container, edges: .bottom)
        }
        .background(Colors.canvas.ignoresSafeArea())
        .foregroundStyle(Colors.textPrimary)
    }
}

enum TerminalPageFooter {
    /// Leaves the lower part of the 34 pt home indicator inset free.
    static let height: CGFloat = 20
}

extension TerminalPage where Footer == EmptyView {
    init(@ViewBuilder content: () -> Content, @ViewBuilder bottom: () -> Bottom) {
        self.init(content: content, bottom: bottom, footer: { EmptyView() })
    }
}

extension TerminalPage where Bottom == EmptyView, Footer == EmptyView {
    init(@ViewBuilder content: () -> Content) {
        self.init(content: content, bottom: { EmptyView() }, footer: { EmptyView() })
    }
}

/// `$ json2proxy` with a block cursor.
struct PromptTitle: View {
    var body: some View {
        HStack(spacing: 8) {
            Text(verbatim: Glyph.prompt).foregroundStyle(Colors.accent).accessibilityHidden(true)
            Text(verbatim: Glyph.command).foregroundStyle(Colors.textPrimary)
            Rectangle().fill(Colors.accent).frame(width: 10, height: 20).accessibilityHidden(true)
        }
        .font(Typo.title)
        .accessibilityElement(children: .combine)
        .accessibilityAddTraits(.isHeader)
    }
}

/// `~/servers/edit`: earlier segments are dimmed; read aloud without the punctuation.
struct PathTitle: View {
    let segments: [String]

    init(_ segments: String...) {
        self.segments = segments
    }

    var body: some View {
        let leading = Glyph.homeDir + segments.dropLast().map { $0 + Glyph.pathSeparator }.joined()
        (Text(verbatim: leading).foregroundColor(Colors.textSecondary) + Text(verbatim: segments.last ?? "").foregroundColor(Colors.textPrimary))
            .font(Typo.title)
            .lineLimit(1)
            .truncationMode(.head)
            .accessibilityLabel(Text(verbatim: segments.joined(separator: " ")))
            .accessibilityAddTraits(.isHeader)
    }
}

struct TerminalCard<Content: View>: View {
    @ViewBuilder var content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 0) { content }
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Colors.surface, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 10, style: .continuous).strokeBorder(Colors.border))
            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}

struct CardDivider: View {
    var body: some View {
        Rectangle().fill(Colors.border).frame(height: 1)
    }
}

/// `# Title`
struct SectionTitle: View {
    let text: String

    var body: some View {
        HStack(spacing: 0) {
            Text(verbatim: Glyph.section + " ").foregroundStyle(Colors.accent).accessibilityHidden(true)
            Text(text).foregroundStyle(Colors.textPrimary)
        }
        .font(Typo.section)
        .accessibilityAddTraits(.isHeader)
    }
}

/// `# Title` followed by its controls.
struct TerminalSection<Content: View>: View {
    let title: String
    @ViewBuilder var content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            SectionTitle(text: title)
            content
        }
    }
}

/// `// note`, with the comment marker hidden from VoiceOver.
struct Note: View {
    let text: String
    var color = Colors.textSecondary

    init(_ text: String, color: Color = Colors.textSecondary) {
        self.text = text
        self.color = color
    }

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 0) {
            Text(verbatim: Glyph.comment + " ").accessibilityHidden(true)
            Text(text).fixedSize(horizontal: false, vertical: true)
        }
        .font(Typo.label)
        .foregroundStyle(color)
    }
}

struct PrimaryButtonStyle: ButtonStyle {
    var height: CGFloat = 44
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(Typo.bodyBold)
            .lineLimit(1)
            .padding(.horizontal, 16)
            .frame(minHeight: height)
            .foregroundStyle(isEnabled ? Colors.onAccent : Colors.textSecondary)
            .background(isEnabled ? Colors.accent : Colors.surface, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            .opacity(configuration.isPressed ? 0.8 : 1)
    }
}

struct SecondaryButtonStyle: ButtonStyle {
    var color = Colors.textPrimary
    var borderColor = Colors.border
    var background = Colors.surface
    var height: CGFloat = 44
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(Typo.small)
            .lineLimit(1)
            .padding(.horizontal, 12)
            .frame(minHeight: height)
            .foregroundStyle(isEnabled ? color : Colors.textSecondary)
            .background(background, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(isEnabled ? borderColor : Colors.border))
            .opacity(configuration.isPressed ? 0.7 : 1)
    }
}

/// Label above a field; errors show below it.
struct TerminalField<Field: View>: View {
    let label: String
    var error: String?
    var isFocused = false
    var multiline = false
    @ViewBuilder var field: Field

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label)
                .font(Typo.label)
                .foregroundStyle(error != nil ? Colors.danger : Colors.textSecondary)
                .accessibilityHidden(true)
            field
                .font(Typo.body)
                .foregroundStyle(Colors.textPrimary)
                .tint(Colors.accent)
                .padding(.horizontal, 12)
                .padding(.vertical, multiline ? 10 : 0)
                .frame(minHeight: 46, alignment: multiline ? .topLeading : .leading)
                .background(Colors.canvas, in: RoundedRectangle(cornerRadius: 6, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 6, style: .continuous)
                        .strokeBorder(error != nil ? Colors.danger : isFocused ? Colors.accent : Colors.control)
                )
                .accessibilityLabel(Text(label))
                .accessibilityHint(error.map { Text($0) } ?? Text(verbatim: ""))
            if let error { FieldError(text: error) }
        }
    }
}

/// Placeholder text in the secondary color, which `TextField`'s prompt alone cannot set before iOS 17.
func placeholder(_ text: String) -> Text {
    Text(verbatim: text).foregroundColor(Colors.textSecondary)
}

/// `✕ message` under a field or a group of controls.
struct FieldError: View {
    let text: String

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 0) {
            Text(verbatim: Glyph.error + " ").accessibilityHidden(true)
            Text(text)
        }
        .font(Typo.label)
        .foregroundStyle(Colors.danger)
    }
}

/// Radio rows in a card: `(•)` or `( )`, a title, and an optional wire value on the right.
struct ChoiceList<Option: Hashable>: View {
    let options: [Option]
    @Binding var selection: Option
    let title: (Option) -> String
    var detail: (Option) -> String? = { _ in nil }

    var body: some View {
        TerminalCard {
            ForEach(Array(options.enumerated()), id: \.element) { index, option in
                if index > 0 { CardDivider() }
                let isSelected = option == selection
                Button {
                    selection = option
                } label: {
                    HStack(spacing: 12) {
                        Text(verbatim: isSelected ? Glyph.radioOn : Glyph.radioOff)
                            .font(Typo.bodyBold)
                            .foregroundStyle(isSelected ? Colors.accent : Colors.textSecondary)
                            .accessibilityHidden(true)
                        Text(title(option)).font(Typo.body).foregroundStyle(Colors.textPrimary)
                        Spacer(minLength: 8)
                        if let detail = detail(option) {
                            Text(verbatim: detail).font(Typo.label).foregroundStyle(Colors.textSecondary).accessibilityHidden(true)
                        }
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)
                    .frame(maxWidth: .infinity, minHeight: 52, alignment: .leading)
                    .background(isSelected ? Colors.surfaceSelected : .clear)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isSelected ? [.isSelected] : [])
            }
        }
    }
}

/// Equal-width option buttons, one selected.
struct SegmentedChoice<Option: Hashable>: View {
    let options: [Option]
    @Binding var selection: Option
    let label: (Option) -> String

    var body: some View {
        HStack(spacing: 8) {
            ForEach(options, id: \.self) { option in
                let isSelected = option == selection
                Button {
                    selection = option
                } label: {
                    Text(label(option))
                        .font(isSelected ? Typo.smallBold : Typo.small)
                        .foregroundStyle(isSelected ? Colors.accent : Colors.textPrimary)
                        .lineLimit(1)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .background(isSelected ? Colors.surfaceSelected : Colors.surface, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .strokeBorder(isSelected ? Colors.accent : Colors.border)
                        )
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isSelected ? [.isSelected] : [])
            }
        }
    }
}

/// A card row whose whole area toggles the square switch on its right.
struct SwitchRow: View {
    let label: String
    @Binding var isOn: Bool

    var body: some View {
        TerminalCard {
            Toggle(isOn: $isOn) {
                Text(label).font(Typo.body).foregroundStyle(Colors.textPrimary)
            }
            .toggleStyle(SquareSwitchStyle())
            .padding(.horizontal, 14)
            .frame(minHeight: 56)
        }
    }
}

struct SquareSwitchStyle: ToggleStyle {
    func makeBody(configuration: Configuration) -> some View {
        Button {
            configuration.isOn.toggle()
        } label: {
            HStack {
                configuration.label
                Spacer(minLength: 12)
                ZStack(alignment: configuration.isOn ? .trailing : .leading) {
                    RoundedRectangle(cornerRadius: 6, style: .continuous)
                        .fill(configuration.isOn ? Colors.accent : Colors.control)
                        .frame(width: 52, height: 30)
                    RoundedRectangle(cornerRadius: 4, style: .continuous)
                        .fill(Colors.textPrimary)
                        .frame(width: 22, height: 22)
                        .padding(4)
                }
                .animation(.easeOut(duration: 0.15), value: configuration.isOn)
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityRepresentation { Toggle(isOn: configuration.$isOn) { configuration.label } }
    }
}

enum BannerKind { case info, warning, danger }

struct Banner: View {
    let kind: BannerKind
    let text: String
    var systemImage: String?

    var body: some View {
        let (background, outline, color): (Color, Color, Color) = switch kind {
        case .info: (Colors.infoBackground, Colors.infoBorder, Colors.infoText)
        case .warning: (Colors.warningBackground, Colors.warningBorder, Colors.warningText)
        case .danger: (Colors.dangerBackground, Colors.danger, Colors.dangerText)
        }
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            if let systemImage { Image(systemName: systemImage).accessibilityHidden(true) }
            Text(text).fixedSize(horizontal: false, vertical: true)
        }
        .font(Typo.small)
        .foregroundStyle(color)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(background, in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).strokeBorder(outline))
    }
}

extension LatencyGrade {
    var color: Color {
        switch self {
        case .testing: Colors.textSecondary
        case .good: Colors.accent
        case .slow: Colors.warning
        case .failed: Colors.danger
        }
    }
}

extension PingState {
    var grade: LatencyGrade { Latency.shared.grade(ping: self) }
}

/// 24-cell character bar; decorative, the latency text says the same.
struct LatencyBar: View {
    let ping: PingState

    var body: some View {
        if let filled = Latency.shared.filledCells(ping: ping)?.intValue {
            let cells = Int(Latency.shared.BAR_CELLS)
            Group {
                if filled == 0 {
                    Text(verbatim: String(repeating: Glyph.barFailed, count: cells)).foregroundColor(Colors.border)
                } else {
                    Text(verbatim: String(repeating: Glyph.barFilled, count: filled)).foregroundColor(ping.grade.color)
                        + Text(verbatim: String(repeating: Glyph.barEmpty, count: cells - filled)).foregroundColor(Colors.border)
                }
            }
            .font(Typo.label)
            .kerning(1)
            .lineLimit(1)
            .accessibilityHidden(true)
        }
    }
}

/// Splits a formatted rate such as `4.60 MB/s` into number and unit.
func splitRate(_ rate: String) -> (number: String, unit: String) {
    guard let space = rate.lastIndex(of: " ") else { return (rate, "") }
    return (String(rate[..<space]), String(rate[rate.index(after: space)...]))
}

/// Upload and download speeds of the last minute, newest on the right. The
/// scale has a floor so idle noise stays flat.
struct TrafficChart: View {
    let history: [TrafficPoint]
    let description: String

    private static let points = 60
    private static let headroom = 1.2
    private static let floorBytes = 64.0 * 1024

    var body: some View {
        Canvas { context, size in
            for line in 1...2 {
                let y = size.height * CGFloat(line) / 3
                var grid = Path()
                grid.move(to: CGPoint(x: 0, y: y))
                grid.addLine(to: CGPoint(x: size.width, y: y))
                context.stroke(grid, with: .color(Colors.gridline), style: StrokeStyle(lineWidth: 1, dash: [2, 4]))
            }
            guard !history.isEmpty else { return }
            let peak = max(Double(history.map { max($0.uploadSpeed, $0.downloadSpeed) }.max() ?? 0) * Self.headroom, Self.floorBytes)
            let step = size.width / CGFloat(Self.points - 1)
            let start = size.width - step * CGFloat(history.count - 1)
            func point(_ index: Int, _ speed: Int64) -> CGPoint {
                CGPoint(x: start + step * CGFloat(index), y: size.height * (1 - CGFloat(Double(speed) / peak)))
            }
            var download = Path()
            var upload = Path()
            for (index, sample) in history.enumerated() {
                let down = point(index, sample.downloadSpeed)
                let up = point(index, sample.uploadSpeed)
                if index == 0 {
                    download.move(to: down)
                    upload.move(to: up)
                } else {
                    download.addLine(to: down)
                    upload.addLine(to: up)
                }
            }
            var area = download
            area.addLine(to: CGPoint(x: start + step * CGFloat(history.count - 1), y: size.height))
            area.addLine(to: CGPoint(x: start, y: size.height))
            area.closeSubpath()
            let stroke = StrokeStyle(lineWidth: 1.6, lineJoin: .round)
            context.fill(area, with: .color(Colors.syntaxKey.opacity(0.10)))
            context.stroke(download, with: .color(Colors.syntaxKey), style: stroke)
            context.stroke(upload, with: .color(Colors.syntaxNumber), style: stroke)
        }
        .frame(height: 72)
        .accessibilityElement()
        .accessibilityLabel(Text(description))
    }
}

/// Plain text that VoiceOver skips, for glyphs such as `→`.
struct GlyphText: View {
    let text: String
    var color = Colors.textSecondary
    var font = Typo.body

    var body: some View {
        Text(verbatim: text).font(font).foregroundStyle(color).accessibilityHidden(true)
    }
}
