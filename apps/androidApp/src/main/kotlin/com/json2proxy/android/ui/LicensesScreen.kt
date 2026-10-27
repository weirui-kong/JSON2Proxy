package com.json2proxy.android.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.json2proxy.android.R
import com.json2proxy.shared.NoticeCategory
import com.json2proxy.shared.NoticeId
import com.json2proxy.shared.NoticeLicense
import com.json2proxy.shared.NoticePlatform
import com.json2proxy.shared.OpenSourceNotice
import com.json2proxy.shared.OpenSourceNotices

/** Third-party components and data the app uses, grouped by kind, each with its license. */
@Composable
internal fun LicensesScreen(onBack: () -> Unit) {
    val notices = OpenSourceNotices.forPlatform(NoticePlatform.Android)
    TerminalPage(
        header = {
            PageHeader(onBack = onBack) { PathTitle(stringResource(R.string.path_settings), stringResource(R.string.path_licenses)) }
        },
    ) {
        Note(stringResource(R.string.licenses_intro))
        NoticeCategory.entries.forEach { category ->
            val group = notices.filter { it.category == category }
            if (group.isNotEmpty()) {
                Section(category.title()) {
                    TerminalCard {
                        group.forEachIndexed { index, notice ->
                            if (index > 0) CardDivider()
                            NoticeRow(notice)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun NoticeRow(notice: OpenSourceNotice) {
    var expanded by rememberSaveable(notice.id) { mutableStateOf(false) }
    val uriHandler = LocalUriHandler.current
    Column(Modifier.fillMaxWidth().padding(14.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(notice.name, style = Type.bodyMedium, color = Colors.textPrimary, modifier = Modifier.weight(1f))
            Text(notice.license.title(), style = Type.label, color = Colors.accent)
        }
        Text(notice.id.purpose(), style = Type.small, color = Colors.textSecondary)
        Text(notice.copyright, style = Type.label, color = Colors.textSecondary)
        Text(
            notice.url,
            style = Type.label,
            color = Colors.syntaxKey,
            modifier = Modifier
                .heightIn(min = 32.dp)
                .clickable(role = Role.Button) { uriHandler.openUri(notice.url) }
                .padding(vertical = 6.dp),
        )
        val text = notice.text
        if (text != null) {
            Text(
                stringResource(if (expanded) R.string.notice_hide_text else R.string.notice_show_text),
                style = Type.small.copy(fontWeight = FontWeight.Medium),
                color = Colors.textPrimary,
                modifier = Modifier
                    .heightIn(min = 44.dp)
                    .clickable(role = Role.Button) { expanded = !expanded }
                    .padding(vertical = 12.dp),
            )
            if (expanded) {
                Text(
                    text,
                    style = Type.caption.copy(fontSize = 10.sp, lineHeight = 14.sp),
                    color = Colors.textSecondary,
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(Colors.canvas, ControlShape)
                        .padding(12.dp),
                )
            }
        }
    }
}

@Composable
private fun NoticeCategory.title(): String = stringResource(
    when (this) {
        NoticeCategory.Library -> R.string.notice_category_library
        NoticeCategory.Font -> R.string.notice_category_font
        NoticeCategory.Icons -> R.string.notice_category_icons
        NoticeCategory.Data -> R.string.notice_category_data
    },
)

@Composable
private fun NoticeLicense.title(): String = spdx ?: stringResource(R.string.notice_license_public_data)

@Composable
private fun NoticeId.purpose(): String = stringResource(
    when (this) {
        NoticeId.GoRuntime -> R.string.notice_purpose_go_runtime
        NoticeId.GoExtensions -> R.string.notice_purpose_go_extensions
        NoticeId.Gvisor -> R.string.notice_purpose_gvisor
        NoticeId.Btree -> R.string.notice_purpose_btree
        NoticeId.KotlinStdlib -> R.string.notice_purpose_kotlin
        NoticeId.KotlinxCoroutines -> R.string.notice_purpose_coroutines
        NoticeId.KotlinxSerialization -> R.string.notice_purpose_serialization
        NoticeId.Skie -> R.string.notice_purpose_skie
        NoticeId.AndroidX -> R.string.notice_purpose_androidx
        NoticeId.Zxing -> R.string.notice_purpose_zxing
        NoticeId.TerminalUi -> R.string.notice_purpose_terminal_ui
        NoticeId.JetBrainsMono -> R.string.notice_purpose_jetbrains_mono
        NoticeId.MaterialIcons -> R.string.notice_purpose_material_icons
        NoticeId.ApnicStats -> R.string.notice_purpose_apnic
    },
)
