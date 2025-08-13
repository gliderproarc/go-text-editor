//go:build tree_sitter

package app

import (
	"example.com/texteditor/pkg/plugins"
	"example.com/texteditor/pkg/search"
)

// syntaxHighlights returns tree-sitter based highlight ranges.
// Results are cached until the buffer content changes.
func (r *Runner) syntaxHighlights() []search.Range {
	if r.Buf == nil {
		return nil
	}
	if r.syntaxCache != nil && r.syntaxSrc != "" {
		return r.syntaxCache
	}
	src := r.Buf.String()
	if r.Syntax == nil {
		r.Syntax = plugins.NewTreeSitterPlugin()
	}
	ranges := r.Syntax.Highlight([]byte(src))
	if ranges == nil {
		ranges = []search.Range{}
	}
	r.syntaxSrc = src
	r.syntaxCache = ranges
	return ranges
}
