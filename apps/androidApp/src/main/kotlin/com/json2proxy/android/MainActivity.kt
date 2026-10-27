package com.json2proxy.android

import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.json2proxy.shared.ConnectionState
import com.json2proxy.shared.NoticeLevel
import com.json2proxy.shared.ProxyController
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    private val controller = ProxyController()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            MaterialTheme {
                ConnectionScreen(controller)
            }
        }
    }
}

@Composable
private fun ConnectionScreen(controller: ProxyController) {
    val state by controller.state.collectAsStateWithLifecycle()
    var server by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    LaunchedEffect(controller) {
        controller.notices.collect { notice ->
            val duration = when (notice.level) {
                NoticeLevel.Info -> Toast.LENGTH_SHORT
                NoticeLevel.Error -> Toast.LENGTH_LONG
            }
            Toast.makeText(context, notice.message, duration).show()
        }
    }

    Scaffold { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text("JSON2Proxy", style = MaterialTheme.typography.headlineMedium)
            Text(controller.platform, style = MaterialTheme.typography.bodySmall)
            OutlinedTextField(
                value = server,
                onValueChange = { server = it },
                label = { Text("Server") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Text(state.label())
            when (state) {
                is ConnectionState.Connected, is ConnectionState.Connecting ->
                    Button(onClick = controller::disconnect) { Text("Disconnect") }
                else ->
                    Button(onClick = { scope.launch { controller.connect(server) } }) { Text("Connect") }
            }
        }
    }
}

private fun ConnectionState.label(): String = when (this) {
    ConnectionState.Disconnected -> "Disconnected"
    is ConnectionState.Connecting -> "Connecting to $server…"
    is ConnectionState.Connected -> "Connected to $server"
    is ConnectionState.Failed -> "Failed: $message"
}
