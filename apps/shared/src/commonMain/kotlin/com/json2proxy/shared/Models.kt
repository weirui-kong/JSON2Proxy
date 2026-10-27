package com.json2proxy.shared

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** One saved server. The secret is stored separately in [SecretStore]. */
@Serializable
data class ServerProfile(
    val id: String,
    val name: String,
    /** host:port; IPv6 addresses use [addr]:port. */
    val server: String,
    val clientId: String,
    val obfsProvider: String = "none",
    val obfsParticipation: Double = 0.0,
    val performanceMode: String = PerformanceModes.INTERACTIVE,
    val udpEnabled: Boolean = true,
)

object ObfsProviders {
    const val NONE = "none"
    val all = listOf(NONE, "iot-cam", "mysql")

    /** Participation the server and client use when a provider is enabled. */
    const val DEFAULT_PARTICIPATION = 0.65
}

object PerformanceModes {
    const val INTERACTIVE = "interactive"
    const val THROUGHPUT = "throughput"
    val all = listOf(INTERACTIVE, THROUGHPUT)
}

@Serializable
enum class RoutingMode(val wire: String, val defaultDoH: String) {
    @SerialName("global")
    Global("global", "https://dns.google/dns-query"),

    @SerialName("geoip-cn")
    GeoIpCn("geoip-cn", "https://dns.alidns.com/dns-query"),

    @SerialName("custom")
    Custom("custom", "https://dns.google/dns-query"),
}

@Serializable
data class AppSettings(
    val routingMode: RoutingMode = RoutingMode.Global,
    /** Rule text used when [routingMode] is [RoutingMode.Custom]. */
    val customRules: String = "",
    /** Empty means the default for the routing mode. */
    val dohUrl: String = "",
    /** Empty means 8.8.8.8:53. */
    val bootstrap: String = "",
    /** Keep the VPN up and block traffic when the proxy fails. */
    val killSwitch: Boolean = true,
    /** iOS Connect On Demand. Android uses the system always-on setting. */
    val autoConnect: Boolean = false,
    /** iOS includeAllNetworks; off by default because it affects system features. */
    val strictKillSwitch: Boolean = false,
) {
    val effectiveDoH: String get() = dohUrl.trim().ifEmpty { routingMode.defaultDoH }
    val effectiveBootstrap: String get() = bootstrap.trim().ifEmpty { DEFAULT_BOOTSTRAP }

    companion object {
        const val DEFAULT_BOOTSTRAP = "8.8.8.8:53"
    }
}

/** Splits host:port or [ipv6]:port. Returns null when malformed. */
fun splitHostPort(address: String): Pair<String, Int>? {
    val value = address.trim()
    val host: String
    val portText: String
    if (value.startsWith("[")) {
        val end = value.indexOf("]:")
        if (end < 0) return null
        host = value.substring(1, end)
        portText = value.substring(end + 2)
    } else {
        val colon = value.lastIndexOf(':')
        if (colon <= 0 || value.indexOf(':') != colon) return null
        host = value.substring(0, colon)
        portText = value.substring(colon + 1)
    }
    val port = portText.toIntOrNull() ?: return null
    if (host.isEmpty() || port !in 1..65535) return null
    return host to port
}

/**
 * Turns the full-width forms that CJK input methods produce (ＡＢＣ１２３,
 * "。", full-width space) into ASCII and trims. Use it for addresses, IDs
 * and secrets, never for display names.
 */
fun normalizeAscii(input: String): String = buildString(input.length) {
    for (c in input) {
        append(
            when (c) {
                in '\uFF01'..'\uFF5E' -> c - 0xFEE0
                '\u3002' -> '.'
                '\u3000' -> ' '
                else -> c
            },
        )
    }
}.trim()
