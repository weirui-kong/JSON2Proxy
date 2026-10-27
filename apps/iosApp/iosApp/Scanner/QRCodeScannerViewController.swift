@preconcurrency import AVFoundation
import UIKit

/// The scanner's verdict on a payload: accept a value, or reject it with a
/// message shown in the hint pill while scanning continues.
enum ScanDecision<Value> {
    case accept(Value)
    case reject(String)
}

/// Generic QR scanner: `validator` inspects each payload. On the first
/// accepted one the scanner dismisses itself, then calls `handler`.
/// Rejection messages are shown instead of the raw payload, which may hold a
/// secret.
@MainActor
final class QRCodeScannerViewController: UIViewController, @preconcurrency AVCaptureMetadataOutputObjectsDelegate {

    private enum Layout {
        static let viewfinderMaxSide: CGFloat = 260
        static let viewfinderWidthRatio: CGFloat = 0.68
        static let viewfinderCenterOffset: CGFloat = -30
        static let viewfinderCornerRadius: CGFloat = 32
        static let bracketLength: CGFloat = 26
        static let bracketLineWidth: CGFloat = 5
        static let codeHighlightPadding: CGFloat = 10
        static let glassButtonSide: CGFloat = 44
        static let hintHeight: CGFloat = 44
        static let hintBottomInset: CGFloat = 24
        static let dimAlpha: CGFloat = 0.35
        static let successDismissDelay: TimeInterval = 0.35
        static let hintResetDelay: TimeInterval = 2
    }

    private let hint: String
    // Type-erased validator + handler, so the controller stays non-generic (@objc delegate conformance).
    private let resolve: (String) -> ScanDecision<() -> Void>

    private let session = AVCaptureSession()
    private let sessionQueue = DispatchQueue(label: "com.json2proxy.qrscanner")
    private var previewLayer: AVCaptureVideoPreviewLayer?
    private var isHandled = false
    private var hasRequestedCamera = false
    private var lastRejectedPayload: String?
    private var hintResetWorkItem: DispatchWorkItem?

    private let dimLayer = CAShapeLayer()
    private let bracketLayer = CAShapeLayer()
    private var viewfinderRect: CGRect = .zero

    private lazy var closeButton = makeGlassButton(
        systemImageName: "xmark", label: String(localized: "Close"), action: #selector(closeTapped)
    )
    private lazy var torchButton = makeGlassButton(
        systemImageName: "flashlight.off.fill", label: String(localized: "Flashlight"), action: #selector(torchTapped)
    )
    private lazy var hintView = makeGlassView(cornerRadius: Layout.hintHeight / 2, isInteractive: false)

    private let hintIcon = UIImageView(image: UIImage(systemName: "qrcode.viewfinder"))

    private let hintLabel: UILabel = {
        let label = UILabel()
        label.font = .preferredFont(forTextStyle: .subheadline)
        label.textColor = .white
        label.lineBreakMode = .byTruncatingMiddle
        return label
    }()

    init<Value>(hint: String, validator: @escaping (String) -> ScanDecision<Value>, handler: @escaping (Value) -> Void) {
        self.hint = hint
        self.resolve = { raw in
            switch validator(raw) {
            case .accept(let value): .accept({ handler(value) })
            case .reject(let message): .reject(message)
            }
        }
        super.init(nibName: nil, bundle: nil)
        modalPresentationStyle = .fullScreen
        // Camera UI stays dark regardless of the app appearance.
        overrideUserInterfaceStyle = .dark
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    /// Zooms out of `sourceView` on iOS 18+; falls back to the default modal transition otherwise.
    func present(from host: UIViewController, sourceView: (() -> UIView?)? = nil) {
        if #available(iOS 18.0, *), let sourceView {
            preferredTransition = .zoom { _ in sourceView() }
        }
        host.present(self, animated: true)
    }

    override var preferredStatusBarStyle: UIStatusBarStyle { .lightContent }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black
        setupOverlay()
    }

    // Deferred to viewDidAppear so the permission alert can be presented.
    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        guard !hasRequestedCamera else { return }
        hasRequestedCamera = true
        requestCameraAccess()
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        previewLayer?.frame = view.bounds
        layoutViewfinder()
    }

    override func viewDidDisappear(_ animated: Bool) {
        super.viewDidDisappear(animated)
        sessionQueue.async { [session] in
            if session.isRunning { session.stopRunning() }
        }
    }

    // MARK: - Camera

    private func requestCameraAccess() {
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized:
            configureSession()
        case .notDetermined:
            AVCaptureDevice.requestAccess(for: .video) { granted in
                DispatchQueue.main.async { [weak self] in
                    granted ? self?.configureSession() : self?.showPermissionDenied()
                }
            }
        default:
            showPermissionDenied()
        }
    }

    private func configureSession() {
        guard let device = AVCaptureDevice.default(for: .video),
              let input = try? AVCaptureDeviceInput(device: device),
              session.canAddInput(input) else {
            showFatal(String(localized: "Camera unavailable"))
            return
        }
        session.addInput(input)

        let output = AVCaptureMetadataOutput()
        guard session.canAddOutput(output) else {
            showFatal(String(localized: "Failed to set up QR scanning"))
            return
        }
        session.addOutput(output)
        output.setMetadataObjectsDelegate(self, queue: .main)
        output.metadataObjectTypes = [.qr]

        let preview = AVCaptureVideoPreviewLayer(session: session)
        preview.videoGravity = .resizeAspectFill
        preview.frame = view.bounds
        view.layer.insertSublayer(preview, at: 0)
        previewLayer = preview

        torchButton.isHidden = !device.hasTorch
        sessionQueue.async { [weak self, session] in
            session.startRunning()
            DispatchQueue.main.async { self?.updateRectOfInterest() }
        }
    }

    // Restrict recognition to the viewfinder so codes outside it are ignored.
    private func updateRectOfInterest() {
        guard let previewLayer, !viewfinderRect.isEmpty,
              let output = session.outputs.first as? AVCaptureMetadataOutput else { return }
        output.rectOfInterest = previewLayer.metadataOutputRectConverted(fromLayerRect: viewfinderRect)
    }

    // The delegate queue is .main; the @preconcurrency conformance checks that at run time.
    func metadataOutput(_ output: AVCaptureMetadataOutput, didOutput metadataObjects: [AVMetadataObject], from connection: AVCaptureConnection) {
        guard let code = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
              let raw = code.stringValue else { return }
        handle(code, raw: raw)
    }

    private func handle(_ code: AVMetadataMachineReadableCodeObject, raw: String) {
        guard !isHandled else { return }
        let action: () -> Void
        switch resolve(raw) {
        case .reject(let message):
            // The same invalid code stays in view and fires every frame; only react once per payload.
            guard raw != lastRejectedPayload else { return }
            lastRejectedPayload = raw
            UINotificationFeedbackGenerator().notificationOccurred(.warning)
            showRejection(message)
            return
        case .accept(let accepted):
            action = accepted
        }
        isHandled = true
        UINotificationFeedbackGenerator().notificationOccurred(.success)
        sessionQueue.async { [session] in session.stopRunning() }
        if let codeBounds = previewLayer?.transformedMetadataObject(for: code)?.bounds {
            snapBrackets(to: codeBounds.insetBy(dx: -Layout.codeHighlightPadding, dy: -Layout.codeHighlightPadding))
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + Layout.successDismissDelay) { [weak self] in
            self?.dismiss(animated: true, completion: action)
        }
    }

    // MARK: - UI

    private func setupOverlay() {
        dimLayer.fillRule = .evenOdd
        dimLayer.fillColor = UIColor.black.withAlphaComponent(Layout.dimAlpha).cgColor
        view.layer.addSublayer(dimLayer)

        bracketLayer.strokeColor = UIColor.white.cgColor
        bracketLayer.fillColor = UIColor.clear.cgColor
        bracketLayer.lineWidth = Layout.bracketLineWidth
        bracketLayer.lineCap = .round
        bracketLayer.lineJoin = .round
        view.layer.addSublayer(bracketLayer)

        hintIcon.tintColor = .white
        hintIcon.setContentHuggingPriority(.required, for: .horizontal)
        hintLabel.text = hint
        let hintStack = UIStackView(arrangedSubviews: [hintIcon, hintLabel])
        hintStack.spacing = 8
        hintStack.alignment = .center
        hintStack.translatesAutoresizingMaskIntoConstraints = false
        hintView.contentView.addSubview(hintStack)

        [closeButton, torchButton, hintView].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
            view.addSubview($0)
        }
        let guide = view.safeAreaLayoutGuide
        NSLayoutConstraint.activate([
            hintStack.leadingAnchor.constraint(equalTo: hintView.contentView.leadingAnchor, constant: 16),
            hintStack.trailingAnchor.constraint(equalTo: hintView.contentView.trailingAnchor, constant: -18),
            hintStack.topAnchor.constraint(equalTo: hintView.contentView.topAnchor),
            hintStack.bottomAnchor.constraint(equalTo: hintView.contentView.bottomAnchor),

            closeButton.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: 16),
            closeButton.topAnchor.constraint(equalTo: guide.topAnchor, constant: 8),
            closeButton.widthAnchor.constraint(equalToConstant: Layout.glassButtonSide),
            closeButton.heightAnchor.constraint(equalToConstant: Layout.glassButtonSide),

            torchButton.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -16),
            torchButton.topAnchor.constraint(equalTo: closeButton.topAnchor),
            torchButton.widthAnchor.constraint(equalTo: closeButton.widthAnchor),
            torchButton.heightAnchor.constraint(equalTo: closeButton.heightAnchor),

            hintView.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            hintView.leadingAnchor.constraint(greaterThanOrEqualTo: view.leadingAnchor, constant: 24),
            hintView.bottomAnchor.constraint(equalTo: guide.bottomAnchor, constant: -Layout.hintBottomInset),
            hintView.heightAnchor.constraint(equalToConstant: Layout.hintHeight),
        ])
        torchButton.isHidden = true
    }

    private func layoutViewfinder() {
        let side = min(view.bounds.width * Layout.viewfinderWidthRatio, Layout.viewfinderMaxSide)
        let rect = CGRect(
            x: (view.bounds.width - side) / 2,
            y: view.bounds.midY - side / 2 + Layout.viewfinderCenterOffset,
            width: side, height: side
        )
        guard rect != viewfinderRect else { return }
        viewfinderRect = rect

        let dimPath = UIBezierPath(rect: view.bounds)
        dimPath.append(UIBezierPath(roundedRect: rect, cornerRadius: Layout.viewfinderCornerRadius))
        dimLayer.path = dimPath.cgPath

        bracketLayer.frame = view.bounds
        bracketLayer.path = Self.bracketPath(in: rect)
        startBreathing()
        updateRectOfInterest()
    }

    private func startBreathing() {
        let animation = CABasicAnimation(keyPath: "opacity")
        animation.fromValue = 1
        animation.toValue = 0.55
        animation.duration = 1.1
        animation.autoreverses = true
        animation.repeatCount = .infinity
        animation.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
        bracketLayer.add(animation, forKey: "breathing")
    }

    // Brackets glide from the viewfinder onto the detected code, like the system Code Scanner.
    private func snapBrackets(to rect: CGRect) {
        bracketLayer.removeAnimation(forKey: "breathing")
        let newPath = Self.bracketPath(in: rect)
        let animation = CASpringAnimation(keyPath: "path")
        animation.fromValue = bracketLayer.path
        animation.toValue = newPath
        animation.damping = 18
        animation.duration = animation.settlingDuration
        bracketLayer.path = newPath
        bracketLayer.strokeColor = UIColor.systemYellow.cgColor
        bracketLayer.add(animation, forKey: "snap")
    }

    // Rounded corner brackets hugging a continuous rounded rect; element count is fixed so paths can animate into each other.
    private static func bracketPath(in rect: CGRect) -> CGPath {
        let radius = min(Layout.viewfinderCornerRadius, rect.width / 4, rect.height / 4)
        let length = max(0, min(Layout.bracketLength, rect.width / 2 - radius, rect.height / 2 - radius))
        let path = UIBezierPath()
        path.move(to: CGPoint(x: rect.minX, y: rect.minY + radius + length))
        path.addArc(withCenter: CGPoint(x: rect.minX + radius, y: rect.minY + radius), radius: radius, startAngle: .pi, endAngle: .pi * 1.5, clockwise: true)
        path.addLine(to: CGPoint(x: rect.minX + radius + length, y: rect.minY))
        path.move(to: CGPoint(x: rect.maxX - radius - length, y: rect.minY))
        path.addArc(withCenter: CGPoint(x: rect.maxX - radius, y: rect.minY + radius), radius: radius, startAngle: .pi * 1.5, endAngle: 0, clockwise: true)
        path.addLine(to: CGPoint(x: rect.maxX, y: rect.minY + radius + length))
        path.move(to: CGPoint(x: rect.maxX, y: rect.maxY - radius - length))
        path.addArc(withCenter: CGPoint(x: rect.maxX - radius, y: rect.maxY - radius), radius: radius, startAngle: 0, endAngle: .pi * 0.5, clockwise: true)
        path.addLine(to: CGPoint(x: rect.maxX - radius - length, y: rect.maxY))
        path.move(to: CGPoint(x: rect.minX + radius + length, y: rect.maxY))
        path.addArc(withCenter: CGPoint(x: rect.minX + radius, y: rect.maxY - radius), radius: radius, startAngle: .pi * 0.5, endAngle: .pi, clockwise: true)
        path.addLine(to: CGPoint(x: rect.minX, y: rect.maxY - radius - length))
        return path.cgPath
    }

    private func showRejection(_ message: String) {
        hintResetWorkItem?.cancel()
        hintIcon.image = UIImage(systemName: "exclamationmark.triangle.fill")
        hintIcon.tintColor = .systemYellow
        hintLabel.text = message
        let work = DispatchWorkItem { [weak self] in
            guard let self else { return }
            self.hintIcon.image = UIImage(systemName: "qrcode.viewfinder")
            self.hintIcon.tintColor = .white
            self.hintLabel.text = self.hint
            self.lastRejectedPayload = nil
        }
        hintResetWorkItem = work
        DispatchQueue.main.asyncAfter(deadline: .now() + Layout.hintResetDelay, execute: work)
    }

    // Liquid Glass on iOS 26+, system material blur below.
    private func makeGlassView(cornerRadius: CGFloat, isInteractive: Bool) -> UIVisualEffectView {
        let effectView: UIVisualEffectView
        if #available(iOS 26.0, *) {
            let glass = UIGlassEffect()
            glass.isInteractive = isInteractive
            effectView = UIVisualEffectView(effect: glass)
        } else {
            effectView = UIVisualEffectView(effect: UIBlurEffect(style: .systemThinMaterialDark))
        }
        effectView.layer.cornerRadius = cornerRadius
        effectView.layer.cornerCurve = .continuous
        effectView.clipsToBounds = true
        return effectView
    }

    private func makeGlassButton(systemImageName: String, label: String, action: Selector) -> UIVisualEffectView {
        let effectView = makeGlassView(cornerRadius: Layout.glassButtonSide / 2, isInteractive: true)
        let button = UIButton(type: .system)
        button.setImage(UIImage(systemName: systemImageName, withConfiguration: UIImage.SymbolConfiguration(textStyle: .body, scale: .large)), for: .normal)
        button.tintColor = .white
        button.accessibilityLabel = label
        button.addTarget(self, action: action, for: .touchUpInside)
        button.translatesAutoresizingMaskIntoConstraints = false
        effectView.contentView.addSubview(button)
        NSLayoutConstraint.activate([
            button.leadingAnchor.constraint(equalTo: effectView.contentView.leadingAnchor),
            button.trailingAnchor.constraint(equalTo: effectView.contentView.trailingAnchor),
            button.topAnchor.constraint(equalTo: effectView.contentView.topAnchor),
            button.bottomAnchor.constraint(equalTo: effectView.contentView.bottomAnchor),
        ])
        return effectView
    }

    private func showPermissionDenied() {
        let alert = UIAlertController(title: nil, message: String(localized: "Camera access is disabled"), preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: String(localized: "Cancel"), style: .cancel) { [weak self] _ in
            self?.presentingViewController?.dismiss(animated: true)
        })
        alert.addAction(UIAlertAction(title: String(localized: "Settings"), style: .default) { [weak self] _ in
            if let url = URL(string: UIApplication.openSettingsURLString) {
                UIApplication.shared.open(url)
            }
            self?.presentingViewController?.dismiss(animated: true)
        })
        present(alert, animated: true)
    }

    private func showFatal(_ message: String) {
        let alert = UIAlertController(title: nil, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: String(localized: "OK"), style: .default) { [weak self] _ in
            self?.presentingViewController?.dismiss(animated: true)
        })
        present(alert, animated: true)
    }

    // MARK: - Actions

    @objc private func closeTapped() {
        dismiss(animated: true)
    }

    @objc private func torchTapped(_ sender: UIButton) {
        guard let device = (session.inputs.first as? AVCaptureDeviceInput)?.device, device.hasTorch,
              (try? device.lockForConfiguration()) != nil else { return }
        let isOn = device.torchMode != .on
        device.torchMode = isOn ? .on : .off
        device.unlockForConfiguration()
        sender.setImage(UIImage(systemName: isOn ? "flashlight.on.fill" : "flashlight.off.fill", withConfiguration: UIImage.SymbolConfiguration(textStyle: .body, scale: .large)), for: .normal)
        sender.tintColor = isOn ? .systemYellow : .white
    }
}
