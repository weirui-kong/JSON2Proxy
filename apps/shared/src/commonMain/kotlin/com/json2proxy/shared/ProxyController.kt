package com.json2proxy.shared

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.IO
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import kotlinx.coroutines.withContext

/**
 * Entry point the native UIs talk to. It owns the connection state, turns
 * engine events into notices, and runs TCP pings. The VPN itself is started
 * by the platform [VpnBackend].
 */
class ProxyController(
    val profiles: ProfileRepository,
    val settings: SettingsRepository,
    private val backend: VpnBackend,
    private val core: CoreBridge,
    /** Go heap limit passed to the engine; iOS sets it for the extension. */
    private val memoryLimitMb: Int = 0,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow<ConnectionState>(ConnectionState.Disconnected)
    val state: StateFlow<ConnectionState> = _state.asStateFlow()

    private val _notices = MutableSharedFlow<Notice>(
        extraBufferCapacity = 8,
        onBufferOverflow = BufferOverflow.DROP_OLDEST,
    )
    /** One-shot events for the platform toast. Not replayed to late collectors. */
    val notices: SharedFlow<Notice> = _notices.asSharedFlow()

    private val _pings = MutableStateFlow<Map<String, PingState>>(emptyMap())
    /** Latest ping per server address (host:port), so editing an address drops its old result. */
    val pings: StateFlow<Map<String, PingState>> = _pings.asStateFlow()

    val platform: String = platformName()

    private var activeServer: String = ""

    fun connect() {
        val config = engineConfigForSelected() ?: return
        _state.value = ConnectionState.Connecting(activeServer)
        AppLog.i(TAG, "Connecting to $activeServer")
        backend.start(config)
    }

    fun disconnect() {
        AppLog.i(TAG, "Disconnect requested")
        backend.stop()
    }

    /**
     * Engine configuration for the selected server, or null after reporting
     * why it cannot be built. Android's always-on service calls it directly.
     */
    fun engineConfigForSelected(): String? {
        val profile = profiles.selected ?: return fail(FailureReason.NoServerSelected, "")
        val secret = profiles.secret(profile.id) ?: return fail(FailureReason.MissingSecret, profile.name)
        activeServer = profile.name
        return EngineConfig.build(profile, secret, settings.current, memoryLimitMb)
    }

    /** Called by the platform when the system VPN changes state. */
    fun onVpnStatus(status: VpnStatus, error: String? = null) {
        when (status) {
            VpnStatus.Connecting -> _state.value = ConnectionState.Connecting(serverLabel())
            VpnStatus.Connected -> _state.value = ConnectionState.Connected(serverLabel())
            VpnStatus.Disconnecting -> _state.value = ConnectionState.Disconnecting
            VpnStatus.Disconnected -> {
                if (error != null) {
                    fail(FailureReason.Error, error)
                } else {
                    _state.value = ConnectionState.Disconnected
                }
            }
        }
    }

    /** Called with events the Go engine reports through Platform.Notice. */
    fun onEngineNotice(kind: String, detail: String) {
        val notice = when (val mapped = Notice.fromEngine(kind, detail)) {
            null -> {
                AppLog.w(TAG, "Unknown engine notice $kind: $detail")
                return
            }
            // The engine only knows the address; users know the server by name.
            is Notice.Connected -> Notice.Connected(serverLabel().ifEmpty { detail })
            else -> mapped
        }
        AppLog.i(TAG, "Engine notice $kind: $detail")
        _notices.tryEmit(notice)
        val proxyDown = notice is Notice.ConnectionFailed || notice is Notice.ProxyUnavailable
        if (proxyDown && !settings.current.killSwitch) {
            // Without the kill switch, a dead proxy should not keep blocking traffic.
            AppLog.i(TAG, "Kill switch off; stopping the VPN")
            backend.stop()
        }
    }

    fun validateRules(text: String): String? = core.validateRules(text)

    fun ping(profileId: String) {
        val profile = profiles.profiles.value.firstOrNull { it.id == profileId } ?: return
        scope.launch { pingOne(profile) }
    }

    fun pingAll() {
        val list = profiles.profiles.value
        scope.launch {
            val permits = Semaphore(PING_CONCURRENCY)
            list.forEach { profile -> launch { permits.withPermit { pingOne(profile) } } }
        }
    }

    private suspend fun pingOne(profile: ServerProfile) {
        setPing(profile.server, PingState.Testing)
        val target = splitHostPort(profile.server)
        if (target == null) {
            setPing(profile.server, PingState.Failed(""))
            return
        }
        val result = withContext(Dispatchers.IO) {
            core.tcpPing(settings.current.bootstrap.trim(), target.first, target.second, PING_TIMEOUT_MS, PING_ATTEMPTS)
        }
        setPing(
            profile.server,
            if (result.successes > 0) PingState.Success(result.connectMillis, result.address)
            else PingState.Failed(result.error),
        )
    }

    private fun setPing(server: String, state: PingState) {
        _pings.update { it + (server to state) }
    }

    private fun serverLabel(): String = activeServer.ifEmpty { profiles.selected?.name.orEmpty() }

    private fun fail(reason: FailureReason, detail: String): String? {
        _state.value = ConnectionState.Failed(reason, detail)
        AppLog.e(TAG, "Connection failed: $reason $detail")
        _notices.tryEmit(Notice.ConnectionFailed(reason, detail))
        return null
    }

    private companion object {
        const val TAG = "ProxyController"
        const val PING_CONCURRENCY = 4
        const val PING_TIMEOUT_MS = 3000
        const val PING_ATTEMPTS = 3
    }
}
