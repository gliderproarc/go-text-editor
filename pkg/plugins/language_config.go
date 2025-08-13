package plugins

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
)

// LanguageSpec defines a language entry in config.
type LanguageSpec struct {
    ID         string   `json:"id"`
    Name       string   `json:"name"`
    Extensions []string `json:"extensions"`
    Highlighter string  `json:"highlighter"`
}

// LanguageConfig is the root schema.
type LanguageConfig struct {
    Languages []LanguageSpec `json:"languages"`
}

var defaultLanguageConfig = LanguageConfig{
    Languages: []LanguageSpec{
        {ID: "go", Name: "Go", Extensions: []string{".go"}, Highlighter: "tree-sitter-go"},
        {ID: "markdown", Name: "Markdown", Extensions: []string{".md", ".markdown"}, Highlighter: "markdown-basic"},
    },
}

// LoadLanguageConfig loads config from the given JSON path.
// If missing or invalid, returns defaults.
func LoadLanguageConfig(path string) *LanguageConfig {
    data, err := os.ReadFile(path)
    if err != nil {
        return &defaultLanguageConfig
    }
    var cfg LanguageConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return &defaultLanguageConfig
    }
    return &cfg
}

// DetectLanguageByPath returns the first matching language by extension.
func DetectLanguageByPath(cfg *LanguageConfig, path string) *LanguageSpec {
    ext := strings.ToLower(filepath.Ext(path))
    if ext == "" {
        return nil
    }
    for _, lang := range cfg.Languages {
        for _, e := range lang.Extensions {
            if strings.EqualFold(e, ext) {
                // return a copy to avoid external mutation
                l := lang
                return &l
            }
        }
    }
    return nil
}

// HighlighterFor is provided by build-specific files (see language_provider_*.go).
