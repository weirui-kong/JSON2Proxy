package com.json2proxy.shared

enum class NoticeLevel { Info, Error }

/**
 * Events important enough to interrupt the user with a toast. Anything not
 * modelled here, such as a single DNS lookup failing, belongs in [AppLog].
 */
sealed interface Notice {
    val level: NoticeLevel
    val message: String

    data class Connected(val server: String) : Notice {
        override val level get() = NoticeLevel.Info
        override val message get() = "Connected to $server"
    }

    data class ConnectionFailed(val server: String, val reason: String) : Notice {
        override val level get() = NoticeLevel.Error
        override val message get() = "Connection failed: $reason"
    }

    /** Bootstrap DNS could not resolve an infrastructure host (proxy server or DoH). */
    data class BootstrapFailed(val host: String, val reason: String) : Notice {
        override val level get() = NoticeLevel.Error
        override val message get() = "Bootstrap DNS failed for $host: $reason"
    }

    /** The DoH server itself is unreachable, so no domain can be resolved locally. */
    data class DohUnavailable(val url: String, val reason: String) : Notice {
        override val level get() = NoticeLevel.Error
        override val message get() = "DoH unavailable: $reason"
    }
}
