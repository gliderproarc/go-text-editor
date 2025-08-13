package plugins

import "example.com/texteditor/pkg/search"

// Highlighter provides syntax highlighting ranges for source text.
type Highlighter interface {
	Plugin
	Highlight(src []byte) []search.Range
}
