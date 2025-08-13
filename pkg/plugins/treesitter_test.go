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

func TestTreeSitterHighlight(t *testing.T) {
	ts := NewTreeSitterPlugin()
	code := []byte("package main\nfunc main() {return}\n")
	ranges := ts.Highlight(code)
	if len(ranges) == 0 {
		t.Fatalf("expected highlights, got none")
	}
	// ensure the \"func\" keyword is highlighted
	found := false
	for _, r := range ranges {
		if r.Start == 13 { // byte offset of "func"
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected func keyword to be highlighted")
	}
}
