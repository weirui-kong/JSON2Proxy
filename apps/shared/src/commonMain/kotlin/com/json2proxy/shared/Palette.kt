package com.json2proxy.shared

/**
 * The app's color card, as ARGB values. Screens on both platforms take every
 * color from here; contrast ratios are listed in the terminal UI spec.
 */
object Palette {
    /** Page background and text field background. */
    const val canvas: Long = 0xFF0E1216

    /** Cards, lists and secondary buttons. */
    const val surface: Long = 0xFF161C22

    /** Selected rows and options. */
    const val surfaceSelected: Long = 0xFF16291D

    /** Decorative outlines and dividers. */
    const val border: Long = 0xFF2E363F

    /** Outlines that identify a control: text fields and switches that are off. */
    const val control: Long = 0xFF646F7C

    /** Traffic chart grid lines. */
    const val gridline: Long = 0xFF222A31

    const val textPrimary: Long = 0xFFE4EAF0

    /** Labels, notes, placeholders and other secondary text. */
    const val textSecondary: Long = 0xFF8D97A3

    /** Prompt glyphs, selection, connected state, good latency and primary buttons. */
    const val accent: Long = 0xFF46C35A

    /** Text on [accent]. */
    const val onAccent: Long = 0xFF0E1216

    /** Disconnect button, errors, failures and timeouts. */
    const val danger: Long = 0xFFF2564B

    /** Slow latency, transitional connection states and the torch when on. */
    const val warning: Long = 0xFFD9A12B

    /** JSON keys, download curve and speed. */
    const val syntaxKey: Long = 0xFF7CB8F5

    /** JSON string values. */
    const val syntaxString: Long = 0xFFA8CFF7

    /** JSON numbers, upload curve and speed. */
    const val syntaxNumber: Long = 0xFFF5A25A

    const val infoBackground: Long = 0xFF0F1E30
    const val infoBorder: Long = 0xFF2A6BC9
    const val infoText: Long = 0xFFCCDDF5

    const val warningBackground: Long = 0xFF2B2110
    const val warningBorder: Long = 0xFFA0721A
    const val warningText: Long = 0xFFF0CF6E

    /** Error banners use [danger] as their outline. */
    const val dangerBackground: Long = 0xFF2E1517
    const val dangerText: Long = 0xFFFFA69D
}
