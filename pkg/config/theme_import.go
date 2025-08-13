package config

import (
    "bufio"
    "errors"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "github.com/gdamore/tcell/v2"
)

// ImportTheme reads a theme file in a known format and converts it to Theme.
// Supported:
// - Base16 YAML (keys base00..base0F)
// - Alacritty YAML (colors.primary/normal/bright/cursor/selection)
func ImportTheme(path string) (Theme, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Theme{}, err
    }
    content := string(data)
    lower := strings.ToLower(content)
    switch {
    case strings.Contains(lower, "base00:"):
        return importBase16(content), nil
    case strings.Contains(lower, "colors:"):
        return importAlacritty(content), nil
    default:
        return Theme{}, errors.New("unrecognized theme format: " + filepath.Base(path))
    }
}

var reKVHex = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*:\s*['\"]?([#0-9a-fA-Fx]{6,8})['\"]?\s*$`)

func parseHexToColor(v string, fallback tcell.Color) tcell.Color {
    v = strings.TrimSpace(v)
    v = strings.Trim(v, "'\"")
    if strings.HasPrefix(v, "#") {
        v = v[1:]
    } else if strings.HasPrefix(strings.ToLower(v), "0x") {
        v = v[2:]
    }
    if len(v) != 6 {
        return fallback
    }
    if _, err := strconv.ParseInt(v, 16, 32); err != nil {
        return fallback
    }
    return ParseColor("#"+strings.ToLower(v), fallback)
}

// importBase16 parses a Base16 YAML scheme or theme.
func importBase16(s string) Theme {
    // Collect baseXX values
    bases := map[string]string{}
    scanner := bufio.NewScanner(strings.NewReader(s))
    for scanner.Scan() {
        line := scanner.Text()
        m := reKVHex.FindStringSubmatch(line)
        if len(m) == 3 && strings.HasPrefix(strings.ToLower(m[1]), "base") {
            key := strings.ToLower(m[1])
            bases[key] = m[2]
        }
    }
    t := DefaultTheme()
    get := func(k string, fb tcell.Color) tcell.Color { return parseHexToColor(bases[k], fb) }

    // Map common roles
    t.UIBackground = get("base00", t.UIBackground)
    t.UIForeground = get("base05", t.UIForeground)
    t.TextDefault = t.UIForeground

    // Choose status/mini backgrounds slightly offset if available
    t.StatusBackground = get("base01", t.StatusBackground)
    if t.StatusBackground == tcell.ColorDefault {
        t.StatusBackground = get("base02", t.StatusBackground)
    }
    if t.StatusBackground == tcell.ColorDefault {
        t.StatusBackground = t.UIBackground
    }
    t.StatusForeground = t.UIForeground
    t.MiniBackground = t.StatusBackground
    t.MiniForeground = t.StatusForeground

    // Cursor styles from blue/green
    t.CursorInsertBG = get("base0d", t.CursorInsertBG)
    t.CursorNormalBG = get("base0b", t.CursorNormalBG)
    t.CursorText = t.UIBackground

    // Search highlights
    t.HighlightSearchBG = get("base0a", t.HighlightSearchBG)
    t.HighlightSearchFG = t.UIBackground
    t.HighlightSearchCurrentBG = t.CursorInsertBG
    t.HighlightSearchCurrentFG = t.UIBackground

    // Syntax groups
    if t.SyntaxColors == nil {
        t.SyntaxColors = map[string]tcell.Color{}
    }
    t.SyntaxColors["keyword"] = get("base08", t.SyntaxColors["keyword"]) // red
    t.SyntaxColors["string"] = get("base0b", t.SyntaxColors["string"])    // green
    t.SyntaxColors["comment"] = get("base03", t.SyntaxColors["comment"])  // comments
    t.SyntaxColors["number"] = get("base0a", t.SyntaxColors["number"])    // yellow
    t.SyntaxColors["type"] = get("base0d", t.SyntaxColors["type"])        // blue
    t.SyntaxColors["function"] = get("base0c", t.SyntaxColors["function"]) // cyan

    return t
}

// importAlacritty parses an Alacritty colors YAML fragment.
func importAlacritty(s string) Theme {
    // Very light YAML walker based on indentation and key:value lines
    type entry struct{ indent int; key string }
    var stack []entry
    kv := map[string]string{}
    scanner := bufio.NewScanner(strings.NewReader(s))
    for scanner.Scan() {
        raw := scanner.Text()
        line := strings.TrimRight(raw, "\r\n")
        if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
            continue
        }
        indent := len(line) - len(strings.TrimLeft(line, " "))
        for len(stack) > 0 && indent <= stack[len(stack)-1].indent {
            stack = stack[:len(stack)-1]
        }
        m := reKVHex.FindStringSubmatch(line)
        if len(m) == 3 {
            // form full key path
            path := ""
            for _, e := range stack {
                if path == "" { path = e.key } else { path += "." + e.key }
            }
            if path != "" { path += "." }
            full := path + strings.ToLower(m[1])
            kv[full] = m[2]
            continue
        }
        // section header without value
        if i := strings.Index(line, ":"); i >= 0 && i == len(line)-1 {
            key := strings.ToLower(strings.TrimSpace(line[:i]))
            stack = append(stack, entry{indent: indent, key: key})
        }
    }

    t := DefaultTheme()
    getPath := func(p string, fb tcell.Color) tcell.Color {
        if v, ok := kv[strings.ToLower(p)]; ok {
            return parseHexToColor(v, fb)
        }
        return fb
    }

    // Primary
    t.UIBackground = getPath("colors.primary.background", t.UIBackground)
    t.UIForeground = getPath("colors.primary.foreground", t.UIForeground)
    t.TextDefault = t.UIForeground

    // Cursor
    t.CursorInsertBG = getPath("colors.normal.blue", t.CursorInsertBG)
    t.CursorNormalBG = getPath("colors.normal.green", t.CursorNormalBG)
    t.CursorText = getPath("colors.cursor.text", t.UIBackground)
    if t.CursorText == tcell.ColorDefault {
        t.CursorText = t.UIBackground
    }

    // Highlights
    t.HighlightSearchBG = getPath("colors.normal.yellow", t.HighlightSearchBG)
    t.HighlightSearchFG = t.UIBackground
    t.HighlightSearchCurrentBG = getPath("colors.normal.blue", t.HighlightSearchCurrentBG)
    t.HighlightSearchCurrentFG = t.UIBackground

    // Status/Mini bars
    t.StatusBackground = getPath("colors.bright.black", t.StatusBackground)
    if t.StatusBackground == tcell.ColorDefault {
        t.StatusBackground = getPath("colors.normal.white", t.StatusBackground)
    }
    if t.StatusBackground == tcell.ColorDefault {
        t.StatusBackground = t.UIBackground
    }
    t.StatusForeground = t.UIForeground
    t.MiniBackground = t.StatusBackground
    t.MiniForeground = t.StatusForeground

    // Syntax groups from palette
    if t.SyntaxColors == nil {
        t.SyntaxColors = map[string]tcell.Color{}
    }
    t.SyntaxColors["keyword"] = getPath("colors.normal.red", t.SyntaxColors["keyword"])      // red
    t.SyntaxColors["string"] = getPath("colors.normal.green", t.SyntaxColors["string"])      // green
    t.SyntaxColors["comment"] = getPath("colors.bright.black", t.SyntaxColors["comment"])    // gray
    if t.SyntaxColors["comment"] == tcell.ColorDefault {
        t.SyntaxColors["comment"] = getPath("colors.normal.black", t.SyntaxColors["comment"]) // fallback
    }
    t.SyntaxColors["number"] = getPath("colors.normal.yellow", t.SyntaxColors["number"])     // yellow
    t.SyntaxColors["type"] = getPath("colors.normal.blue", t.SyntaxColors["type"])           // blue
    t.SyntaxColors["function"] = getPath("colors.normal.cyan", t.SyntaxColors["function"])   // cyan

    return t
}
