//go:build !tree_sitter

package app

import (
    "path/filepath"

    "example.com/texteditor/pkg/plugins"
    "example.com/texteditor/pkg/search"
)

// syntaxHighlights uses non-tree-sitter highlighters when available (e.g., Markdown),
// and otherwise returns no highlights. This builds without the tree_sitter tag.
func (r *Runner) syntaxHighlights() []search.Range {
    if r.Buf == nil {
        return nil
    }
    var lang *plugins.LanguageSpec
    if r.FilePath != "" {
        cfg := plugins.LoadLanguageConfig(filepath.Join("config", "languages.json"))
        lang = plugins.DetectLanguageByPath(cfg, r.FilePath)
    }
    if lang == nil {
        return nil
    }
    // Only support non-tree-sitter highlighters here
    if lang.Highlighter != "markdown-basic" {
        return nil
    }
    src := r.Buf.String()
    if r.syntaxCache != nil && r.syntaxSrc == src {
        return r.syntaxCache
    }
    if r.Syntax == nil || r.Syntax.Name() != lang.Highlighter {
        r.Syntax = plugins.HighlighterFor(lang)
    }
    if r.Syntax == nil {
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
