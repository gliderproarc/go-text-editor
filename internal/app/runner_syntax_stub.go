//go:build !tree_sitter

package app

import "example.com/texteditor/pkg/search"

// syntaxHighlights is a no-op when tree-sitter support is disabled.
func (r *Runner) syntaxHighlights() []search.Range { return nil }
