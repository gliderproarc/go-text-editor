package search

import "testing"

func TestSearchAll(t *testing.T) {
	text := "hello world hello"
	r := SearchAll(text, "hello")
	if len(r) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(r))
	}
	if r[0].Start != 0 || r[0].End != 5 {
		t.Fatalf("first match incorrect: %#v", r[0])
	}
	if r[1].Start != 12 || r[1].End != 17 {
		t.Fatalf("second match incorrect: %#v", r[1])
	}
}

func TestSearchNext(t *testing.T) {
	text := "hello world hello"
	ranges := SearchAll(text, "hello")
	if idx := SearchNext(ranges, 0); idx != 0 {
		t.Fatalf("expected next index 0 for pos 0, got %d", idx)
	}
	if idx := SearchNext(ranges, 6); idx != 1 {
		t.Fatalf("expected next index 1 for pos 6, got %d", idx)
	}
	// past last match should wrap to 0
	if idx := SearchNext(ranges, 100); idx != 0 {
		t.Fatalf("expected wrap to 0 for pos past end, got %d", idx)
	}
}
