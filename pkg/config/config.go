package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Keybinding represents a single key combination.
type Keybinding struct {
	Key  tcell.Key
	Rune rune
	Mod  tcell.ModMask
}

// Config holds user configuration values.
type Config struct {
    Keymap map[string]Keybinding `yaml:"keymap"`
    Theme  Theme                 `yaml:"theme"`
}

// Default returns a Config with default key mappings.
func Default() *Config {
    // Default to the terminal-compliant theme so the editor inherits
    // the user's terminal colors when no config is provided.
    return &Config{Keymap: DefaultKeymap(), Theme: TerminalTheme()}
}

// DefaultKeymap provides builtin command bindings.
func DefaultKeymap() map[string]Keybinding {
	return map[string]Keybinding{
		"quit":   mustParse("Ctrl+Q"),
		"save":   mustParse("Ctrl+S"),
		"search": mustParse("Ctrl+W"),
		"menu":   mustParse("Ctrl+T"),
	}
}

// Load loads configuration from the provided path. If the file does not
// exist, defaults are returned.
func Load(path string) (*Config, error) {
    cfg := Default()
    data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
    inKeymap := false
    inTheme := false
    // allow "theme" block with flat keys like "ui.background: black"
    // and "syntax.<group>: <color>" as well as "preset: <name>"
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        // section entry
        if !inKeymap && !inTheme {
            switch line {
            case "keymap:":
                inKeymap = true
                continue
            case "theme:":
                inTheme = true
                continue
            default:
                // ignore unknown top-level for now
                continue
            }
        }
        parts := strings.SplitN(line, ":", 2)
        if len(parts) != 2 {
            return nil, errors.New("invalid config line: " + line)
        }
        k := strings.TrimSpace(parts[0])
        v := strings.TrimSpace(parts[1])
        if inKeymap {
            kb, err := ParseKeybinding(v)
            if err != nil {
                return nil, err
            }
            cfg.Keymap[k] = kb
            continue
        }
        if inTheme {
            if k == "preset" || k == "name" {
                // load builtin preset first; then allow overrides below
                if t, ok := BuiltinThemes[strings.ToLower(v)]; ok {
                    cfg.Theme = t
                }
                continue
            }
            if k == "file" || k == "path" || k == "import" {
                full := v
                if !filepath.IsAbs(full) {
                    // resolve relative to config file directory
                    full = filepath.Join(filepath.Dir(path), v)
                }
                if t, err := ImportTheme(full); err == nil {
                    cfg.Theme = t
                }
                continue
            }
            // route based on known keys and syntax.*
            switch strings.ToLower(k) {
            case "ui.background":
                cfg.Theme.UIBackground = ParseColor(v, cfg.Theme.UIBackground)
            case "ui.foreground":
                cfg.Theme.UIForeground = ParseColor(v, cfg.Theme.UIForeground)
            case "status.bg", "status.background":
                cfg.Theme.StatusBackground = ParseColor(v, cfg.Theme.StatusBackground)
            case "status.fg", "status.foreground":
                cfg.Theme.StatusForeground = ParseColor(v, cfg.Theme.StatusForeground)
            case "mini.bg", "mini.background":
                cfg.Theme.MiniBackground = ParseColor(v, cfg.Theme.MiniBackground)
            case "mini.fg", "mini.foreground":
                cfg.Theme.MiniForeground = ParseColor(v, cfg.Theme.MiniForeground)
            case "cursor.text", "cursor.fg", "cursor.foreground":
                cfg.Theme.CursorText = ParseColor(v, cfg.Theme.CursorText)
            case "cursor.insert.bg", "cursor.insert.background":
                cfg.Theme.CursorInsertBG = ParseColor(v, cfg.Theme.CursorInsertBG)
            case "cursor.normal.bg", "cursor.normal.background":
                cfg.Theme.CursorNormalBG = ParseColor(v, cfg.Theme.CursorNormalBG)
            case "cursor.visual.bg", "cursor.visual.background":
                cfg.Theme.CursorVisualBG = ParseColor(v, cfg.Theme.CursorVisualBG)
            case "text.default", "text.fg":
                cfg.Theme.TextDefault = ParseColor(v, cfg.Theme.TextDefault)
            case "highlight.search.bg":
                cfg.Theme.HighlightSearchBG = ParseColor(v, cfg.Theme.HighlightSearchBG)
            case "highlight.search.fg":
                cfg.Theme.HighlightSearchFG = ParseColor(v, cfg.Theme.HighlightSearchFG)
            case "highlight.search.current.bg":
                cfg.Theme.HighlightSearchCurrentBG = ParseColor(v, cfg.Theme.HighlightSearchCurrentBG)
            case "highlight.search.current.fg":
                cfg.Theme.HighlightSearchCurrentFG = ParseColor(v, cfg.Theme.HighlightSearchCurrentFG)
            default:
                if strings.HasPrefix(strings.ToLower(k), "syntax.") {
                    group := strings.TrimPrefix(strings.ToLower(k), "syntax.")
                    if cfg.Theme.SyntaxColors == nil {
                        cfg.Theme.SyntaxColors = map[string]tcell.Color{}
                    }
                    cfg.Theme.SyntaxColors[group] = ParseColor(v, cfg.Theme.SyntaxColors[group])
                }
            }
            continue
        }
    }
    return cfg, nil
}

// LoadDefault attempts to read ~/.texteditor/config.yaml.
func LoadDefault() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Default(), nil
	}
	path := filepath.Join(home, ".texteditor", "config.yaml")
	return Load(path)
}

// ParseKeybinding converts a textual key description like "Ctrl+S" into a
// Keybinding. Currently only Ctrl+<letter> is supported.
func ParseKeybinding(s string) (Keybinding, error) {
	parts := strings.Split(s, "+")
	if len(parts) != 2 {
		return Keybinding{}, errors.New("invalid keybinding: " + s)
	}
	if !strings.EqualFold(parts[0], "ctrl") {
		return Keybinding{}, errors.New("invalid modifier in keybinding: " + s)
	}
	r := []rune(strings.ToLower(parts[1]))
	if len(r) != 1 || r[0] < 'a' || r[0] > 'z' {
		return Keybinding{}, errors.New("invalid key in keybinding: " + s)
	}
	return Keybinding{Key: tcell.KeyRune, Rune: r[0], Mod: tcell.ModCtrl}, nil
}

func mustParse(s string) Keybinding {
	kb, _ := ParseKeybinding(s)
	return kb
}

var ctrlMap = map[rune]tcell.Key{
	'a': tcell.KeyCtrlA,
	'b': tcell.KeyCtrlB,
	'c': tcell.KeyCtrlC,
	'd': tcell.KeyCtrlD,
	'e': tcell.KeyCtrlE,
	'f': tcell.KeyCtrlF,
	'g': tcell.KeyCtrlG,
	'h': tcell.KeyCtrlH,
	'i': tcell.KeyCtrlI,
	'j': tcell.KeyCtrlJ,
	'k': tcell.KeyCtrlK,
	'l': tcell.KeyCtrlL,
	'm': tcell.KeyCtrlM,
	'n': tcell.KeyCtrlN,
	'o': tcell.KeyCtrlO,
	'p': tcell.KeyCtrlP,
	'q': tcell.KeyCtrlQ,
	'r': tcell.KeyCtrlR,
	's': tcell.KeyCtrlS,
	't': tcell.KeyCtrlT,
	'u': tcell.KeyCtrlU,
	'v': tcell.KeyCtrlV,
	'w': tcell.KeyCtrlW,
	'x': tcell.KeyCtrlX,
	'y': tcell.KeyCtrlY,
	'z': tcell.KeyCtrlZ,
}

// Matches returns true if the binding matches the provided event.
func (k Keybinding) Matches(ev *tcell.EventKey) bool {
	if k.Key == ev.Key() && k.Rune == ev.Rune() && k.Mod == ev.Modifiers() {
		return true
	}
	if k.Key == tcell.KeyRune && k.Mod == tcell.ModCtrl {
		if ctrlKey, ok := ctrlMap[k.Rune]; ok && ev.Key() == ctrlKey {
			return true
		}
	}
	return false
}
