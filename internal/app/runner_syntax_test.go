//go:build tree_sitter

package app

import (
	"testing"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/search"
)

type fakeHighlighter struct{ calls int }

func (f *fakeHighlighter) Name() string { return "fake" }
func (f *fakeHighlighter) Highlight(src []byte) []search.Range {
	f.calls++
	return []search.Range{}
}

func TestSyntaxHighlightsCacheInvalidation(t *testing.T) {
    r := &Runner{Buf: buffer.NewGapBufferFromString("package main\n"), FilePath: "example.go"}
    fh := &fakeHighlighter{}
    r.Syntax = fh

	r.syntaxHighlights()
	if fh.calls != 1 {
		t.Fatalf("expected first call to run highlighter, got %d", fh.calls)
	}
	r.syntaxHighlights()
	if fh.calls != 1 {
		t.Fatalf("expected cached highlights, got %d", fh.calls)
	}
	if err := r.Buf.Insert(0, []rune("func ")); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	r.syntaxHighlights()
	if fh.calls != 2 {
		t.Fatalf("expected cache invalidation after edit, got %d", fh.calls)
	}
}
