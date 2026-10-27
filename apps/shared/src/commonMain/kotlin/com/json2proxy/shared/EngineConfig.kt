package com.json2proxy.shared

import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject

/** Builds the JSON that the Go core's StartEngine accepts. */
object EngineConfig {
    const val MTU = 1500

    /**
     * Empty DoH and bootstrap values are sent as-is so the Go core applies
     * its per-mode defaults. [memoryLimitMb] caps the Go heap (iOS extension).
     */
    fun build(profile: ServerProfile, secret: String, settings: AppSettings, memoryLimitMb: Int = 0): String =
        buildJsonObject {
            putJsonObject("client") {
                put("server", profile.server.trim())
                put("client_id", profile.clientId.trim())
                put("secret", secret.trim())
                putJsonObject("obfs") {
                    put("provider", profile.obfsProvider)
                    put("participation", profile.obfsParticipation)
                }
                putJsonObject("performance") { put("mode", profile.performanceMode) }
                putJsonObject("udp") { put("enabled", profile.udpEnabled) }
            }
            put("mode", settings.routingMode.wire)
            if (settings.routingMode == RoutingMode.Custom) put("rules", settings.customRules)
            put("doh", settings.dohUrl.trim())
            put("bootstrap", settings.bootstrap.trim())
            put("mtu", MTU)
            if (memoryLimitMb > 0) put("memory_limit_mb", memoryLimitMb)
        }.toString()
}
