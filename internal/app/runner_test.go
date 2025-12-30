package app

import (
	"os"
	"strings"
	"testing"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"example.com/texteditor/pkg/history"
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

func TestHandleKeyEvent_RemapQuit(t *testing.T) {
	kb, err := config.ParseKeybinding("Ctrl+X")
	if err != nil {
		t.Fatalf("parse keybinding: %v", err)
	}
	r := &Runner{Keymap: config.DefaultKeymap()}
	r.Keymap["quit"] = kb

	if r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl)) {
		t.Fatalf("Ctrl+Q should not quit after remap")
	}
	if !r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModCtrl)) {
		t.Fatalf("Ctrl+X should quit after remap")
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

func TestRunner_BufferSwitch(t *testing.T) {
	f1, err := os.CreateTemp("", "buf1_*.txt")
	if err != nil {
		t.Fatalf("CreateTemp f1: %v", err)
	}
	f1.Close()
	defer os.Remove(f1.Name())

	f2, err := os.CreateTemp("", "buf2_*.txt")
	if err != nil {
		t.Fatalf("CreateTemp f2: %v", err)
	}
	if _, err := f2.WriteString("two"); err != nil {
		t.Fatalf("write f2: %v", err)
	}
	f2.Close()
	defer os.Remove(f2.Name())

	r := New()
	if err := r.LoadFile(f1.Name()); err != nil {
		t.Fatalf("LoadFile f1: %v", err)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))

	if err := r.LoadFile(f2.Name()); err != nil {
		t.Fatalf("LoadFile f2: %v", err)
	}
	if r.FilePath != f2.Name() {
		t.Fatalf("expected current file %q, got %q", f2.Name(), r.FilePath)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModCtrl))
	if r.FilePath != f1.Name() {
		t.Fatalf("expected switch back to %q, got %q", f1.Name(), r.FilePath)
	}
	if got := r.Buf.String(); got != "a" {
		t.Fatalf("expected buffer content 'a', got %q", got)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModCtrl))
	if r.FilePath != f2.Name() {
		t.Fatalf("expected switch forward to %q, got %q", f2.Name(), r.FilePath)
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

func TestWordMotionsNormalMode(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("one two"), Mode: ModeNormal}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))
	if r.Cursor != 2 {
		t.Fatalf("expected cursor at 2 after 'e', got %d", r.Cursor)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))
	if r.Cursor != 6 {
		t.Fatalf("expected cursor at 6 after second 'e', got %d", r.Cursor)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	if r.Cursor != 4 {
		t.Fatalf("expected cursor at 4 after 'b', got %d", r.Cursor)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	if r.Cursor != 0 {
		t.Fatalf("expected cursor at 0 after second 'b', got %d", r.Cursor)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'w', 0))
	if r.Cursor != 4 {
		t.Fatalf("expected cursor at 4 after 'w', got %d", r.Cursor)
	}
}

func TestWordEndVisualMode(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("one two"), Mode: ModeVisual, VisualStart: 0}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))
	if r.Cursor != 2 {
		t.Fatalf("expected cursor at 2 after 'e', got %d", r.Cursor)
	}

	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))
	if r.Cursor != 6 {
		t.Fatalf("expected cursor at 6 after second 'e', got %d", r.Cursor)
	}
}

func TestGotoTopAndBottomNormalMode(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("abc\ndef\nghi"), Cursor: 5, Mode: ModeNormal}
	r.recomputeCursorLine()
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'g', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'g', 0))
	if r.Cursor != 0 || r.CursorLine != 0 {
		t.Fatalf("expected cursor at start after gg, got %d line %d", r.Cursor, r.CursorLine)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'G', 0))
	lines := r.Buf.Lines()
	last := len(lines) - 1
	if last > 0 && len(lines[last]) == 0 {
		last--
	}
	if r.Cursor != r.Buf.Len()-1 || r.CursorLine != last {
		t.Fatalf("expected cursor at last char after G, got %d line %d", r.Cursor, r.CursorLine)
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

func TestVisualGotoTopAndBottom(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("abc\ndef\nghi"), Cursor: 5, Mode: ModeNormal}
	r.recomputeCursorLine()
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'g', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'g', 0))
	if r.Cursor != 0 || r.CursorLine != 0 {
		t.Fatalf("expected cursor at start after gg, got %d line %d", r.Cursor, r.CursorLine)
	}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'G', 0))
	lines := r.Buf.Lines()
	last := len(lines) - 1
	if last > 0 && len(lines[last]) == 0 {
		last--
	}
	if r.Cursor != r.Buf.Len()-1 || r.CursorLine != last {
		t.Fatalf("expected cursor at last char after G, got %d line %d", r.Cursor, r.CursorLine)
	}
}

func TestVisualHalfPageMovement(t *testing.T) {
	text := strings.Repeat("x\n", 30)
	r := &Runner{Buf: buffer.NewGapBufferFromString(text), Mode: ModeNormal, KillRing: history.KillRing{}}
	orig := r.Buf.String()
	r.recomputeCursorLine()
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	start := r.Cursor
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlD, 0, 0))
	if r.CursorLine != 10 {
		t.Fatalf("expected cursor line 10 after Ctrl+D, got %d", r.CursorLine)
	}
	if r.Mode != ModeVisual || r.VisualStart != start {
		t.Fatalf("expected to remain in visual mode after Ctrl+D")
	}
	r.KillRing.Set("ZZ")
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyCtrlU, 0, 0))
	if r.CursorLine != 0 {
		t.Fatalf("expected cursor line 0 after Ctrl+U, got %d", r.CursorLine)
	}
	if r.Mode != ModeVisual || r.VisualStart != start {
		t.Fatalf("expected to remain in visual mode after Ctrl+U")
	}
	if got := r.Buf.String(); got != orig {
		t.Fatalf("expected buffer unchanged after Ctrl+U, got %q", got)
	}
}

func TestNormalDeleteLineDD(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello\nworld"), KillRing: history.KillRing{}}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'd', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'd', 0))
	if got := r.Buf.String(); got != "world" {
		t.Fatalf("expected buffer 'world', got %q", got)
	}
	if kr := r.KillRing.Get(); kr != "hello\n" {
		t.Fatalf("expected kill ring to contain 'hello\\n', got %q", kr)
	}
}

func TestVisualCutX(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("hello"), KillRing: history.KillRing{}}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'l', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'l', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := r.Buf.String(); got != "lo" {
		t.Fatalf("expected buffer 'lo', got %q", got)
	}
	if data := r.KillRing.Get(); data != "hel" {
		t.Fatalf("expected kill ring 'hel', got %q", data)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode normal after cut")
	}
}

func TestNormalCutX_DeletesCharAtCursor(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("cat"), Cursor: 1, Mode: ModeNormal, KillRing: history.KillRing{}}
	// Press 'x' in normal mode to cut the character under the cursor
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := r.Buf.String(); got != "ct" {
		t.Fatalf("expected buffer 'ct', got %q", got)
	}
	if r.Cursor != 1 { // cursor stays at same index, now on 't'
		t.Fatalf("expected cursor to remain at 1 on 't', got %d", r.Cursor)
	}
	if kr := r.KillRing.Get(); kr != "a" {
		t.Fatalf("expected kill ring to contain 'a', got %q", kr)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected to remain in normal mode after 'x'")
	}
}

func TestVisualLineSelectionCut(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("abc\ndef\nghi"), KillRing: history.KillRing{}}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'V', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'j', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := r.Buf.String(); got != "ghi" {
		t.Fatalf("expected remaining text 'ghi', got %q", got)
	}
	if kr := r.KillRing.Get(); kr != "abc\ndef\n" {
		t.Fatalf("expected kill ring to contain first two lines, got %q", kr)
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
	if got := r.KillRing.Get(); got != "ell" {
		t.Fatalf("expected kill ring to contain 'ell', got %q", got)
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

func TestRunner_VisualCutWord(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBufferFromString("word"), Cursor: 0, History: history.New()}
	// Enter visual mode
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'v', 0))
	// Select to end of word
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))
	// Cut selection
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if got := r.KillRing.Get(); got != "word" {
		t.Fatalf("expected kill ring to contain 'word', got %q", got)
	}
	if got := r.Buf.String(); got != "" {
		t.Fatalf("expected buffer to be empty after cut, got %q", got)
	}
	if r.Cursor != 0 {
		t.Fatalf("expected cursor reset to start after cut, got %d", r.Cursor)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected mode to return to normal after cut")
	}
	if r.VisualStart != -1 {
		t.Fatalf("expected visual start reset after cut")
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

func TestRunner_NormalModeUndoRedoKeys(t *testing.T) {
	r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New(), Mode: ModeNormal}
	// Enter insert mode and type 'a', 'b'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	// Exit insert mode
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if got := r.Buf.String(); got != "ab" {
		t.Fatalf("expected buffer 'ab' before undo, got %q", got)
	}
	// Undo via normal-mode 'u'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'u', 0))
	if got := r.Buf.String(); got != "a" {
		t.Fatalf("expected 'a' after undo with 'u', got %q", got)
	}
	// Redo via Ctrl+R in normal mode
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModCtrl))
	if got := r.Buf.String(); got != "ab" {
		t.Fatalf("expected 'ab' after redo with Ctrl+R, got %q", got)
	}
	if r.Mode != ModeNormal {
		t.Fatalf("expected to remain in normal mode, got %v", r.Mode)
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
