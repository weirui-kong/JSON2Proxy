package com.json2proxy.android

import android.app.Application
import com.json2proxy.android.platform.AppPlatform
import com.json2proxy.android.platform.GoCoreBridge
import com.json2proxy.android.platform.KeystoreSecretStore
import com.json2proxy.android.platform.PrefsStore
import com.json2proxy.android.vpn.AndroidVpnBackend
import com.json2proxy.core.mobile.Mobile
import com.json2proxy.shared.AppGraph
import java.io.File

class JsonProxyApp : Application() {
    lateinit var graph: AppGraph
        private set

    override fun onCreate() {
        super.onCreate()
        // The VPN service replaces this while it runs so engine sockets are protected.
        Mobile.setPlatform(AppPlatform)
        graph = AppGraph(
            store = PrefsStore(this),
            secrets = KeystoreSecretStore(this),
            core = GoCoreBridge,
            backend = AndroidVpnBackend(this),
            // The engine runs in the app process; Android does not cap it like iOS.
            memoryLimitMb = 0,
            totalsFile = File(filesDir, "traffic-totals.json").path,
        )
    }
}
