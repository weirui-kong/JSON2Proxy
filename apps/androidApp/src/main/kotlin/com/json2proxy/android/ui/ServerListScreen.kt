package com.json2proxy.android.ui

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph
import com.json2proxy.shared.Latency
import com.json2proxy.shared.PingState
import com.json2proxy.shared.ServerLinks
import com.json2proxy.shared.ServerProfile
import com.json2proxy.shared.ServerSummary

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

    TerminalPage(
        header = { PageHeader(onBack = onBack) { PathTitle(stringResource(R.string.path_servers)) } },
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            SecondaryButton(stringResource(R.string.action_scan), onClick = onScan, modifier = Modifier.weight(1f))
            SecondaryButton(
                stringResource(R.string.action_test_all),
                onClick = { graph.controller.pingAll() },
                enabled = profiles.isNotEmpty(),
                modifier = Modifier.weight(1f),
            )
            PrimaryButton(
                stringResource(R.string.glyph_add, stringResource(R.string.action_add)),
                onClick = { onEdit(null) },
                modifier = Modifier.weight(1f),
            )
        }
        Summary(ServerSummary.of(profiles, pings))
        if (profiles.isEmpty()) {
            Note(stringResource(R.string.servers_empty))
        } else {
            TerminalCard {
                profiles.forEachIndexed { index, profile ->
                    if (index > 0) CardDivider()
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
                }
            }
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Note(stringResource(R.string.glyph_selected) + " " + stringResource(R.string.servers_note_selected))
                Note(stringResource(R.string.servers_note_latency))
            }
        }
    }
}

/** Total servers, the best latency and how many failed; `—` until there are results. */
@Composable
private fun Summary(summary: ServerSummary) {
    val none = stringResource(R.string.glyph_none)
    TerminalCard {
        Row(Modifier.height(IntrinsicSize.Min)) {
            SummaryCell(stringResource(R.string.summary_total), summary.total.toString(), Colors.textPrimary)
            Box(Modifier.width(1.dp).fillMaxHeight().background(Colors.border))
            SummaryCell(
                stringResource(R.string.summary_best),
                summary.bestMillis?.let {
                    if (it < 1) stringResource(R.string.ping_under_1ms) else stringResource(R.string.ping_ms, it.toInt())
                } ?: none,
                if (summary.bestMillis != null) Colors.accent else Colors.textSecondary,
            )
            Box(Modifier.width(1.dp).fillMaxHeight().background(Colors.border))
            SummaryCell(
                stringResource(R.string.summary_failed),
                summary.failed?.toString() ?: none,
                if ((summary.failed ?: 0) > 0) Colors.danger else Colors.textSecondary,
            )
        }
    }
}

@Composable
private fun androidx.compose.foundation.layout.RowScope.SummaryCell(label: String, value: String, color: Color) {
    Column(
        Modifier.weight(1f).padding(horizontal = 12.dp, vertical = 10.dp).semantics(mergeDescendants = true) {},
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(label, style = Type.caption, color = Colors.textSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
        Text(value, style = Type.stat, color = color, maxLines = 1)
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
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(if (selected) Colors.surfaceSelected else Color.Transparent)
            .clickable(onClick = onSelect)
            .semantics { this.selected = selected }
            .padding(start = 14.dp, top = 12.dp, bottom = 12.dp, end = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Box(Modifier.width(14.dp)) {
            if (selected) Glyph(stringResource(R.string.glyph_selected), Colors.accent, Type.body.copy(fontWeight = FontWeight.Bold))
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    profile.name,
                    style = Type.bodyMedium,
                    color = Colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                if (ping != null) {
                    Text(ping.label(), style = Type.small, color = Latency.grade(ping).color(), maxLines = 1)
                }
            }
            Text(profile.server, style = Type.label, color = Colors.textSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (ping != null) LatencyBar(ping)
        }
        Box {
            IconButton(onClick = { menuOpen = true }) {
                Icon(
                    painterResource(R.drawable.ic_more_vert),
                    contentDescription = stringResource(R.string.action_more),
                    tint = Colors.textSecondary,
                    modifier = Modifier.size(20.dp),
                )
            }
            DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                MenuItem(R.string.action_edit) { menuOpen = false; onEdit() }
                MenuItem(R.string.action_test_latency) { menuOpen = false; onPing() }
                MenuItem(R.string.action_show_qr) { menuOpen = false; onShowQr() }
                MenuItem(R.string.action_delete, Colors.danger) { menuOpen = false; onDelete() }
            }
        }
    }
}

@Composable
private fun MenuItem(label: Int, color: Color = Colors.textPrimary, onClick: () -> Unit) {
    DropdownMenuItem(
        text = { Text(stringResource(label), style = Type.body, color = color) },
        onClick = onClick,
        modifier = Modifier.padding(horizontal = 4.dp),
    )
}
