package app

import (
    "fmt"
    "os"
    "path/filepath"

    "example.com/texteditor/pkg/config"
)

// themeEntry represents a theme option that can be applied at runtime.
type themeEntry struct {
    Name string
    // If Path is non-empty, load from file; otherwise use Builtin name
    Path string
}

// initThemeList builds a list of themes from built-ins and config/themes files.
func (r *Runner) initThemeList() {
    // Start with built-ins
    list := []themeEntry{
        {Name: "default"},
        {Name: "light"},
        {Name: "dark"},
    }
    // Add any YAML files under ./config/themes
    base := filepath.Join("config", "themes")
    _ = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
        if err != nil || info == nil || info.IsDir() {
            return nil
        }
        ext := filepath.Ext(info.Name())
        if ext == ".yaml" || ext == ".yml" { // add simple YAML themes
            rel, _ := filepath.Rel(base, p)
            name := rel
            list = append(list, themeEntry{Name: name, Path: p})
        }
        return nil
    })
    r.themeList = list
    // Initialize index to current theme if possible; default to 0
    r.themeIndex = 0
}

func (r *Runner) applyThemeEntry(e themeEntry) {
    var t config.Theme
    if e.Path != "" {
        if nt, err := config.ImportTheme(e.Path); err == nil {
            t = nt
        } else {
            // fallback to default if import fails
            t = config.DefaultTheme()
        }
    } else {
        if bt, ok := config.BuiltinThemes[e.Name]; ok {
            t = bt
        } else {
            t = config.DefaultTheme()
        }
    }
    r.Theme = t
    // notify via mini-buffer
    r.setMiniBuffer([]string{fmt.Sprintf("Theme: %s", e.Name)})
    r.draw(nil)
    r.clearMiniBuffer()
}

// NextTheme cycles to the next theme and applies it.
func (r *Runner) NextTheme() {
    if len(r.themeList) == 0 {
        r.initThemeList()
    }
    if len(r.themeList) == 0 {
        return
    }
    r.themeIndex = (r.themeIndex + 1) % len(r.themeList)
    r.applyThemeEntry(r.themeList[r.themeIndex])
}

// PrevTheme cycles to the previous theme and applies it.
func (r *Runner) PrevTheme() {
    if len(r.themeList) == 0 {
        r.initThemeList()
    }
    if len(r.themeList) == 0 {
        return
    }
    r.themeIndex = (r.themeIndex - 1 + len(r.themeList)) % len(r.themeList)
    r.applyThemeEntry(r.themeList[r.themeIndex])
}

