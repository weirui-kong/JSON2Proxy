# Native apps

This directory holds the JSON2Proxy client apps built with Kotlin
Multiplatform. The Go module at the repository root remains the protocol core;
the apps reuse it instead of reimplementing the protocol in Kotlin.

The KMP project has not been created yet. When it is, it lives entirely under
this directory so its Gradle build stays separate from the Go toolchain.
