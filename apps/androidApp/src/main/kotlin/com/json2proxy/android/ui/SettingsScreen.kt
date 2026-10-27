package com.json2proxy.android.ui

import android.content.Intent
import android.provider.Settings
import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
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
internal fun SettingsScreen(graph: AppGraph, onBack: () -> Unit, onLicenses: () -> Unit) {
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

    TerminalPage(
        header = {
            PageHeader(
                onBack = onBack,
                actions = { PrimaryButton(stringResource(R.string.action_save), onClick = save) },
            ) { PathTitle(stringResource(R.string.path_settings)) }
        },
        bottom = {
            // A small, quiet entry at the very end of the page.
            SecondaryButton(
                stringResource(R.string.settings_licenses),
                onClick = onLicenses,
                color = Colors.textSecondary,
                modifier = Modifier.align(Alignment.CenterHorizontally),
            )
        },
    ) {
        Section(stringResource(R.string.section_routing)) {
            ChoiceList(RoutingMode.entries, mode, title = { it.title() }, detail = { it.wire }) { mode = it }
            if (mode == RoutingMode.Custom) {
                TerminalTextField(
                    value = rules,
                    onValueChange = { rules = it },
                    label = stringResource(R.string.field_rules),
                    error = rulesError?.let { stringResource(R.string.rules_invalid, it) },
                    keyboardOptions = TechnicalKeyboard,
                    minLines = 6,
                    textStyle = Type.small,
                )
            }
            Note(mode.footnote())
        }

        Section(stringResource(R.string.section_dns)) {
            TerminalTextField(
                value = options.dohUrl,
                onValueChange = { editOptions(options.copy(dohUrl = it)) },
                label = stringResource(R.string.field_doh),
                placeholder = mode.defaultDoH,
                keyboardOptions = TechnicalKeyboard.copy(keyboardType = KeyboardType.Uri),
            )
            TerminalTextField(
                value = options.bootstrap,
                onValueChange = { editOptions(options.copy(bootstrap = it)) },
                label = stringResource(R.string.field_bootstrap),
                placeholder = AppSettings.DEFAULT_BOOTSTRAP,
                keyboardOptions = TechnicalKeyboard,
            )
            Note(stringResource(R.string.dns_footnote))
        }

        Section(stringResource(R.string.section_protection)) {
            SwitchRow(stringResource(R.string.field_kill_switch), killSwitch) { killSwitch = it }
            Note(stringResource(R.string.kill_switch_footnote))
            SecondaryButton(
                stringResource(R.string.action_system_vpn_settings) + " " + stringResource(R.string.glyph_external),
                onClick = { context.startActivity(Intent(Settings.ACTION_VPN_SETTINGS)) },
                modifier = Modifier.fillMaxWidth(),
            )
            Note(stringResource(R.string.always_on_footnote))
        }

        Section(stringResource(R.string.section_traffic_unit)) {
            ChoiceList(TrafficUnit.entries, trafficUnit, title = { it.title() }) { trafficUnit = it }
        }
    }
}

/** `# Title` followed by its controls. */
@Composable
internal fun Section(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        SectionTitle(title)
        content()
    }
}

@Composable
private fun TrafficUnit.title(): String = stringResource(
    when (this) {
        TrafficUnit.Bytes -> R.string.traffic_unit_bytes
        TrafficUnit.Bits -> R.string.traffic_unit_bits
    },
)
