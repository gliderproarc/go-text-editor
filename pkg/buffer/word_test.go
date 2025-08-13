package buffer

import "testing"

func TestWordEndAtWordBoundary(t *testing.T) {
	g := NewGapBufferFromString("one two")
	if got := WordEnd(g, 2); got != 6 {
		t.Fatalf("expected 6, got %d", got)
	}
}

func TestWordEndInsideWord(t *testing.T) {
	g := NewGapBufferFromString("one")
	if got := WordEnd(g, 1); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}
