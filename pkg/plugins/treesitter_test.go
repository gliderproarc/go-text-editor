//go:build tree_sitter

package plugins

import (
	"bytes"
	"testing"
)

func TestTreeSitterParse(t *testing.T) {
	ts := NewTreeSitterPlugin()
	code := []byte("package main\nfunc main() {}\n")
	tree := ts.Parse(code)
	if tree.RootNode().Type() != "source_file" {
		t.Fatalf("expected source_file, got %s", tree.RootNode().Type())
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager()
	ts := NewTreeSitterPlugin()
	m.Register(ts)
	if _, ok := m.Get(ts.Name()); !ok {
		t.Fatalf("tree-sitter plugin not registered")
	}
}

func TestTreeSitterHighlight(t *testing.T) {
	ts := NewTreeSitterPlugin()
	code := []byte("package main\nfunc main() {return \"hi\"}\n")
	ranges := ts.Highlight(code)
	if len(ranges) == 0 {
		t.Fatalf("expected highlights, got none")
	}
	funcOff := bytes.Index(code, []byte("func"))
	strOff := bytes.Index(code, []byte("\"hi\""))
	funcFound := false
	strFound := false
	for _, r := range ranges {
		if r.Start == funcOff {
			funcFound = true
		}
		if r.Start == strOff {
			strFound = true
		}
	}
	if !funcFound {
		t.Fatalf("expected func keyword to be highlighted")
	}
	if !strFound {
		t.Fatalf("expected string literal to be highlighted")
	}
}
