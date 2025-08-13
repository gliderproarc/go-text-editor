//go:build tree_sitter

package plugins

// HighlighterFor returns a highlighter instance for the language when tree-sitter providers are available.
func HighlighterFor(lang *LanguageSpec) Highlighter {
    if lang == nil {
        return nil
    }
    switch lang.Highlighter {
    case "tree-sitter-go":
        return NewTreeSitterPlugin()
    case "markdown-basic":
        return NewMarkdownHighlighter()
    default:
        return nil
    }
}

