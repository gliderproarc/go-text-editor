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
}

// Default returns a Config with default key mappings.
func Default() *Config {
	return &Config{Keymap: DefaultKeymap()}
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
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !inKeymap {
			if line == "keymap:" {
				inKeymap = true
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid config line: " + line)
		}
		cmd := strings.TrimSpace(parts[0])
		binding := strings.TrimSpace(parts[1])
		kb, err := ParseKeybinding(binding)
		if err != nil {
			return nil, err
		}
		cfg.Keymap[cmd] = kb
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
