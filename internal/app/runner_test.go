package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/history"
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
	// Prefer F1 which is not terminal-dependent
	ev := tcell.NewEventKey(tcell.KeyF1, 0, 0)
	if r.handleKeyEvent(ev) {
		t.Fatalf("F1 should not signal quit")
	}
	if !r.ShowHelp {
		t.Fatalf("expected ShowHelp to be set after F1")
	}
}

func TestHandleKeyEvent_ShowHelp_CtrlKey(t *testing.T) {
    r := &Runner{}
    ev := tcell.NewEventKey(tcell.KeyCtrlH, 0, 0)
    if r.handleKeyEvent(ev) {
        t.Fatalf("Ctrl+H should not signal quit")
    }
    if !r.ShowHelp {
        t.Fatalf("expected ShowHelp to be set after Ctrl+H")
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
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
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
	r := &Runner{Buf: buffer.NewGapBufferFromString("ab"), Cursor: 2, Mode: ModeInsert}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, 0))
	if r.Buf.String() != "a" {
		t.Fatalf("expected 'a' after backspace, got %q", r.Buf.String())
	}
	if r.Cursor != 1 {
		t.Fatalf("expected cursor 1 after backspace, got %d", r.Cursor)
	}

	// delete forward
	r = &Runner{Buf: buffer.NewGapBufferFromString("ab"), Cursor: 0, Mode: ModeInsert}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyDelete, 0, 0))
	if r.Buf.String() != "b" {
		t.Fatalf("expected 'b' after delete, got %q", r.Buf.String())
	}
}

func TestRunner_CursorMoveHorizontal(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("ab"), Cursor: 1}
	// Right arrow
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if r.Cursor != 2 {
		t.Fatalf("expected cursor 2 after right arrow, got %d", r.Cursor)
	}
	// Dedicated Ctrl+B key
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlB, 0, 0))
	if r.Cursor != 1 {
		t.Fatalf("expected cursor 1 after KeyCtrlB, got %d", r.Cursor)
	}
	// Dedicated Ctrl+F key
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlF, 0, 0))
	if r.Cursor != 2 {
		t.Fatalf("expected cursor 2 after KeyCtrlF, got %d", r.Cursor)
	}
	// Rune with Ctrl modifiers
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModCtrl))
	if r.Cursor != 1 {
		t.Fatalf("expected cursor 1 after Ctrl+B rune, got %d", r.Cursor)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModCtrl))
	if r.Cursor != 2 {
		t.Fatalf("expected cursor 2 after Ctrl+F rune, got %d", r.Cursor)
	}
}

func TestRunner_CursorMoveVertical(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("ab\ncde\nf"), Cursor: 1}
	// Down arrow
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if r.Cursor != 4 {
		t.Fatalf("expected cursor 4 after down arrow, got %d", r.Cursor)
	}
	// Dedicated Ctrl+N key
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlN, 0, 0))
	if r.Cursor != 8 {
		t.Fatalf("expected cursor 8 after KeyCtrlN, got %d", r.Cursor)
	}
	// Up arrow
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if r.Cursor != 4 {
		t.Fatalf("expected cursor 4 after up arrow, got %d", r.Cursor)
	}
	// Dedicated Ctrl+P key
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlP, 0, 0))
	if r.Cursor != 1 {
		t.Fatalf("expected cursor 1 after KeyCtrlP, got %d", r.Cursor)
	}
	// Rune with Ctrl modifiers
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'n', tcell.ModCtrl))
	if r.Cursor != 4 {
		t.Fatalf("expected cursor 4 after Ctrl+N rune, got %d", r.Cursor)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'p', tcell.ModCtrl))
	if r.Cursor != 1 {
		t.Fatalf("expected cursor 1 after Ctrl+P rune, got %d", r.Cursor)
	}
}

func TestModeTransitions(t *testing.T) {
	r := &Runner{}
	if r.Mode != ModeNormal {
		t.Fatalf("expected initial mode normal")
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	if r.Mode != ModeInsert {
		t.Fatalf("expected mode insert after 'i'")
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode normal after Esc")
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	if r.Mode != ModeVisual {
		t.Fatalf("expected mode visual after 'v'")
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode normal after Esc from visual")
	}
}

func TestOpenLineFromNormalMode(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello"), Mode: ModeNormal}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'o', 0))
	if got := r.Buf.String(); got != "hello\n" {
		t.Fatalf("expected buffer 'hello\\n' after open line, got %q", got)
	}
	if r.Mode != ModeInsert {
		t.Fatalf("expected mode insert after 'o', got %v", r.Mode)
	}
	if r.Cursor != len("hello\n") {
		t.Fatalf("expected cursor at end of new line, got %d", r.Cursor)
	}
}

func TestLineNavigationNormalMode(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("abc\ndef"), Cursor: 1, Mode: ModeNormal}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '$', 0))
	if r.Cursor != 3 {
		t.Fatalf("expected cursor at end of line, got %d", r.Cursor)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '0', 0))
	if r.Cursor != 0 {
		t.Fatalf("expected cursor at start of line, got %d", r.Cursor)
	}
}

func TestLineNavigationVisualMode(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("abc\ndef"), Cursor: 1, Mode: ModeNormal}
	// enter visual mode
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '$', 0))
	if r.Cursor != 3 {
		t.Fatalf("expected cursor at end of line in visual mode, got %d", r.Cursor)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, '0', 0))
	if r.Cursor != 0 {
		t.Fatalf("expected cursor at start of line in visual mode, got %d", r.Cursor)
	}
}

func TestVisualCutX(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello"), KillRing: history.KillRing{}}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'l', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'l', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := r.Buf.String(); got != "llo" {
		t.Fatalf("expected buffer 'llo', got %q", got)
	}
	if data := r.KillRing.Get(); data != "he" {
		t.Fatalf("expected kill ring 'he', got %q", data)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode normal after cut")
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
	drawFile(s, "f.txt", lines, ranges, -1, false, ModeInsert)

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

func TestDrawBuffer_DirtyIndicator(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("hello")
	drawBuffer(s, buf, "f.txt", nil, 0, true, ModeInsert)

	_, height := s.Size()
	expected := "f.txt [+] — Press Ctrl+Q to exit"
	for i, r := range expected {
		cr, _, _, _ := s.GetContent(i, height-1)
		if cr != r {
			t.Fatalf("expected status %q, got mismatch at %d: %q", expected, i, string(cr))
		}
	}
}

func TestRunner_KillToEndOfLineAndYank(t *testing.T) {
	tests := []struct {
		name string
		kill *tcell.EventKey
		yank *tcell.EventKey
	}{
		{
			name: "rune_ctrl",
			kill: tcell.NewEventKey(tcell.KeyRune, 'k', tcell.ModCtrl),
			yank: tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModCtrl),
		},
		{
			name: "dedicated_ctrl",
			kill: tcell.NewEventKey(tcell.KeyCtrlK, 0, 0),
			yank: tcell.NewEventKey(tcell.KeyCtrlY, 0, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{Buf: buffer.NewGapBufferFromString("one\ntwo\n"), Cursor: 1, History: history.New(), Mode: ModeInsert}
			// Cut from cursor to end of line
			r.handleKeyEvent(tt.kill)
			if got := r.Buf.String(); got != "o\ntwo\n" {
				t.Fatalf("expected buffer 'o\\ntwo\\n' after kill, got %q", got)
			}
			if !r.KillRing.HasData() || r.KillRing.Get() != "ne" {
				t.Fatalf("expected kill ring to contain 'ne', got %q", r.KillRing.Get())
			}
			// Yank back
			r.handleKeyEvent(tt.yank)
			if got := r.Buf.String(); got != "one\ntwo\n" {
				t.Fatalf("expected buffer restored to 'one\\ntwo\\n' after yank, got %q", got)
			}
		})
	}
}

func TestRunner_CtrlACtrlE(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello\n"), Cursor: 2, Mode: ModeInsert}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModCtrl))
	if r.Cursor != 0 {
		t.Fatalf("expected cursor at start of line, got %d", r.Cursor)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModCtrl))
	if r.Cursor != 5 {
		t.Fatalf("expected cursor at end of line, got %d", r.Cursor)
	}
}

func TestRunner_VisualYank(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello"), Cursor: 1, History: history.New()}
	// Enter visual mode
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	// Extend selection to include "el"
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	// Yank selection
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'y', 0))
	if got := r.KillRing.Get(); got != "el" {
		t.Fatalf("expected kill ring to contain 'el', got %q", got)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode to return to normal after yank")
	}
	if r.VisualStart != -1 {
		t.Fatalf("expected visual start reset after yank")
	}
	if r.Buf.String() != "hello" {
		t.Fatalf("buffer should remain unchanged after yank, got %q", r.Buf.String())
	}
	if r.Cursor != 1 {
		t.Fatalf("expected cursor at start of selection after yank, got %d", r.Cursor)
	}
}

func TestRunner_NormalPaste(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello"), Cursor: 1, History: history.New()}
	r.KillRing.Set("XY")
	// Paste at cursor in normal mode
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
	if got := r.Buf.String(); got != "hXYello" {
		t.Fatalf("expected buffer 'hXYello' after paste, got %q", got)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode to remain normal after paste")
	}
	if r.Cursor != 1+len("XY") {
		t.Fatalf("expected cursor at %d after paste, got %d", 1+len("XY"), r.Cursor)
	}
	if r.KillRing.Get() != "XY" {
		t.Fatalf("expected kill ring to remain unchanged after paste")
	}
}

func TestNormalModeAppend(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("ab"), Cursor: 0}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	if r.Mode != ModeInsert {
		t.Fatalf("expected insert mode after 'a'")
	}
	if r.Cursor != 1 {
		t.Fatalf("expected cursor at 1 after 'a', got %d", r.Cursor)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'X', 0))
	if got := r.Buf.String(); got != "aXb" {
		t.Fatalf("expected buffer 'aXb', got %q", got)
	}
}

func TestVisualModeOpenLine(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello"), Cursor: 0}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'o', 0))
	if r.Mode != ModeInsert {
		t.Fatalf("expected insert mode after 'o'")
	}
	if r.VisualStart != -1 {
		t.Fatalf("expected visual start reset after 'o'")
	}
	if got := r.Buf.String(); got != "hello\n" {
		t.Fatalf("expected buffer 'hello\\n', got %q", got)
	}
	if r.Cursor != len("hello\n") {
		t.Fatalf("expected cursor at %d after open line, got %d", len("hello\n"), r.Cursor)
	}
}

func TestRunner_UndoRedo(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	// type 'a', 'b'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	if got := r.Buf.String(); got != "ab" {
		t.Fatalf("expected 'ab', got %q", got)
	}
	// undo -> 'a'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModCtrl))
	if got := r.Buf.String(); got != "a" {
		t.Fatalf("expected 'a' after undo, got %q", got)
	}
	// undo -> ''
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModCtrl))
	if got := r.Buf.String(); got != "" {
		t.Fatalf("expected '' after second undo, got %q", got)
	}
	// exit insert mode for redo
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	// redo -> 'a'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModCtrl))
	if got := r.Buf.String(); got != "a" {
		t.Fatalf("expected 'a' after redo, got %q", got)
	}
	// redo -> 'ab'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModCtrl))
	if got := r.Buf.String(); got != "ab" {
		t.Fatalf("expected 'ab' after second redo, got %q", got)
	}
}

func TestRunner_Undo_DedicatedCtrlZ(t *testing.T) {
    r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
    r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
    // type 'a', 'b'
    r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
    r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
    if got := r.Buf.String(); got != "ab" {
        t.Fatalf("expected 'ab', got %q", got)
    }
    // undo via dedicated Ctrl+Z key -> 'a'
    r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
    if got := r.Buf.String(); got != "a" {
        t.Fatalf("expected 'a' after undo via KeyCtrlZ, got %q", got)
    }
}

// TestRun_TypingSaveQuit_Simulation reflects README behavior: type directly, save with Ctrl+S, quit with Ctrl+Q.
func TestRun_TypingSaveQuit_Simulation(t *testing.T) {
	// Set up simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	// Temp file for save
	tmp, err := os.CreateTemp("", "texteditor_run_*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New(), FilePath: tmp.Name()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// Give the loop a moment to start
	time.Sleep(10 * time.Millisecond)

	// Enter insert mode and type 'a', 'b' (avoid 'h' which opens help per README)
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	// Save via Ctrl+S
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModCtrl))
	// Dismiss save dialog
	s.PostEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// Quit via Ctrl+Q (use rune+ctrl for portability)
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	// Verify buffer content and saved file per README expectations
	if got := r.Buf.String(); got != "ab" {
		t.Fatalf("expected buffer 'ab', got %q", got)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "ab" {
		t.Fatalf("expected saved content 'ab', got %q", string(data))
	}
	if r.Dirty {
		t.Fatalf("expected Dirty=false after save")
	}
}

func TestRunner_LoadFile_NormalizesCRLF(t *testing.T) {
	f, err := os.CreateTemp("", "texteditor_crlf_*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	path := f.Name()
	_, _ = f.WriteString("a\r\nb\r\n")
	_ = f.Close()
	defer os.Remove(path)

	r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
	if err := r.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	t.Logf("LoadFile OK: path=%s, buf_len=%d", path, r.Buf.Len())
	if got := r.Buf.String(); got != "a\nb\n" {
		t.Fatalf("expected normalized newlines, got %q", got)
	}
}

func TestRunner_SaveAs_WritesAndClearsDirty(t *testing.T) {
	f, err := os.CreateTemp("", "texteditor_saveas_*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	path := f.Name()
	_ = f.Close()
	defer os.Remove(path)

	r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	// type 'a','b'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	if !r.Dirty {
		t.Fatalf("expected Dirty after typing")
	}
	if err := r.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved: %v", err)
	}
	if string(data) != "ab" {
		t.Fatalf("expected 'ab', got %q", string(data))
	}
	if r.Dirty {
		t.Fatalf("expected Dirty=false after save")
	}
	if r.FilePath != path {
		t.Fatalf("expected FilePath %q, got %q", path, r.FilePath)
	}
}

// TestRun_OpenFilePrompt_Simulation verifies that pressing Ctrl+O opens the
// prompt; typing a valid path and pressing Enter loads the file into the buffer.
func TestRun_OpenFilePrompt_Simulation(t *testing.T) {
	// Prepare a temp file to open
	tf, err := os.CreateTemp("", "texteditor_open_*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	path := tf.Name()
	content := "hello\nworld\n"
	if _, err := tf.WriteString(content); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	_ = tf.Close()
	defer os.Remove(path)

	// Set up simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// Give the loop a moment to start
	time.Sleep(10 * time.Millisecond)

	// Open prompt via dedicated control key (as emitted in real logs)
	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlO, 0, 0))
	// Type the path and press Enter
	for _, ch := range path {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// Quit to end the loop
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	// Assert file got loaded (not raw typed path into buffer)
	if r.FilePath != path {
		t.Fatalf("expected FilePath=%q after open, got %q", path, r.FilePath)
	}
	if got := r.Buf.String(); got != content {
		t.Fatalf("expected buffer to equal file content, got %q", got)
	}
}

// TestRun_SearchPrompt_Simulation verifies that the search prompt moves the cursor
// to the first match of the query when Enter is pressed.
func TestRun_SearchPrompt_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("hello world hello")
	r := &Runner{Screen: s, Buf: buf, History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// allow event loop to start
	time.Sleep(10 * time.Millisecond)

	// open search prompt via Ctrl+W
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModCtrl))
	// type query
	for _, ch := range "world" {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}
	// accept search
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// quit editor
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	expected := len([]rune("hello "))
	if r.Cursor != expected {
		t.Fatalf("expected cursor %d after search, got %d", expected, r.Cursor)
	}
	if got := r.Buf.String(); got != "hello world hello" {
		t.Fatalf("buffer modified during search: %q", got)
	}
}

// TestRun_SearchPrompt_CtrlKey_Simulation verifies the search prompt also opens
// when the terminal emits the dedicated Ctrl+W key (as seen in real logs), and
// that it moves the cursor on Enter.
func TestRun_SearchPrompt_CtrlKey_Simulation(t *testing.T) {
    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil {
        t.Fatalf("init sim screen: %v", err)
    }
    defer s.Fini()

    buf := buffer.NewGapBufferFromString("hello world hello")
    r := &Runner{Screen: s, Buf: buf, History: history.New()}

    done := make(chan error, 1)
    go func() { done <- r.Run() }()

    // allow event loop to start
    time.Sleep(10 * time.Millisecond)

    // open search prompt via dedicated Ctrl+W key
    s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlW, 0, 0))
    // type query
    for _, ch := range "world" {
        s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ch, 0))
    }
    // accept search
    s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
    // quit editor
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

    select {
    case err := <-done:
        if err != nil {
            t.Fatalf("runner returned error: %v", err)
        }
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for runner to quit")
    }

    expected := len([]rune("hello "))
    if r.Cursor != expected {
        t.Fatalf("expected cursor %d after search, got %d", expected, r.Cursor)
    }
    if got := r.Buf.String(); got != "hello world hello" {
        t.Fatalf("buffer modified during search: %q", got)
    }
}

// TestRun_GoToPrompt_Simulation verifies that the go-to prompt jumps to the specified line.
func TestRun_GoToPrompt_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	content := "one\ntwo\nthree\n"
	r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString(content), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	time.Sleep(10 * time.Millisecond)

	// open go-to prompt via Alt+G
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModAlt))
	// type line number
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, '3', 0))
	// accept
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// quit editor
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	expected := len([]rune("one\ntwo\n"))
	if r.Cursor != expected {
		t.Fatalf("expected cursor %d after go-to, got %d", expected, r.Cursor)
	}
}
