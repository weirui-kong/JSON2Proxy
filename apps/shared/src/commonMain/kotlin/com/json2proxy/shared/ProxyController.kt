package com.json2proxy.shared

import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Entry point the native UIs talk to. The Go core is not bound yet, so
 * [connect] only walks through the state transitions the UI must handle.
 */
class ProxyController {
    private val _state = MutableStateFlow<ConnectionState>(ConnectionState.Disconnected)
    val state: StateFlow<ConnectionState> = _state.asStateFlow()

    private val _notices = MutableSharedFlow<Notice>(
        extraBufferCapacity = 8,
        onBufferOverflow = BufferOverflow.DROP_OLDEST,
    )
    /** One-shot events for the platform toast. Not replayed to late collectors. */
    val notices: SharedFlow<Notice> = _notices.asSharedFlow()

    val platform: String = platformName()

    suspend fun connect(server: String) {
        if (server.isBlank()) {
            fail(server, "Server address is empty")
            return
        }
        _state.value = ConnectionState.Connecting(server)
        AppLog.i(TAG, "Connecting to $server")
        delay(500)
        _state.value = ConnectionState.Connected(server)
        _notices.tryEmit(Notice.Connected(server))
    }

    fun disconnect() {
        _state.value = ConnectionState.Disconnected
        AppLog.i(TAG, "Disconnected")
    }

    private fun fail(server: String, reason: String) {
        _state.value = ConnectionState.Failed(reason)
        AppLog.e(TAG, "Connection to '$server' failed: $reason")
        _notices.tryEmit(Notice.ConnectionFailed(server, reason))
    }

    private companion object {
        const val TAG = "ProxyController"
    }
}
