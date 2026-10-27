package com.json2proxy.android.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import com.json2proxy.android.R
import com.json2proxy.shared.Palette

/** The shared color card as Compose colors; screens take every color from here. */
internal object Colors {
    val canvas = Color(Palette.canvas)
    val surface = Color(Palette.surface)
    val surfaceSelected = Color(Palette.surfaceSelected)
    val border = Color(Palette.border)
    val control = Color(Palette.control)
    val gridline = Color(Palette.gridline)
    val textPrimary = Color(Palette.textPrimary)
    val textSecondary = Color(Palette.textSecondary)
    val accent = Color(Palette.accent)
    val onAccent = Color(Palette.onAccent)
    val danger = Color(Palette.danger)
    val warning = Color(Palette.warning)
    val syntaxKey = Color(Palette.syntaxKey)
    val syntaxString = Color(Palette.syntaxString)
    val syntaxNumber = Color(Palette.syntaxNumber)
    val infoBackground = Color(Palette.infoBackground)
    val infoBorder = Color(Palette.infoBorder)
    val infoText = Color(Palette.infoText)
    val warningBackground = Color(Palette.warningBackground)
    val warningBorder = Color(Palette.warningBorder)
    val warningText = Color(Palette.warningText)
    val dangerBackground = Color(Palette.dangerBackground)
    val dangerText = Color(Palette.dangerText)
}

/** JetBrains Mono NL (SIL OFL 1.1); Chinese falls back to the system font. */
internal val Mono = FontFamily(
    Font(R.font.jetbrains_mono_regular, FontWeight.Normal),
    Font(R.font.jetbrains_mono_medium, FontWeight.Medium),
    Font(R.font.jetbrains_mono_bold, FontWeight.Bold),
)

/** Type scale of the terminal UI spec. */
internal object Type {
    val title = TextStyle(fontFamily = Mono, fontWeight = FontWeight.Bold, fontSize = 20.sp, lineHeight = 26.sp)
    val section = TextStyle(fontFamily = Mono, fontWeight = FontWeight.Bold, fontSize = 15.sp, lineHeight = 20.sp)
    val body = TextStyle(fontFamily = Mono, fontSize = 15.sp, lineHeight = 21.sp)
    val bodyMedium = body.copy(fontWeight = FontWeight.Medium)
    val code = TextStyle(fontFamily = Mono, fontSize = 14.sp, lineHeight = 24.sp)
    val stat = TextStyle(fontFamily = Mono, fontWeight = FontWeight.Bold, fontSize = 17.sp, lineHeight = 22.sp)
    val small = TextStyle(fontFamily = Mono, fontSize = 13.sp, lineHeight = 19.sp)
    val label = TextStyle(fontFamily = Mono, fontSize = 12.sp, lineHeight = 18.sp)
    val caption = TextStyle(fontFamily = Mono, fontSize = 11.sp, lineHeight = 15.sp)
}

private val colorScheme = darkColorScheme(
    primary = Colors.accent,
    onPrimary = Colors.onAccent,
    secondary = Colors.accent,
    onSecondary = Colors.onAccent,
    background = Colors.canvas,
    onBackground = Colors.textPrimary,
    surface = Colors.surface,
    onSurface = Colors.textPrimary,
    surfaceVariant = Colors.surface,
    onSurfaceVariant = Colors.textSecondary,
    surfaceContainerLowest = Colors.canvas,
    surfaceContainerLow = Colors.surface,
    surfaceContainer = Colors.surface,
    surfaceContainerHigh = Colors.surface,
    surfaceContainerHighest = Colors.surface,
    inverseSurface = Colors.textPrimary,
    inverseOnSurface = Colors.canvas,
    outline = Colors.control,
    outlineVariant = Colors.border,
    error = Colors.danger,
    onError = Colors.onAccent,
)

private val typography = Typography().let { base ->
    fun TextStyle.mono() = copy(fontFamily = Mono)
    Typography(
        displayLarge = base.displayLarge.mono(),
        displayMedium = base.displayMedium.mono(),
        displaySmall = base.displaySmall.mono(),
        headlineLarge = base.headlineLarge.mono(),
        headlineMedium = base.headlineMedium.mono(),
        headlineSmall = base.headlineSmall.mono(),
        titleLarge = base.titleLarge.mono(),
        titleMedium = base.titleMedium.mono(),
        titleSmall = base.titleSmall.mono(),
        bodyLarge = base.bodyLarge.mono(),
        bodyMedium = base.bodyMedium.mono(),
        bodySmall = base.bodySmall.mono(),
        labelLarge = base.labelLarge.mono(),
        labelMedium = base.labelMedium.mono(),
        labelSmall = base.labelSmall.mono(),
    )
}

/** Dark only: system components such as menus and dialogs also draw from the color card. */
@Composable
internal fun JsonProxyTheme(content: @Composable () -> Unit) {
    MaterialTheme(colorScheme = colorScheme, typography = typography, content = content)
}
