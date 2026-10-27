package com.json2proxy.android.ui

import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph
import com.json2proxy.shared.PingState
import com.json2proxy.shared.ServerLinks
import com.json2proxy.shared.ServerProfile

@Composable
internal fun ServerListScreen(
    graph: AppGraph,
    onBack: () -> Unit,
    onEdit: (String?) -> Unit,
    onScan: () -> Unit,
    onShowQr: (ServerProfile, String) -> Unit,
) {
    val context = LocalContext.current
    val profiles by graph.profiles.profiles.collectAsStateWithLifecycle()
    val selectedId by graph.profiles.selectedId.collectAsStateWithLifecycle()
    val pings by graph.controller.pings.collectAsStateWithLifecycle()

    ScreenScaffold(
        title = stringResource(R.string.servers_title),
        onBack = onBack,
        actions = {
            ToolbarIcon(R.drawable.ic_speed, stringResource(R.string.action_test_all), enabled = profiles.isNotEmpty()) {
                graph.controller.pingAll()
            }
            ToolbarIcon(R.drawable.ic_qr_scanner, stringResource(R.string.action_scan), onClick = onScan)
            ToolbarIcon(R.drawable.ic_add, stringResource(R.string.action_add)) { onEdit(null) }
        },
    ) { padding ->
        if (profiles.isEmpty()) {
            Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                Text(stringResource(R.string.servers_empty), color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        } else {
            LazyColumn(Modifier.fillMaxSize().padding(padding)) {
                items(profiles, key = { it.id }) { profile ->
                    ServerRow(
                        profile = profile,
                        selected = profile.id == selectedId,
                        ping = pings[profile.server],
                        onSelect = { graph.profiles.select(profile.id) },
                        onEdit = { onEdit(profile.id) },
                        onPing = { graph.controller.ping(profile.id) },
                        onDelete = { graph.profiles.delete(profile.id) },
                        onShowQr = {
                            val secret = graph.profiles.secret(profile.id)
                            if (secret == null) {
                                Toast.makeText(
                                    context, context.getString(R.string.state_failed_missing_secret, profile.name), Toast.LENGTH_LONG,
                                ).show()
                            } else {
                                onShowQr(profile, ServerLinks.format(profile, secret))
                            }
                        },
                    )
                    HorizontalDivider()
                }
            }
        }
    }
}

@Composable
private fun ServerRow(
    profile: ServerProfile,
    selected: Boolean,
    ping: PingState?,
    onSelect: () -> Unit,
    onEdit: () -> Unit,
    onPing: () -> Unit,
    onDelete: () -> Unit,
    onShowQr: () -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    val moreLabel = stringResource(R.string.action_more)
    ListItem(
        headlineContent = { Text(profile.name) },
        supportingContent = { Text(profile.server) },
        trailingContent = {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                if (ping != null) {
                    Text(
                        ping.label(),
                        style = MaterialTheme.typography.bodySmall,
                        color = if (ping is PingState.Failed) MaterialTheme.colorScheme.error
                        else MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.widthIn(max = 140.dp),
                    )
                }
                if (selected) {
                    Icon(
                        painterResource(R.drawable.ic_check),
                        contentDescription = stringResource(R.string.server_selected),
                        tint = MaterialTheme.colorScheme.primary,
                    )
                }
                Box {
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(painterResource(R.drawable.ic_more_vert), contentDescription = moreLabel)
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(text = { Text(stringResource(R.string.action_edit)) }, onClick = { menuOpen = false; onEdit() })
                        DropdownMenuItem(text = { Text(stringResource(R.string.action_test_latency)) }, onClick = { menuOpen = false; onPing() })
                        DropdownMenuItem(text = { Text(stringResource(R.string.action_show_qr)) }, onClick = { menuOpen = false; onShowQr() })
                        DropdownMenuItem(text = { Text(stringResource(R.string.action_delete)) }, onClick = { menuOpen = false; onDelete() })
                    }
                }
            }
        },
        modifier = Modifier.clickable(onClick = onSelect),
    )
}
