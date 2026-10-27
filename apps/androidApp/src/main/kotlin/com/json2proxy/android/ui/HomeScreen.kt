package com.json2proxy.android.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph

@Composable
internal fun HomeScreen(
    graph: AppGraph,
    onConnect: () -> Unit,
    onServers: () -> Unit,
    onSettings: () -> Unit,
) {
    val state by graph.controller.state.collectAsStateWithLifecycle()
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
