//go:build tree_sitter

package plugins

import (
	"example.com/texteditor/pkg/search"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// TreeSitterPlugin wraps a tree-sitter parser for Go code.
type TreeSitterPlugin struct {
	parser *sitter.Parser
}

// adapter types to satisfy SyntaxTree and SyntaxNode without exposing sitter
type tsTree struct{ t *sitter.Tree }
type tsNode struct{ n *sitter.Node }

func (tw tsTree) RootNode() SyntaxNode { return tsNode{n: tw.t.RootNode()} }
func (nw tsNode) Type() string         { return nw.n.Type() }

// NewTreeSitterPlugin initializes the parser with the Go language grammar.
func NewTreeSitterPlugin() *TreeSitterPlugin {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	return &TreeSitterPlugin{parser: p}
}

// Name identifies the plugin.
func (t *TreeSitterPlugin) Name() string { return "tree-sitter-go" }

// Parse returns a syntax tree for the provided source code.
func (t *TreeSitterPlugin) Parse(src []byte) SyntaxTree {
	return tsTree{t: t.parser.Parse(nil, src)}
}

// Highlight returns byte ranges for basic Go syntax tokens.
// Currently it highlights keywords, comments and string literals.
func (t *TreeSitterPlugin) Highlight(src []byte) []search.Range {
	tree := t.parser.Parse(nil, src)
	if tree == nil {
		return nil
	}
	root := tree.RootNode()
	var ranges []search.Range
	keywords := map[string]struct{}{
		"break": {}, "case": {}, "chan": {}, "const": {}, "continue": {},
		"default": {}, "defer": {}, "else": {}, "fallthrough": {}, "for": {},
		"func": {}, "go": {}, "goto": {}, "if": {}, "import": {},
		"interface": {}, "map": {}, "package": {}, "range": {}, "return": {},
		"select": {}, "struct": {}, "switch": {}, "type": {}, "var": {},
	}
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		typ := n.Type()
		if _, ok := keywords[typ]; ok {
			ranges = append(ranges, search.Range{Start: int(n.StartByte()), End: int(n.EndByte())})
		}
		if typ == "comment" || typ == "interpreted_string_literal" || typ == "raw_string_literal" {
			ranges = append(ranges, search.Range{Start: int(n.StartByte()), End: int(n.EndByte())})
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return ranges
}
