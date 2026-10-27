package com.json2proxy.android.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.ConnectivityManager
import android.net.Network
import android.net.VpnService
import android.os.Build
import android.os.Handler
import android.os.Looper
import com.json2proxy.android.JsonProxyApp
import com.json2proxy.android.MainActivity
import com.json2proxy.android.R
import com.json2proxy.android.platform.AppPlatform
import com.json2proxy.android.platform.logFromGo
import com.json2proxy.core.mobile.Mobile
import com.json2proxy.core.mobile.Platform
import com.json2proxy.shared.AppLog
import com.json2proxy.shared.VpnStatus

/**
 * Owns the TUN interface and the Go engine. Started by the app to connect,
 * or by the system for an always-on VPN, in which case the configuration is
 * rebuilt from the saved profile.
 */
class ProxyVpnService : VpnService() {
    private val main = Handler(Looper.getMainLooper())
    private var running = false
    private var networkCallback: ConnectivityManager.NetworkCallback? = null

    private val controller get() = (application as JsonProxyApp).graph.controller

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_DISCONNECT) {
            shutdown()
            return START_NOT_STICKY
        }
        startInForeground()
        // Without an extra this is an always-on start (or a restart after the
        // process died), so rebuild the configuration from storage.
        val config = intent?.getStringExtra(EXTRA_CONFIG) ?: controller.engineConfigForSelected()
        if (config == null) {
            shutdown()
            return START_NOT_STICKY
        }
        if (running) stopEngine()
        report(VpnStatus.Connecting)
        try {
            startEngine(config)
            report(VpnStatus.Connected)
        } catch (e: Exception) {
            AppLog.e(TAG, "VPN start failed: ${e.message}")
            shutdown(e.message ?: "VPN start failed")
        }
        return START_STICKY
    }

    override fun onRevoke() {
        // Another VPN took over or the user revoked the permission.
        shutdown()
    }

    override fun onDestroy() {
        if (running) stopEngine()
        super.onDestroy()
    }

    private fun startEngine(config: String) {
        Mobile.setPlatform(EnginePlatform())
        val tun = Builder()
            .setSession(getString(R.string.app_name))
            .setMtu(MTU)
            .addAddress("172.19.0.1", 30)
            .addRoute("0.0.0.0", 0)
            .addAddress("fdfe:dcba:9876::1", 126)
            .addRoute("::", 0)
            // Any address works: the engine answers all DNS on port 53.
            .addDnsServer("172.19.0.2")
            // The app's own sockets (TCP ping, engine) stay outside the tunnel.
            .addDisallowedApplication(packageName)
            .setConfigureIntent(openAppIntent())
            .establish() ?: throw IllegalStateException("VPN permission was revoked")
        // The Go engine takes ownership of the descriptor and closes it.
        Mobile.startEngine(tun.detachFd().toLong(), config)
        running = true
        registerNetworkCallback()
    }

    private fun stopEngine() {
        unregisterNetworkCallback()
        Mobile.stopEngine()
        Mobile.setPlatform(AppPlatform)
        running = false
    }

    private fun shutdown(error: String? = null) {
        if (running) stopEngine()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
        report(VpnStatus.Disconnected, error)
    }

    private fun report(status: VpnStatus, error: String? = null) {
        main.post { controller.onVpnStatus(status, error) }
    }

    private fun registerNetworkCallback() {
        val connectivity = getSystemService(ConnectivityManager::class.java)
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) = Mobile.networkChanged()
        }
        connectivity.registerDefaultNetworkCallback(callback)
        networkCallback = callback
    }

    private fun unregisterNetworkCallback() {
        networkCallback?.let { getSystemService(ConnectivityManager::class.java).unregisterNetworkCallback(it) }
        networkCallback = null
    }

    private fun startInForeground() {
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, getString(R.string.notification_channel), NotificationManager.IMPORTANCE_LOW),
        )
        val disconnect = PendingIntent.getService(
            this, 1,
            Intent(this, ProxyVpnService::class.java).setAction(ACTION_DISCONNECT),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = Notification.Builder(this, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentTitle(getString(R.string.app_name))
            .setContentText(getString(R.string.notification_text))
            .setContentIntent(openAppIntent())
            .setOngoing(true)
            .addAction(Notification.Action.Builder(null, getString(R.string.action_disconnect), disconnect).build())
            .build()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            startForeground(NOTIFICATION_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_SYSTEM_EXEMPTED)
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun openAppIntent(): PendingIntent = PendingIntent.getActivity(
        this, 0, Intent(this, MainActivity::class.java), PendingIntent.FLAG_IMMUTABLE,
    )

    /** Engine callbacks while the VPN runs. */
    private inner class EnginePlatform : Platform {
        override fun protect(fd: Long): Boolean = this@ProxyVpnService.protect(fd.toInt())
        override fun log(level: Long, tag: String?, message: String?) = logFromGo(level, tag, message)
        override fun notice(kind: String?, message: String?) {
            if (kind == null) return
            main.post { controller.onEngineNotice(kind, message.orEmpty()) }
        }
    }

    companion object {
        const val ACTION_CONNECT = "com.json2proxy.android.CONNECT"
        const val ACTION_DISCONNECT = "com.json2proxy.android.DISCONNECT"
        const val EXTRA_CONFIG = "engine_config"
        private const val TAG = "ProxyVpnService"
        private const val CHANNEL_ID = "vpn"
        private const val NOTIFICATION_ID = 1
        private const val MTU = 1500
    }
}
