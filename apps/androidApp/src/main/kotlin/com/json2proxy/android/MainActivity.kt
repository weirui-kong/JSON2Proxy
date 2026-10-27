package com.json2proxy.android

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.json2proxy.android.ui.JsonProxyUi

class MainActivity : ComponentActivity() {
    /** A json2proxy:// link waiting to be imported. */
    private var incomingLink by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        // Recreating the activity redelivers the same intent; import it only once.
        if (savedInstanceState == null) incomingLink = intent.linkOrNull()
        val graph = (application as JsonProxyApp).graph
        setContent {
            MaterialTheme {
                JsonProxyUi(graph, incomingLink = incomingLink, onLinkHandled = { incomingLink = null })
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        intent.linkOrNull()?.let { incomingLink = it }
    }

    private fun Intent.linkOrNull(): String? = if (action == Intent.ACTION_VIEW) dataString else null
}
