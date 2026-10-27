package com.json2proxy.android.vpn

import android.content.Context
import android.content.Intent
import com.json2proxy.shared.VpnBackend

/**
 * Starts and stops [ProxyVpnService]. The caller must already hold the VPN
 * permission (VpnService.prepare returned null).
 */
class AndroidVpnBackend(private val context: Context) : VpnBackend {
    override fun start(engineConfigJson: String) {
        val intent = Intent(context, ProxyVpnService::class.java)
            .setAction(ProxyVpnService.ACTION_CONNECT)
            .putExtra(ProxyVpnService.EXTRA_CONFIG, engineConfigJson)
        context.startForegroundService(intent)
    }

    override fun stop() {
        val intent = Intent(context, ProxyVpnService::class.java).setAction(ProxyVpnService.ACTION_DISCONNECT)
        context.startService(intent)
    }
}
