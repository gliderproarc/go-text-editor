//go:build tree_sitter

package plugins

import "testing"

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
