package com.json2proxy.shared

sealed interface ConnectionState {
    data object Disconnected : ConnectionState
    data class Connecting(val server: String) : ConnectionState
    data class Connected(val server: String) : ConnectionState
    data class Failed(val message: String) : ConnectionState
}
