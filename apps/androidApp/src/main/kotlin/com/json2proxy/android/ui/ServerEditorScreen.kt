package com.json2proxy.android.ui

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
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
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.json2proxy.android.R
import com.json2proxy.shared.AppGraph
import com.json2proxy.shared.ObfsProviders
import com.json2proxy.shared.PerformanceModes
import com.json2proxy.shared.ProfileProblem
import com.json2proxy.shared.ProfileValidator
import com.json2proxy.shared.ServerLink
import com.json2proxy.shared.ServerProfile
import com.json2proxy.shared.normalizeAscii

@Composable
internal fun ServerEditorScreen(graph: AppGraph, profileId: String?, imported: ServerLink?, onDone: () -> Unit) {
    val context = LocalContext.current
    val existing = remember(profileId) { graph.profiles.profiles.value.firstOrNull { it.id == profileId } }
    // An imported link prefills a new server; the user reviews it before saving.
    val initial = existing ?: imported?.profile
    var name by remember { mutableStateOf(initial?.name.orEmpty()) }
    var server by remember { mutableStateOf(initial?.server.orEmpty()) }
    var clientId by remember { mutableStateOf(initial?.clientId.orEmpty()) }
    var secret by remember { mutableStateOf(imported?.secret.orEmpty()) }
    var obfs by remember { mutableStateOf(initial?.obfsProvider ?: ObfsProviders.NONE) }
    var participation by remember { mutableStateOf((initial?.obfsParticipation ?: 0.0).toString()) }
    var performance by remember { mutableStateOf(initial?.performanceMode ?: PerformanceModes.INTERACTIVE) }
    var udp by remember { mutableStateOf(initial?.udpEnabled ?: true) }
    var problems by remember { mutableStateOf(emptyList<ProfileProblem>()) }

    val save: () -> Unit = {
        val profile = ServerProfile(
            id = existing?.id ?: graph.profiles.generateId(),
            name = name.trim(),
            server = normalizeAscii(server),
            clientId = normalizeAscii(clientId),
            obfsProvider = obfs,
            obfsParticipation = normalizeAscii(participation).toDoubleOrNull() ?: Double.NaN,
            performanceMode = performance,
            udpEnabled = udp,
        )
        // Editing with an empty field keeps the stored secret.
        val newSecret = if (existing != null && secret.isEmpty()) null else normalizeAscii(secret)
        problems = ProfileValidator.validate(profile, newSecret)
        if (problems.isEmpty()) {
            if (graph.profiles.save(profile, newSecret)) {
                onDone()
            } else {
                Toast.makeText(context, context.getString(R.string.secret_store_failed), Toast.LENGTH_LONG).show()
            }
        }
    }

    ScreenScaffold(
        title = stringResource(if (existing == null) R.string.editor_add_title else R.string.editor_edit_title),
        onBack = onDone,
        backIcon = R.drawable.ic_close,
        backLabel = R.string.action_close,
        actions = { ToolbarIcon(R.drawable.ic_check, stringResource(R.string.action_save), onClick = save) },
    ) { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (imported != null) {
                Text(
                    stringResource(R.string.import_review),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(MaterialTheme.colorScheme.secondaryContainer, RoundedCornerShape(12.dp))
                        .padding(12.dp),
                )
            }
            OutlinedTextField(
                value = name, onValueChange = { name = it }, label = { Text(stringResource(R.string.field_name)) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = server, onValueChange = { server = it }, label = { Text(stringResource(R.string.field_server)) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
                keyboardOptions = TechnicalKeyboard.copy(keyboardType = KeyboardType.Uri),
            )
            OutlinedTextField(
                value = clientId, onValueChange = { clientId = it }, label = { Text(stringResource(R.string.field_client_id)) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
                keyboardOptions = TechnicalKeyboard,
            )
            OutlinedTextField(
                value = secret, onValueChange = { secret = it },
                label = { Text(stringResource(if (existing == null) R.string.field_secret else R.string.field_secret_keep)) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = TechnicalKeyboard.copy(keyboardType = KeyboardType.Password),
            )

            SectionTitle(stringResource(R.string.section_obfuscation))
            Choices(ObfsProviders.all, obfs, label = { obfsLabel(it) }) {
                obfs = it
                participation = if (it == ObfsProviders.NONE) "0" else ObfsProviders.DEFAULT_PARTICIPATION.toString()
            }
            OutlinedTextField(
                value = participation, onValueChange = { participation = it }, label = { Text(stringResource(R.string.field_participation)) },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
                keyboardOptions = TechnicalKeyboard.copy(keyboardType = KeyboardType.Decimal),
            )

            SectionTitle(stringResource(R.string.section_transport))
            Choices(PerformanceModes.all, performance, label = { performanceLabel(it) }) { performance = it }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(stringResource(R.string.field_udp), modifier = Modifier.weight(1f))
                Switch(checked = udp, onCheckedChange = { udp = it })
            }

            problems.forEach { Text(it.text(), color = MaterialTheme.colorScheme.error) }
        }
    }
}

/** Addresses, IDs and secrets: no capitalization or autocorrect, ASCII keyboard. */
internal val TechnicalKeyboard = KeyboardOptions(
    capitalization = KeyboardCapitalization.None,
    autoCorrectEnabled = false,
    keyboardType = KeyboardType.Ascii,
)

@Composable
internal fun SectionTitle(text: String) {
    Text(text, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.primary)
}

@Composable
internal fun <T> Choices(options: List<T>, selected: T, label: @Composable (T) -> String, onSelect: (T) -> Unit) {
    Column {
        options.forEach { option ->
            Row(
                modifier = Modifier.fillMaxWidth().clickable { onSelect(option) }.padding(vertical = 2.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RadioButton(selected = option == selected, onClick = { onSelect(option) })
                Text(label(option))
            }
        }
    }
}

/** Provider names are configuration values; only "none" is translated. */
@Composable
private fun obfsLabel(provider: String): String =
    if (provider == ObfsProviders.NONE) stringResource(R.string.obfs_none) else provider

@Composable
private fun performanceLabel(mode: String): String = when (mode) {
    PerformanceModes.INTERACTIVE -> stringResource(R.string.perf_interactive)
    PerformanceModes.THROUGHPUT -> stringResource(R.string.perf_throughput)
    else -> mode
}

@Composable
private fun ProfileProblem.text(): String = stringResource(
    when (this) {
        ProfileProblem.NameRequired -> R.string.problem_name_required
        ProfileProblem.InvalidServer -> R.string.problem_invalid_server
        ProfileProblem.ClientIdRequired -> R.string.problem_client_id_required
        ProfileProblem.SecretNotBase64Url -> R.string.problem_secret_not_base64url
        ProfileProblem.SecretTooShort -> R.string.problem_secret_too_short
        ProfileProblem.UnknownObfsProvider -> R.string.problem_unknown_obfs
        ProfileProblem.InvalidParticipation -> R.string.problem_invalid_participation
        ProfileProblem.UnknownPerformanceMode -> R.string.problem_unknown_performance
    },
)
