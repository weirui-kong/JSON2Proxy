# Native apps

This directory holds the JSON2Proxy client apps built with Kotlin
Multiplatform. The Go module at the repository root remains the protocol core;
the apps reuse it instead of reimplementing the protocol in Kotlin. The Go core
is not bound yet, so `ProxyController` only simulates connection states.

```text
shared/      KMP module shared by both apps (exported to iOS as Shared.framework)
androidApp/  Android app, Jetpack Compose
iosApp/      iOS app, SwiftUI, iOS 16+, Kotlin API surfaced through SKIE
```

## Requirements

- JDK 17 or newer
- Android Studio (or the Android SDK with platform 36) for `androidApp`
- Xcode with an iOS 16+ simulator runtime for `iosApp`

Kotlin and SKIE versions are pinned together in `gradle/libs.versions.toml`;
SKIE supports specific Kotlin releases, so upgrade them as a pair.

## Build

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
