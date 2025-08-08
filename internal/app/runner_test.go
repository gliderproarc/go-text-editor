package app

import (
	"os"
	"strings"
	"testing"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

func TestHandleKeyEvent_CtrlQ_Rune(t *testing.T) {
	r := &Runner{}
	ev := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl)
	if !r.handleKeyEvent(ev) {
		t.Fatalf("expected Ctrl+q rune event to signal quit")
	}
}

func TestHandleKeyEvent_CtrlQ_Key(t *testing.T) {
	r := &Runner{}
	ev := tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)
	if !r.handleKeyEvent(ev) {
		t.Fatalf("expected KeyCtrlQ event to signal quit")
	}
}

func TestHandleKeyEvent_ShowHelp(t *testing.T) {
	r := &Runner{}
	ev := tcell.NewEventKey(tcell.KeyRune, 'h', 0)
	if r.handleKeyEvent(ev) {
		t.Fatalf("h should not signal quit")
	}
	if !r.ShowHelp {
		t.Fatalf("expected ShowHelp to be set after pressing h")
	}
}

func TestRunner_InsertAndSave(t *testing.T) {
	tmp, err := os.CreateTemp("", "texteditor_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	r := &Runner{Buf: buffer.NewGapBuffer(0), FilePath: tmp.Name()}
	// type 'a' then 'b' (avoid 'h' because it triggers help)
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	if r.Buf.String() != "ab" {
		t.Fatalf("expected buffer 'ab', got %q", r.Buf.String())
	}
	// save via Ctrl+S
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModCtrl))
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(data) != "ab" {
		t.Fatalf("expected file content 'ab', got %q", string(data))
	}
	if r.Dirty {
		t.Fatalf("expected Dirty to be false after save")
	}
}

func TestRunner_BackspaceAndDelete(t *testing.T) {
	// backspace
	r := &Runner{Buf: buffer.NewGapBufferFromString("ab"), Cursor: 2}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, 0))
	if r.Buf.String() != "a" {
		t.Fatalf("expected 'a' after backspace, got %q", r.Buf.String())
	}
	if r.Cursor != 1 {
		t.Fatalf("expected cursor 1 after backspace, got %d", r.Cursor)
	}

	// delete forward
	r = &Runner{Buf: buffer.NewGapBufferFromString("ab"), Cursor: 0}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyDelete, 0, 0))
	if r.Buf.String() != "b" {
		t.Fatalf("expected 'b' after delete, got %q", r.Buf.String())
	}
}

func TestDrawFile_Highlights(t *testing.T) {
	// Use simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	text := "hello world\nhello"
	lines := strings.Split(text, "\n")
	// compute highlights for "hello"
	ranges := search.SearchAll(text, "hello")

	// draw with highlights
	drawFile(s, "f.txt", lines, ranges)

	// check first line "hello" at (0,0..4) is highlighted
	for x := 0; x < 5; x++ {
		cr, _, style, _ := s.GetContent(x, 0)
		if cr != rune("hello"[x]) {
			t.Fatalf("expected rune %q at (%d,0) got %q", rune("hello"[x]), x, cr)
		}
		expStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow)
		if style != expStyle {
			t.Fatalf("expected highlighted style at (%d,0) got %v", x, style)
		}
	}

	// check second line "hello" at (0,1..4)
	for x := 0; x < 5; x++ {
		cr, _, style, _ := s.GetContent(x, 1)
		if cr != rune("hello"[x]) {
			t.Fatalf("expected rune %q at (%d,1) got %q", rune("hello"[x]), x, cr)
		}
		expStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow)
		if style != expStyle {
			t.Fatalf("expected highlighted style at (%d,1) got %v", x, style)
		}
	}
}
