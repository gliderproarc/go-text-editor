package app

import (
	"testing"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/history"
	"github.com/gdamore/tcell/v2"
)

func TestHandleKeyEvent_WordMotionCounts(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("one two three"), History: history.New()}

	// 2w should advance two words to the start of "three".
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '2', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'w', 0))
	if r.Cursor != 8 {
		t.Fatalf("expected cursor at 8 after 2w, got %d", r.Cursor)
	}

	// Count should reset; a plain w advances one word to the end.
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'w', 0))
	if r.Cursor != r.Buf.Len() {
		t.Fatalf("expected cursor at end after final w, got %d", r.Cursor)
	}
}

func TestHandleKeyEvent_DeleteWordsCountAndRepeat(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("one two three four"), History: history.New()}

	// d2w deletes the first two words and records the change for dot-repeat.
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'd', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '2', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'w', 0))

	if got := r.Buf.String(); got != "three four" {
		t.Fatalf("expected buffer %q after d2w, got %q", "three four", got)
	}
	if !r.KillRing.HasData() || r.KillRing.Get() != "one two " {
		t.Fatalf("expected kill ring to contain %q, got %q", "one two ", r.KillRing.Get())
	}
	if r.lastChange == nil || r.lastChange.count != 2 {
		t.Fatalf("expected last change to store count=2, got %#v", r.lastChange)
	}

	// Dot-repeat should delete the next two words using the stored count.
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '.', 0))
	if got := r.Buf.String(); got != "" {
		t.Fatalf("expected buffer to be empty after dot-repeat, got %q", got)
	}
}

func TestHandleKeyEvent_ChangeWordDotRepeat(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("one two three"), History: history.New()}

	// cwONE<space><Esc> replaces the first word and stores change for dot-repeat.
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'w', 0))
	for _, ch := range []rune("ONE ") {
		r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyEsc, 0, 0))

	if got := r.Buf.String(); got != "ONE two three" {
		t.Fatalf("expected buffer %q after cw insert, got %q", "ONE two three", got)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode to return to normal after Esc, got %v", r.Mode)
	}

	// Dot-repeat should apply the same replacement to the next word.
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '.', 0))
	if got := r.Buf.String(); got != "ONE ONE three" {
		t.Fatalf("expected buffer %q after dot-repeat, got %q", "ONE ONE three", got)
	}
}

func TestHandleKeyEvent_DeleteInnerQuotes(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("say \"hello\" world"), History: history.New(), Cursor: 6}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'd', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '"', 0))

	if got := r.Buf.String(); got != "say \"\" world" {
		t.Fatalf("expected buffer %q after di\" , got %q", "say \"\" world", got)
	}
	if r.Cursor != 5 {
		t.Fatalf("expected cursor at 5 after di\", got %d", r.Cursor)
	}
}

func TestHandleKeyEvent_ChangeInnerQuotes(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("say \"hello\" world"), History: history.New(), Cursor: 6}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '"', 0))
	for _, ch := range []rune("hey") {
		r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyEsc, 0, 0))

	if got := r.Buf.String(); got != "say \"hey\" world" {
		t.Fatalf("expected buffer %q after ci\", got %q", "say \"hey\" world", got)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode to return to normal after Esc, got %v", r.Mode)
	}
}

func TestHandleKeyEvent_VisualInnerQuotes(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("say \"hello\" world"), History: history.New(), Cursor: 6}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '"', 0))

	start, end := r.visualSelectionBounds()
	if start != 5 || end != 10 {
		t.Fatalf("expected visual bounds 5..10 after vi\", got %d..%d", start, end)
	}
}
