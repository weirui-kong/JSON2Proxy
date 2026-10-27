# Native apps

This directory holds the JSON2Proxy client apps built with Kotlin
Multiplatform. The Go module at the repository root remains the protocol core;
the apps reuse it through gomobile (`mobile/`) instead of reimplementing the
protocol in Kotlin. Android runs the Go engine in a `VpnService`; iOS runs it in
a Packet Tunnel extension that links only Swift and Go.

```text
shared/      KMP module shared by both apps (exported to iOS as Shared.framework)
androidApp/  Android app, Jetpack Compose
iosApp/      iOS app, SwiftUI, iOS 16+, Kotlin API surfaced through SKIE
```

## Requirements

- JDK 17 or newer and Go (version in `go.mod`)
- Android SDK with platform 37 and NDK r28c for `androidApp`; set
  `ANDROID_HOME` (or `ANDROID_NDK_HOME`) when building the Go core
- Xcode with an iOS 16+ runtime for `iosApp`; running the tunnel needs a device
  and a team with the Network Extension capability (Simulator cannot start it)

Kotlin and SKIE versions are pinned together in `gradle/libs.versions.toml`;
SKIE supports specific Kotlin releases, so upgrade them as a pair.

## Build

From the repository root, build the Go core first; the apps link
`apps/build/json2proxy-core.aar` and `apps/build/Json2ProxyCore.xcframework`:

```bash
scripts/build-mobile-core.sh all   # or: ios, android
```

Run the shared tests:

```bash
./gradlew :shared:allTests
```

Build the Android app:

```bash
./gradlew :androidApp:assembleDebug
```

Open `iosApp/iosApp.xcodeproj` in Xcode and run the `iosApp` scheme. The
"Compile Kotlin Framework" build phase calls
`./gradlew :shared:embedAndSignAppleFrameworkForXcode` before compiling Swift.
