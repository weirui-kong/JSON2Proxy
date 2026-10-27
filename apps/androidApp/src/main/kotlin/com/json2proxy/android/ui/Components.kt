package com.json2proxy.android.ui

import androidx.annotation.DrawableRes
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.systemBars
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.PlainTooltip
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TooltipAnchorPosition
import androidx.compose.material3.TooltipBox
import androidx.compose.material3.TooltipDefaults
import androidx.compose.material3.rememberTooltipState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.error
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.json2proxy.android.R
import com.json2proxy.shared.Latency
import com.json2proxy.shared.LatencyGrade
import com.json2proxy.shared.PingState
import com.json2proxy.shared.TrafficPoint
import kotlin.math.max

internal val CardShape = RoundedCornerShape(10.dp)
internal val ControlShape = RoundedCornerShape(8.dp)
private val FieldShape = RoundedCornerShape(6.dp)

/**
 * A full page: system bar insets, the header, then scrolling content. When the
 * content is shorter than the screen, [bottom] sits at the bottom edge.
 */
@Composable
internal fun TerminalPage(
    header: @Composable () -> Unit,
    scroll: ScrollState = rememberScrollState(),
    bottom: (@Composable ColumnScope.() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    BoxWithConstraints(
        Modifier.fillMaxSize().background(Colors.canvas).windowInsetsPadding(WindowInsets.systemBars),
    ) {
        val viewport = maxHeight
        Column(Modifier.fillMaxSize().verticalScroll(scroll)) {
            Column(
                modifier = Modifier.fillMaxWidth().heightIn(min = viewport).padding(horizontal = 20.dp, vertical = 16.dp),
                verticalArrangement = Arrangement.SpaceBetween,
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(20.dp)) {
                    header()
                    content()
                }
                if (bottom != null) {
                    Column(Modifier.fillMaxWidth().padding(top = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp), content = bottom)
                }
            }
        }
    }
}

/** Header row: optional back button, a title, and actions on the right. */
@Composable
internal fun PageHeader(
    onBack: (() -> Unit)?,
    @DrawableRes backIcon: Int = R.drawable.ic_arrow_back,
    backLabel: Int = R.string.action_back,
    actions: @Composable RowScope.() -> Unit = {},
    title: @Composable () -> Unit,
) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        if (onBack != null) SquareIconButton(backIcon, stringResource(backLabel), onClick = onBack)
        Box(Modifier.weight(1f)) { title() }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically, content = actions)
    }
}

/** `$ json2proxy` with a block cursor. */
@Composable
internal fun PromptTitle() {
    val command = stringResource(R.string.glyph_command)
    Row(
        modifier = Modifier.semantics { heading() },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(stringResource(R.string.glyph_prompt), style = Type.title, color = Colors.accent, modifier = Modifier.clearAndSetSemantics {})
        Text(command, style = Type.title, color = Colors.textPrimary)
        Box(Modifier.size(10.dp, 20.dp).background(Colors.accent))
    }
}

/** `~/servers/edit`: earlier segments are dimmed; read aloud without the punctuation. */
@Composable
internal fun PathTitle(vararg segments: String) {
    val home = stringResource(R.string.glyph_home_dir)
    val separator = stringResource(R.string.glyph_path_separator)
    val text = buildAnnotatedString {
        withStyle(SpanStyle(color = Colors.textSecondary)) {
            append(home)
            segments.dropLast(1).forEach { append(it); append(separator) }
        }
        append(segments.last())
    }
    Text(
        text,
        style = Type.title,
        color = Colors.textPrimary,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
        modifier = Modifier.clearAndSetSemantics {
            heading()
            contentDescription = segments.joinToString(" ")
        },
    )
}

/** 44×44 outlined icon button with a tooltip on long press. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun SquareIconButton(
    @DrawableRes icon: Int,
    label: String,
    enabled: Boolean = true,
    tint: Color = Colors.textPrimary,
    onClick: () -> Unit,
) {
    TooltipBox(
        positionProvider = TooltipDefaults.rememberTooltipPositionProvider(TooltipAnchorPosition.Below),
        tooltip = { PlainTooltip { Text(label) } },
        state = rememberTooltipState(),
    ) {
        Surface(
            onClick = onClick,
            enabled = enabled,
            shape = ControlShape,
            color = Colors.surface,
            border = BorderStroke(1.dp, Colors.border),
            modifier = Modifier.size(44.dp).semantics { contentDescription = label },
        ) {
            Box(contentAlignment = Alignment.Center) {
                Icon(
                    painterResource(icon),
                    contentDescription = null,
                    tint = if (enabled) tint else Colors.textSecondary,
                    modifier = Modifier.size(20.dp),
                )
            }
        }
    }
}

@Composable
internal fun TerminalCard(modifier: Modifier = Modifier, content: @Composable ColumnScope.() -> Unit) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(CardShape)
            .background(Colors.surface)
            .border(1.dp, Colors.border, CardShape),
        content = content,
    )
}

@Composable
internal fun CardDivider() {
    HorizontalDivider(thickness = 1.dp, color = Colors.border)
}

/** `# Title` */
@Composable
internal fun SectionTitle(text: String) {
    Row(Modifier.semantics(mergeDescendants = true) { heading() }) {
        Text(
            stringResource(R.string.glyph_section) + " ",
            style = Type.section,
            color = Colors.accent,
            modifier = Modifier.clearAndSetSemantics {},
        )
        Text(text, style = Type.section, color = Colors.textPrimary)
    }
}

/** `// note`, with the comment marker hidden from screen readers. */
@Composable
internal fun Note(text: String, modifier: Modifier = Modifier, color: Color = Colors.textSecondary) {
    Row(modifier) {
        Text(
            stringResource(R.string.glyph_comment) + " ",
            style = Type.label,
            color = color,
            modifier = Modifier.clearAndSetSemantics {},
        )
        Text(text, style = Type.label, color = color)
    }
}

@Composable
internal fun PrimaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    height: Dp = 44.dp,
) {
    Button(
        onClick = onClick,
        enabled = enabled,
        shape = ControlShape,
        colors = ButtonDefaults.buttonColors(
            containerColor = Colors.accent,
            contentColor = Colors.onAccent,
            disabledContainerColor = Colors.surface,
            disabledContentColor = Colors.textSecondary,
        ),
        contentPadding = PaddingValues(horizontal = 16.dp),
        modifier = modifier.heightIn(min = height),
    ) {
        Text(text, style = Type.bodyMedium.copy(fontWeight = androidx.compose.ui.text.font.FontWeight.Bold), maxLines = 1)
    }
}

@Composable
internal fun SecondaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    color: Color = Colors.textPrimary,
    borderColor: Color = Colors.border,
    containerColor: Color = Colors.surface,
    height: Dp = 44.dp,
) {
    OutlinedButton(
        onClick = onClick,
        enabled = enabled,
        shape = ControlShape,
        border = BorderStroke(1.dp, if (enabled) borderColor else Colors.border),
        colors = ButtonDefaults.outlinedButtonColors(
            containerColor = containerColor,
            contentColor = color,
            disabledContainerColor = containerColor,
            disabledContentColor = Colors.textSecondary,
        ),
        contentPadding = PaddingValues(horizontal = 12.dp),
        modifier = modifier.heightIn(min = height),
    ) {
        Text(text, style = Type.small, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

/** Label above a single-line or multi-line field; errors show below it. */
@Composable
internal fun TerminalTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    placeholder: String? = null,
    error: String? = null,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    minLines: Int = 1,
    textStyle: TextStyle = Type.body.copy(fontSize = 14.sp),
) {
    val interaction = remember { MutableInteractionSource() }
    val focused by interaction.collectIsFocusedAsState()
    val outline = when {
        error != null -> Colors.danger
        focused -> Colors.accent
        else -> Colors.control
    }
    Column(modifier, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            label,
            style = Type.label,
            color = if (error != null) Colors.danger else Colors.textSecondary,
            modifier = Modifier.clearAndSetSemantics {},
        )
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            singleLine = minLines == 1,
            minLines = minLines,
            textStyle = textStyle.copy(color = Colors.textPrimary),
            cursorBrush = SolidColor(Colors.accent),
            keyboardOptions = keyboardOptions,
            visualTransformation = visualTransformation,
            interactionSource = interaction,
            modifier = Modifier
                .fillMaxWidth()
                .semantics {
                    contentDescription = label
                    if (error != null) error(error)
                },
            decorationBox = { field ->
                Box(
                    Modifier
                        .fillMaxWidth()
                        .heightIn(min = 46.dp)
                        .background(Colors.canvas, FieldShape)
                        .border(1.dp, outline, FieldShape)
                        .padding(horizontal = 12.dp, vertical = 12.dp),
                    contentAlignment = if (minLines == 1) Alignment.CenterStart else Alignment.TopStart,
                ) {
                    if (value.isEmpty() && placeholder != null) {
                        Text(placeholder, style = textStyle, color = Colors.textSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    field()
                }
            },
        )
        if (error != null) FieldError(error)
    }
}

/** `✕ message` under a field or a group of controls. */
@Composable
internal fun FieldError(text: String) {
    Row {
        Text(
            stringResource(R.string.glyph_error) + " ",
            style = Type.label,
            color = Colors.danger,
            modifier = Modifier.clearAndSetSemantics {},
        )
        Text(text, style = Type.label, color = Colors.danger)
    }
}

/** Radio rows in a card: `(•)` or `( )`, a title, and an optional wire value on the right. */
@Composable
internal fun <T> ChoiceList(
    options: List<T>,
    selected: T,
    title: @Composable (T) -> String,
    detail: (T) -> String? = { null },
    onSelect: (T) -> Unit,
) {
    val on = stringResource(R.string.glyph_radio_on)
    val off = stringResource(R.string.glyph_radio_off)
    TerminalCard(Modifier.selectableGroup()) {
        options.forEachIndexed { index, option ->
            if (index > 0) CardDivider()
            val isSelected = option == selected
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = 52.dp)
                    .selectable(selected = isSelected, role = Role.RadioButton) { onSelect(option) }
                    .background(if (isSelected) Colors.surfaceSelected else Color.Transparent)
                    .padding(horizontal = 14.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    if (isSelected) on else off,
                    style = Type.body.copy(fontWeight = androidx.compose.ui.text.font.FontWeight.Bold),
                    color = if (isSelected) Colors.accent else Colors.textSecondary,
                    modifier = Modifier.clearAndSetSemantics {},
                )
                Text(title(option), style = Type.body, color = Colors.textPrimary, modifier = Modifier.weight(1f))
                detail(option)?.let {
                    Text(it, style = Type.label, color = Colors.textSecondary, modifier = Modifier.clearAndSetSemantics {})
                }
            }
        }
    }
}

/** Equal-width option buttons, one selected. */
@Composable
internal fun <T> SegmentedChoice(options: List<T>, selected: T, label: @Composable (T) -> String, onSelect: (T) -> Unit) {
    Row(Modifier.fillMaxWidth().selectableGroup(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        options.forEach { option ->
            val isSelected = option == selected
            Box(
                modifier = Modifier
                    .weight(1f)
                    .heightIn(min = 44.dp)
                    .clip(ControlShape)
                    .background(if (isSelected) Colors.surfaceSelected else Colors.surface)
                    .border(1.dp, if (isSelected) Colors.accent else Colors.border, ControlShape)
                    .selectable(selected = isSelected, role = Role.RadioButton) { onSelect(option) },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    label(option),
                    style = if (isSelected) Type.small.copy(fontWeight = androidx.compose.ui.text.font.FontWeight.Bold) else Type.small,
                    color = if (isSelected) Colors.accent else Colors.textPrimary,
                    maxLines = 1,
                )
            }
        }
    }
}

/** A card row whose whole area toggles the square switch on its right. */
@Composable
internal fun SwitchRow(label: String, checked: Boolean, onCheckedChange: (Boolean) -> Unit) {
    TerminalCard {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = 56.dp)
                .toggleable(value = checked, role = Role.Switch, onValueChange = onCheckedChange)
                .padding(horizontal = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(label, style = Type.body, color = Colors.textPrimary, modifier = Modifier.weight(1f))
            TerminalSwitch(checked)
        }
    }
}

@Composable
private fun TerminalSwitch(checked: Boolean) {
    Box(
        Modifier
            .size(52.dp, 30.dp)
            .background(if (checked) Colors.accent else Colors.control, RoundedCornerShape(6.dp))
            .padding(4.dp),
        contentAlignment = if (checked) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Box(Modifier.size(22.dp).background(Colors.textPrimary, RoundedCornerShape(4.dp)))
    }
}

internal enum class BannerKind { Info, Warning, Danger }

@Composable
internal fun Banner(kind: BannerKind, text: String, @DrawableRes icon: Int? = null) {
    val (background, outline, color) = when (kind) {
        BannerKind.Info -> Triple(Colors.infoBackground, Colors.infoBorder, Colors.infoText)
        BannerKind.Warning -> Triple(Colors.warningBackground, Colors.warningBorder, Colors.warningText)
        BannerKind.Danger -> Triple(Colors.dangerBackground, Colors.danger, Colors.dangerText)
    }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(background, ControlShape)
            .border(1.dp, outline, ControlShape)
            .padding(horizontal = 14.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (icon != null) {
            Icon(painterResource(icon), contentDescription = null, tint = color, modifier = Modifier.padding(top = 1.dp).size(18.dp))
        }
        Text(text, style = Type.small, color = color)
    }
}

internal fun LatencyGrade.color(): Color = when (this) {
    LatencyGrade.Testing -> Colors.textSecondary
    LatencyGrade.Good -> Colors.accent
    LatencyGrade.Slow -> Colors.warning
    LatencyGrade.Failed -> Colors.danger
}

/** 24-cell character bar; decorative, the latency text says the same. */
@Composable
internal fun LatencyBar(ping: PingState) {
    val filled = Latency.filledCells(ping) ?: return
    val full = stringResource(R.string.glyph_bar_filled)
    val empty = stringResource(R.string.glyph_bar_empty)
    val failed = stringResource(R.string.glyph_bar_failed)
    val text = buildAnnotatedString {
        if (filled == 0) {
            withStyle(SpanStyle(color = Colors.border)) { append(failed.repeat(Latency.BAR_CELLS)) }
        } else {
            withStyle(SpanStyle(color = Latency.grade(ping).color())) { append(full.repeat(filled)) }
            withStyle(SpanStyle(color = Colors.border)) { append(empty.repeat(Latency.BAR_CELLS - filled)) }
        }
    }
    Text(text, style = Type.label.copy(letterSpacing = 1.sp), maxLines = 1, softWrap = false, modifier = Modifier.clearAndSetSemantics {})
}

/** Splits a formatted rate such as `4.60 MB/s` into number and unit. */
internal fun splitRate(rate: String): Pair<String, String> {
    val space = rate.lastIndexOf(' ')
    return if (space < 0) rate to "" else rate.substring(0, space) to rate.substring(space + 1)
}

/**
 * Upload and download speeds of the last minute, newest on the right. The
 * scale has a floor so idle noise stays flat.
 */
@Composable
internal fun TrafficChart(history: List<TrafficPoint>, description: String, modifier: Modifier = Modifier) {
    Canvas(modifier.fillMaxWidth().height(72.dp).semantics { contentDescription = description }) {
        val dash = PathEffect.dashPathEffect(floatArrayOf(2.dp.toPx(), 4.dp.toPx()))
        for (line in 1..2) {
            val y = size.height * line / 3
            drawLine(Colors.gridline, Offset(0f, y), Offset(size.width, y), strokeWidth = 1.dp.toPx(), pathEffect = dash)
        }
        if (history.isEmpty()) return@Canvas
        val peak = max(history.maxOf { max(it.uploadSpeed, it.downloadSpeed) } * CHART_HEADROOM, CHART_FLOOR_BYTES)
        val step = size.width / (CHART_POINTS - 1)
        val start = size.width - step * (history.size - 1)
        fun x(index: Int) = start + step * index
        fun y(speed: Long) = size.height * (1f - (speed / peak).toFloat())

        val download = Path().apply {
            history.forEachIndexed { i, p -> if (i == 0) moveTo(x(i), y(p.downloadSpeed)) else lineTo(x(i), y(p.downloadSpeed)) }
        }
        val area = Path().apply {
            addPath(download)
            lineTo(x(history.lastIndex), size.height)
            lineTo(x(0), size.height)
            close()
        }
        val upload = Path().apply {
            history.forEachIndexed { i, p -> if (i == 0) moveTo(x(i), y(p.uploadSpeed)) else lineTo(x(i), y(p.uploadSpeed)) }
        }
        val stroke = Stroke(width = 1.6.dp.toPx(), join = StrokeJoin.Round)
        drawPath(area, Colors.syntaxKey.copy(alpha = 0.10f))
        drawPath(download, Colors.syntaxKey, style = stroke)
        drawPath(upload, Colors.syntaxNumber, style = stroke)
    }
}

private const val CHART_POINTS = 60
private const val CHART_HEADROOM = 1.2
private const val CHART_FLOOR_BYTES = 64.0 * 1024

/** A small colored line used in chart legends. */
@Composable
internal fun LegendMark(color: Color) {
    Spacer(Modifier.width(10.dp).height(2.dp).background(color))
}

/** Plain text that screen readers skip, for glyphs such as `→`. */
@Composable
internal fun Glyph(text: String, color: Color = Colors.textSecondary, style: TextStyle = Type.body) {
    Text(text, style = style, color = color, modifier = Modifier.clearAndSetSemantics {})
}

internal fun AnnotatedString.Builder.appendColored(text: String, color: Color) {
    withStyle(SpanStyle(color = color)) { append(text) }
}
