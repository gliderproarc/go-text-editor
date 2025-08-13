//go:build tree_sitter

package plugins

import (
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/golang"
)

// TreeSitterPlugin wraps a tree-sitter parser for Go code.
type TreeSitterPlugin struct {
    parser *sitter.Parser
}

// adapter types to satisfy SyntaxTree and SyntaxNode without exposing sitter
type tsTree struct{ t *sitter.Tree }
type tsNode struct{ n sitter.Node }

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
