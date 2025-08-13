package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestImportTheme_Base16(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "base16.yaml")
    data := `
scheme: "base16-test"
base00: '181818'
base01: '282828'
base02: '383838'
base03: '585858'
base04: 'b8b8b8'
base05: 'd8d8d8'
base06: 'e8e8e8'
base07: 'f8f8f8'
base08: 'ab4642'
base09: 'dc9656'
base0A: 'f7ca88'
base0B: 'a1b56c'
base0C: '86c1b9'
base0D: '7cafc2'
base0E: 'ba8baf'
base0F: 'a16946'
`
    if err := os.WriteFile(path, []byte(data), 0644); err != nil {
        t.Fatalf("write: %v", err)
    }
    th, err := ImportTheme(path)
    if err != nil {
        t.Fatalf("import: %v", err)
    }
    // Basic expectations
    if th.UIForeground == th.UIBackground {
        t.Fatalf("expected fg != bg")
    }
    if _, ok := th.SyntaxColors["keyword"]; !ok {
        t.Fatalf("missing syntax.keyword")
    }
}

func TestImportTheme_Alacritty(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "alacritty.yml")
    data := `
colors:
  primary:
    background: '#1d1f21'
    foreground: '#c5c8c6'
  normal:
    black:   '0x1d1f21'
    red:     '0xcc6666'
    green:   '0xb5bd68'
    yellow:  '0xf0c674'
    blue:    '0x81a2be'
    magenta: '0xb294bb'
    cyan:    '0x8abeb7'
    white:   '0xc5c8c6'
  bright:
    black:   '0x969896'
    white:   '0xffffff'
`
    if err := os.WriteFile(path, []byte(data), 0644); err != nil {
        t.Fatalf("write: %v", err)
    }
    th, err := ImportTheme(path)
    if err != nil {
        t.Fatalf("import: %v", err)
    }
    if th.UIForeground == th.UIBackground {
        t.Fatalf("expected fg != bg")
    }
    if th.SyntaxColors["keyword"] == 0 {
        t.Fatalf("expected syntax keyword color")
    }
}
