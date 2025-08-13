package plugins

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// TreeSitterPlugin wraps a tree-sitter parser for Go code.
type TreeSitterPlugin struct {
	parser *sitter.Parser
}

// NewTreeSitterPlugin initializes the parser with the Go language grammar.
func NewTreeSitterPlugin() *TreeSitterPlugin {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	return &TreeSitterPlugin{parser: p}
}

// Name identifies the plugin.
func (t *TreeSitterPlugin) Name() string { return "tree-sitter-go" }

// Parse returns a syntax tree for the provided source code.
func (t *TreeSitterPlugin) Parse(src []byte) *sitter.Tree {
	return t.parser.Parse(nil, src)
}
