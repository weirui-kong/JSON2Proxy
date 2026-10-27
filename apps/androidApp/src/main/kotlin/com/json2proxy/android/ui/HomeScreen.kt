package com.json2proxy.android.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.border
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph
import com.json2proxy.shared.ConnectionState
import com.json2proxy.shared.Latency
import com.json2proxy.shared.RoutingMode
import com.json2proxy.shared.TrafficFormat
import com.json2proxy.shared.TrafficPoint
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
    val history by graph.controller.trafficHistory.collectAsStateWithLifecycle()
    val profiles by graph.profiles.profiles.collectAsStateWithLifecycle()
    val selectedId by graph.profiles.selectedId.collectAsStateWithLifecycle()
    val settings by graph.settings.settings.collectAsStateWithLifecycle()
    val pings by graph.controller.pings.collectAsStateWithLifecycle()
    val selected = profiles.firstOrNull { it.id == selectedId }

    TerminalPage(
        header = {
            PageHeader(
                onBack = null,
                actions = { SquareIconButton(R.drawable.ic_tune, stringResource(R.string.home_settings), onClick = onSettings) },
            ) { PromptTitle() }
        },
        bottom = {
            TerminalCard {
                NavRow(
                    label = stringResource(R.string.home_server),
                    title = selected?.name ?: stringResource(R.string.home_add_server),
                    subtitle = selected?.server,
                    trailing = selected?.let { pings[it.server] }?.let { ping -> ping.label() to Latency.grade(ping).color() },
                    onClick = onServers,
                )
                CardDivider()
                NavRow(
                    label = stringResource(R.string.section_routing),
                    title = settings.routingMode.title(),
                    onClick = onSettings,
                )
                CardDivider()
                NavRow(
                    label = stringResource(R.string.field_kill_switch),
                    title = null,
                    trailing = if (settings.killSwitch) {
                        stringResource(R.string.kill_switch_on) to Colors.accent
                    } else {
                        stringResource(R.string.kill_switch_off) to Colors.textSecondary
                    },
                    onClick = onSettings,
                )
            }
        },
    ) {
        SessionCard(
            state = state,
            server = selected?.name,
            routing = settings.routingMode,
            traffic = traffic,
            history = history,
            unit = settings.trafficUnit,
        )
        if (state is ConnectionState.Failed) Banner(BannerKind.Danger, state.label())
        ConnectButton(state, enabled = selected != null, onConnect = onConnect, onDisconnect = { graph.controller.disconnect() })
    }
}

/** `session.json`: the connection as a small JSON object, then live traffic. */
@Composable
private fun SessionCard(
    state: ConnectionState,
    server: String?,
    routing: RoutingMode,
    traffic: TrafficStats?,
    history: List<TrafficPoint>,
    unit: TrafficUnit,
) {
    val live = state is ConnectionState.Connected
    val (stateValue, stateColor) = when (state) {
        is ConnectionState.Connected -> stringResource(R.string.json_state_connected) to Colors.accent
        is ConnectionState.Connecting -> stringResource(R.string.json_state_connecting) to Colors.warning
        ConnectionState.Disconnecting -> stringResource(R.string.json_state_disconnecting) to Colors.warning
        is ConnectionState.Failed -> stringResource(R.string.json_state_failed) to Colors.danger
        ConnectionState.Disconnected -> stringResource(R.string.json_state_disconnected) to Colors.textSecondary
    }
    val description = stringResource(R.string.home_session_description, state.label(), routing.title())
    val keyState = stringResource(R.string.json_key_state)
    val keyServer = stringResource(R.string.json_key_server)
    val keyRouting = stringResource(R.string.json_key_routing)
    val nullValue = stringResource(R.string.json_null)

    TerminalCard {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(stringResource(R.string.session_file), style = Type.label, color = Colors.textSecondary, modifier = Modifier.weight(1f))
            val dot = if (live) Colors.accent else Colors.textSecondary
            Box(Modifier.size(7.dp).background(dot, CircleShape))
            Text(
                stringResource(if (live) R.string.session_live else R.string.session_idle),
                style = Type.label,
                color = dot,
                modifier = Modifier.padding(start = 6.dp),
            )
        }
        CardDivider()
        val json = buildAnnotatedString {
            fun key(name: String) {
                append("  ")
                appendColored("\"$name\"", Colors.syntaxKey)
                appendColored(": ", Colors.textSecondary)
            }
            appendColored("{\n", Colors.textSecondary)
            key(keyState)
            appendColored("\"$stateValue\"", stateColor)
            appendColored(",\n", Colors.textSecondary)
            key(keyServer)
            if (server != null) appendColored("\"$server\"", Colors.syntaxString) else appendColored(nullValue, Colors.syntaxNumber)
            appendColored(",\n", Colors.textSecondary)
            key(keyRouting)
            appendColored("\"${routing.wire}\"", Colors.syntaxString)
            appendColored("\n}", Colors.textSecondary)
        }
        Text(
            json,
            style = Type.code,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 12.dp)
                .clearAndSetSemantics { contentDescription = description },
        )
        CardDivider()
        if (live && traffic != null) {
            TrafficPanel(traffic, history, unit)
        } else {
            Note(stringResource(R.string.traffic_waiting), Modifier.padding(14.dp))
        }
    }
}

@Composable
private fun TrafficPanel(traffic: TrafficStats, history: List<TrafficPoint>, unit: TrafficUnit) {
    val download = TrafficFormat.rate(traffic.downloadSpeed, unit)
    val upload = TrafficFormat.rate(traffic.uploadSpeed, unit)
    Column(Modifier.padding(start = 14.dp, end = 14.dp, top = 12.dp, bottom = 14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Note(stringResource(R.string.traffic_legend), Modifier.weight(1f))
            Legend(Colors.syntaxKey, stringResource(R.string.traffic_download))
            Legend(Colors.syntaxNumber, stringResource(R.string.traffic_upload), Modifier.padding(start = 10.dp))
        }
        TrafficChart(history, stringResource(R.string.traffic_chart_description, download, upload))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            StatCell(stringResource(R.string.glyph_download), stringResource(R.string.traffic_download), download, Colors.syntaxKey, Modifier.weight(1f))
            StatCell(stringResource(R.string.glyph_upload), stringResource(R.string.traffic_upload), upload, Colors.syntaxNumber, Modifier.weight(1f))
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            StatCell(null, stringResource(R.string.stat_tcp), traffic.tcpConnections.toString(), Colors.textPrimary, Modifier.weight(1f))
            StatCell(null, stringResource(R.string.stat_udp), traffic.udpConnections.toString(), Colors.textPrimary, Modifier.weight(1f))
        }
    }
}

@Composable
private fun Legend(color: Color, label: String, modifier: Modifier = Modifier) {
    Row(modifier, verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        LegendMark(color)
        Text(label, style = Type.caption, color = Colors.textSecondary)
    }
}

/** Label over a value; a rate's unit is smaller and dimmed. */
@Composable
private fun StatCell(glyph: String?, label: String, value: String, color: Color, modifier: Modifier) {
    val (number, unit) = splitRate(value)
    Column(
        modifier = modifier
            .border(1.dp, Colors.border, ControlShape)
            .padding(horizontal = 10.dp, vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Row {
            if (glyph != null) Glyph("$glyph ", style = Type.caption)
            Text(label, style = Type.caption, color = Colors.textSecondary)
        }
        Text(
            buildAnnotatedString {
                appendColored(number, color)
                if (unit.isNotEmpty()) appendColored(" $unit", Colors.textSecondary)
            },
            style = Type.stat,
            maxLines = 1,
        )
    }
}

@Composable
private fun ConnectButton(state: ConnectionState, enabled: Boolean, onConnect: () -> Unit, onDisconnect: () -> Unit) {
    // The decorated labels read badly aloud; screen readers get the plain action.
    val connect = stringResource(R.string.action_connect)
    val disconnect = stringResource(R.string.action_disconnect)
    if (state.isActive()) {
        SecondaryButton(
            text = stringResource(R.string.glyph_disconnect, disconnect),
            onClick = onDisconnect,
            enabled = state != ConnectionState.Disconnecting,
            color = Colors.danger,
            borderColor = Colors.danger,
            containerColor = Color.Transparent,
            height = 52.dp,
            modifier = Modifier.fillMaxWidth().semantics { contentDescription = disconnect },
        )
    } else {
        PrimaryButton(
            text = stringResource(R.string.glyph_connect, connect),
            onClick = onConnect,
            enabled = enabled,
            height = 52.dp,
            modifier = Modifier.fillMaxWidth().semantics { contentDescription = connect },
        )
    }
}

/** A row of the home card: a short label, a value, and an optional colored status. */
@Composable
private fun NavRow(
    label: String,
    title: String?,
    subtitle: String? = null,
    trailing: Pair<String, Color>? = null,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 52.dp)
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(label, style = Type.small, color = Colors.textSecondary, modifier = Modifier.width(96.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            if (title != null) {
                Text(title, style = Type.bodyMedium, color = Colors.textPrimary, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            if (subtitle != null) {
                Text(subtitle, style = Type.label, color = Colors.textSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
        trailing?.let { (text, color) ->
            Text(text, style = Type.small.copy(fontWeight = FontWeight.Medium), color = color, maxLines = 1)
        }
        Glyph(stringResource(R.string.glyph_next))
    }
}
