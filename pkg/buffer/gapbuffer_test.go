package buffer

import "testing"

func TestGapBuffer_InsertDelete(t *testing.T) {
	g := NewGapBufferFromString("Hello World")
	if g.String() != "Hello World" {
		t.Fatalf("expected initial content 'Hello World', got %q", g.String())
	}
	// insert comma after Hello
	err := g.Insert(5, []rune{','})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if g.String() != "Hello, World" {
		t.Fatalf("expected 'Hello, World', got %q", g.String())
	}
	// delete the comma
	err = g.Delete(5, 6)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if g.String() != "Hello World" {
		t.Fatalf("expected 'Hello World' after delete, got %q", g.String())
	}
}

func TestGapBuffer_LineAt(t *testing.T) {
	g := NewGapBufferFromString("one\ntwo\nthree")
	start, end := g.LineAt(1) // line 1 should be 'two\n'
	line := string(g.Slice(start, end))
	if line != "two\n" {
		t.Fatalf("expected line 'two\\n', got %q", line)
	}
}
