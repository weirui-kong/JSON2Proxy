package com.json2proxy.android

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.SystemBarStyle
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import android.graphics.drawable.ColorDrawable
import com.json2proxy.android.ui.JsonProxyTheme
import com.json2proxy.android.ui.JsonProxyUi
import com.json2proxy.shared.Palette

class MainActivity : ComponentActivity() {
    /** A json2proxy:// link waiting to be imported. */
    private var incomingLink by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Dark only: light system bar icons over the canvas color.
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
        )
        window.setBackgroundDrawable(ColorDrawable(Palette.canvas.toInt()))
        // Recreating the activity redelivers the same intent; import it only once.
        if (savedInstanceState == null) incomingLink = intent.newLinkOrNull()
        val graph = (application as JsonProxyApp).graph
        setContent {
            JsonProxyTheme {
                JsonProxyUi(graph, incomingLink = incomingLink, onLinkHandled = { incomingLink = null })
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        intent.newLinkOrNull()?.let { incomingLink = it }
    }

    /**
     * The link forwarded by [LinkActivity], unless its id was handled before:
     * the system relaunches a task with its original intent, for example from
     * Recents or after an update.
     */
    private fun Intent.newLinkOrNull(): String? {
        val link = getStringExtra(EXTRA_LINK) ?: return null
        val id = getStringExtra(EXTRA_LINK_ID) ?: return null
        val prefs = getSharedPreferences(PREFS, MODE_PRIVATE)
        if (prefs.getString(KEY_HANDLED_LINK_ID, null) == id) return null
        prefs.edit().putString(KEY_HANDLED_LINK_ID, id).apply()
        return link
    }

    companion object {
        const val EXTRA_LINK = "com.json2proxy.app.LINK"
        const val EXTRA_LINK_ID = "com.json2proxy.app.LINK_ID"
        private const val PREFS = "json2proxy_ui"
        private const val KEY_HANDLED_LINK_ID = "handled_link_id"
    }
}
