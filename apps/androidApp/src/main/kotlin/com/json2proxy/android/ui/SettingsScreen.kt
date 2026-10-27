package com.json2proxy.android.ui

import android.content.Intent
import android.provider.Settings
import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph
import com.json2proxy.shared.AppSettings
import com.json2proxy.shared.ModeOptions
import com.json2proxy.shared.RoutingMode
import com.json2proxy.shared.TrafficUnit
import com.json2proxy.shared.normalizeAscii

@Composable
internal fun SettingsScreen(graph: AppGraph, onBack: () -> Unit) {
    val context = LocalContext.current
    val initial = remember { graph.settings.current }
    var mode by remember { mutableStateOf(initial.routingMode) }
    var rules by remember { mutableStateOf(initial.customRules) }
    // Each routing mode keeps its own DNS options; switching modes only changes which ones are shown.
    var modeOptions by remember { mutableStateOf(RoutingMode.entries.associateWith { initial.options(it) }) }
    val options = modeOptions.getValue(mode)
    val editOptions: (ModeOptions) -> Unit = { modeOptions = modeOptions + (mode to it) }
    var killSwitch by remember { mutableStateOf(initial.killSwitch) }
    var trafficUnit by remember { mutableStateOf(initial.trafficUnit) }
    var rulesError by remember { mutableStateOf<String?>(null) }

    val save: () -> Unit = {
        rulesError = if (mode == RoutingMode.Custom) graph.controller.validateRules(rules) else null
        if (rulesError == null) {
            graph.settings.update(
                initial.copy(
                    routingMode = mode,
                    customRules = rules,
                    modeOptions = modeOptions.entries.associate { (key, value) ->
                        key.wire to ModeOptions(normalizeAscii(value.dohUrl), normalizeAscii(value.bootstrap))
                    },
                    killSwitch = killSwitch,
                    trafficUnit = trafficUnit,
                ),
            )
            Toast.makeText(context, context.getString(R.string.settings_saved), Toast.LENGTH_SHORT).show()
            onBack()
        }
    }

    ScreenScaffold(
        title = stringResource(R.string.settings_title),
        onBack = onBack,
        actions = { ToolbarIcon(R.drawable.ic_check, stringResource(R.string.action_save), onClick = save) },
    ) { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            SectionTitle(stringResource(R.string.section_routing))
            Choices(RoutingMode.entries, mode, label = { it.title() }) { mode = it }
            Footnote(mode.footnote())
            if (mode == RoutingMode.Custom) {
                OutlinedTextField(
                    value = rules, onValueChange = { rules = it }, label = { Text(stringResource(R.string.field_rules)) },
                    minLines = 6, modifier = Modifier.fillMaxWidth(),
                    keyboardOptions = TechnicalKeyboard,
                    textStyle = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace),
                    isError = rulesError != null,
                    supportingText = rulesError?.let { error -> { Text(stringResource(R.string.rules_invalid, error)) } },
                )
            }

            SectionTitle(stringResource(R.string.section_dns))
            OutlinedTextField(
                value = options.dohUrl, onValueChange = { editOptions(options.copy(dohUrl = it)) }, label = { Text(stringResource(R.string.field_doh)) },
                placeholder = { Text(mode.defaultDoH) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
                keyboardOptions = TechnicalKeyboard.copy(keyboardType = KeyboardType.Uri),
            )
            OutlinedTextField(
                value = options.bootstrap, onValueChange = { editOptions(options.copy(bootstrap = it)) }, label = { Text(stringResource(R.string.field_bootstrap)) },
                placeholder = { Text(AppSettings.DEFAULT_BOOTSTRAP) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
                keyboardOptions = TechnicalKeyboard,
            )
            Footnote(stringResource(R.string.dns_footnote))

            SectionTitle(stringResource(R.string.section_protection))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(stringResource(R.string.field_kill_switch), modifier = Modifier.weight(1f))
                Switch(checked = killSwitch, onCheckedChange = { killSwitch = it })
            }
            Footnote(stringResource(R.string.kill_switch_footnote))
            OutlinedButton(onClick = { context.startActivity(Intent(Settings.ACTION_VPN_SETTINGS)) }) {
                Text(stringResource(R.string.action_system_vpn_settings))
            }
            Footnote(stringResource(R.string.always_on_footnote))

            SectionTitle(stringResource(R.string.section_traffic_unit))
            Choices(TrafficUnit.entries, trafficUnit, label = { it.title() }) { trafficUnit = it }
        }
    }
}

@Composable
private fun Footnote(text: String) {
    Text(text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

@Composable
private fun TrafficUnit.title(): String = stringResource(
    when (this) {
        TrafficUnit.Bytes -> R.string.traffic_unit_bytes
        TrafficUnit.Bits -> R.string.traffic_unit_bits
    },
)
