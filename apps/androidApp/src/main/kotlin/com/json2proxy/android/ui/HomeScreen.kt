package com.json2proxy.android.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph
import com.json2proxy.shared.TrafficFormat
import com.json2proxy.shared.TrafficStats
import com.json2proxy.shared.TrafficUnit

@Composable
internal fun HomeScreen(
    graph: AppGraph,
    onConnect: () -> Unit,
    onServers: () -> Unit,
    onSettings: () -> Unit,
) {
    val state by graph.controller.state.collectAsStateWithLifecycle()
    val traffic by graph.controller.traffic.collectAsStateWithLifecycle()
    val profiles by graph.profiles.profiles.collectAsStateWithLifecycle()
    val selectedId by graph.profiles.selectedId.collectAsStateWithLifecycle()
    val settings by graph.settings.settings.collectAsStateWithLifecycle()
    val selected = profiles.firstOrNull { it.id == selectedId }

    ScreenScaffold(title = stringResource(R.string.app_name), onBack = null) { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(
                    modifier = Modifier.padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Text(state.label(), style = MaterialTheme.typography.titleMedium)
                    traffic?.let { TrafficSummary(it, settings.trafficUnit) }
                    if (state.isActive()) {
                        OutlinedButton(onClick = { graph.controller.disconnect() }) { Text(stringResource(R.string.action_disconnect)) }
                    } else {
                        Button(onClick = onConnect, enabled = selected != null) { Text(stringResource(R.string.action_connect)) }
                    }
                }
            }
            ListItem(
                headlineContent = { Text(selected?.name ?: stringResource(R.string.home_add_server)) },
                supportingContent = { Text(selected?.server ?: stringResource(R.string.home_no_server_selected)) },
                overlineContent = { Text(stringResource(R.string.home_server)) },
                modifier = Modifier.clickable(onClick = onServers),
            )
            ListItem(
                headlineContent = { Text(stringResource(R.string.home_settings)) },
                supportingContent = { Text(stringResource(R.string.home_routing, settings.routingMode.title())) },
                modifier = Modifier.clickable(onClick = onSettings),
            )
            Text(
                graph.controller.platform,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

/** Overall upload and download speed, and the open TCP connections and UDP flows. */
@Composable
private fun TrafficSummary(traffic: TrafficStats, unit: TrafficUnit) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.CenterVertically) {
            Rate(R.drawable.ic_arrow_upward, stringResource(R.string.traffic_upload), TrafficFormat.rate(traffic.uploadSpeed, unit))
            Rate(R.drawable.ic_arrow_downward, stringResource(R.string.traffic_download), TrafficFormat.rate(traffic.downloadSpeed, unit))
        }
        Text(
            stringResource(R.string.traffic_connections, traffic.tcpConnections, traffic.udpConnections),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun Rate(icon: Int, label: String, value: String) {
    Row(horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.CenterVertically) {
        Icon(painterResource(icon), contentDescription = label, modifier = Modifier.size(16.dp))
        Text(value, style = MaterialTheme.typography.bodyMedium)
    }
}
