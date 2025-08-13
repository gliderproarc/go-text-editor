package config

import (
    "strings"

    "github.com/gdamore/tcell/v2"
)

// Theme represents configurable colors for UI, cursor, highlights, and syntax.
type Theme struct {
    // Base UI
    UIBackground tcell.Color
    UIForeground tcell.Color

    // Status bar and mini-buffer
    StatusBackground tcell.Color
    StatusForeground tcell.Color
    MiniBackground   tcell.Color
    MiniForeground   tcell.Color

    // Cursor styles
    CursorText     tcell.Color
    CursorInsertBG tcell.Color
    CursorNormalBG tcell.Color

    // Text defaults
    TextDefault tcell.Color

    // Search/selection highlights
    HighlightSearchBG        tcell.Color
    HighlightSearchFG        tcell.Color
    HighlightSearchCurrentBG tcell.Color
    HighlightSearchCurrentFG tcell.Color

    // Syntax groups (keyword, string, comment, number, type, function, ...)
    SyntaxColors map[string]tcell.Color
}

// DefaultTheme returns the built-in light theme matching existing hardcoded colors.
func DefaultTheme() Theme {
    return Theme{
        UIBackground: tcell.ColorBlack,
        UIForeground: tcell.ColorWhite,

        StatusBackground: tcell.ColorWhite,
        StatusForeground: tcell.ColorBlack,
        MiniBackground:   tcell.ColorWhite,
        MiniForeground:   tcell.ColorBlack,

        CursorText:     tcell.ColorBlack,
        CursorInsertBG: tcell.ColorBlue,
        CursorNormalBG: tcell.ColorGreen,

        TextDefault: tcell.ColorWhite,

        HighlightSearchBG:        tcell.ColorYellow,
        HighlightSearchFG:        tcell.ColorBlack,
        HighlightSearchCurrentBG: tcell.ColorBlue,
        HighlightSearchCurrentFG: tcell.ColorWhite,

        SyntaxColors: map[string]tcell.Color{
            "keyword":  tcell.ColorRed,
            "string":   tcell.ColorGreen,
            "comment":  tcell.ColorGray,
            "number":   tcell.ColorYellow,
            "type":     tcell.ColorBlue,
            "function": tcell.ColorBlue,
        },
    }
}

// BuiltinThemes exposes a couple of presets by name.
var BuiltinThemes = map[string]Theme{
    "default": DefaultTheme(),
    "light":   DefaultTheme(),
    "dark": {
        UIBackground: tcell.ColorBlack,
        UIForeground: tcell.ColorWhite,

        StatusBackground: tcell.ColorGray,
        StatusForeground: tcell.ColorWhite,
        MiniBackground:   tcell.ColorGray,
        MiniForeground:   tcell.ColorWhite,

        CursorText:     tcell.ColorBlack,
        CursorInsertBG: tcell.ColorLightBlue,
        CursorNormalBG: tcell.ColorDarkGreen,

        TextDefault: tcell.ColorWhite,

        HighlightSearchBG:        tcell.ColorDarkOliveGreen,
        HighlightSearchFG:        tcell.ColorWhite,
        HighlightSearchCurrentBG: tcell.ColorBlue,
        HighlightSearchCurrentFG: tcell.ColorWhite,

        SyntaxColors: map[string]tcell.Color{
            "keyword":  tcell.ColorRed,
            "string":   tcell.ColorLightGreen,
            "comment":  tcell.ColorSilver,
            "number":   tcell.ColorLightYellow,
            "type":     tcell.ColorLightBlue,
            "function": tcell.ColorLightCyan,
        },
    },
}

// ParseColor returns a tcell.Color from a name or hex like "#aabbcc".
// If parsing fails, it returns the provided fallback.
func ParseColor(s string, fallback tcell.Color) tcell.Color {
    if s == "" {
        return fallback
    }
    // tcell.GetColor supports W3C names or #RRGGBB (case-insensitive)
    c := tcell.GetColor(strings.ToLower(s))
    if c == tcell.ColorDefault {
        return fallback
    }
    return c
}
