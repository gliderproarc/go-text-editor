//go:build tree_sitter

package app

import (
    "path/filepath"

    "example.com/texteditor/pkg/plugins"
    "example.com/texteditor/pkg/search"
)

// syntaxHighlights returns tree-sitter based highlight ranges.
// Results are cached until the buffer content changes.
func (r *Runner) syntaxHighlights() []search.Range {
    if r.Buf == nil {
        return nil
    }
    // Determine language by config and file extension
    var lang *plugins.LanguageSpec
    if r.FilePath != "" {
        cfg := plugins.LoadLanguageConfig(filepath.Join("config", "languages.json"))
        lang = plugins.DetectLanguageByPath(cfg, r.FilePath)
    }
    if lang == nil {
        return nil
    }
    src := r.Buf.String()
    if r.syntaxCache != nil && r.syntaxSrc == src {
        return r.syntaxCache
    }
    // instantiate highlighter for this language if not present or mismatched
    if r.Syntax == nil || r.Syntax.Name() != lang.Highlighter {
        r.Syntax = plugins.HighlighterFor(lang)
    }
    if r.Syntax == nil { // unsupported highlighter
        return nil
    }
    ranges := r.Syntax.Highlight([]byte(src))
    if ranges == nil {
        ranges = []search.Range{}
    }
    r.syntaxSrc = src
    r.syntaxCache = ranges
    return ranges
}
