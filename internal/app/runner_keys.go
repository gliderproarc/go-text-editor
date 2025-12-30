package app

import (
	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"github.com/gdamore/tcell/v2"
)

// handleKeyEvent processes a key event. It returns true if the event signals
// the runner should quit.
func (r *Runner) handleKeyEvent(ev *tcell.EventKey) bool {
	switch r.Mode {
	case ModeNormal:
		if r.PendingG && !(ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0) {
			r.PendingG = false
		}
		if r.PendingD && !(ev.Key() == tcell.KeyRune && ev.Rune() == 'd' && ev.Modifiers() == 0) {
			r.PendingD = false
		}
	case ModeInsert:
		r.PendingG = false
		r.PendingD = false
	}
	// Mode transitions similar to Vim
	if ev.Key() == tcell.KeyEsc {
		switch r.Mode {
		case ModeInsert:
			r.Mode = ModeNormal
			r.draw(nil)
			return false
		case ModeVisual:
			r.Mode = ModeNormal
			r.VisualStart = -1
			r.VisualLine = false
			r.PendingG = false
			r.draw(nil)
			return false
		default:
			return false
		}
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		switch ev.Rune() {
		case 'i':
			r.Mode = ModeInsert
			r.draw(nil)
			return false
		case 'a':
			if r.Buf != nil && r.Cursor < r.Buf.Len() {
				if r.Buf.RuneAt(r.Cursor) == '\n' {
					r.CursorLine++
				}
				r.Cursor++
			}
			r.Mode = ModeInsert
			r.draw(nil)
			return false
		case 'u':
			r.performUndo("undo.normal")
			return false
		case 'v':
			r.Mode = ModeVisual
			r.VisualStart = r.Cursor
			r.VisualLine = false
			r.draw(nil)
			return false
		case 'V':
			if r.Buf != nil {
				start, _ := r.currentLineBounds()
				r.Cursor = start
				r.VisualStart = start
			} else {
				r.VisualStart = r.Cursor
			}
			r.Mode = ModeVisual
			r.VisualLine = true
			r.draw(nil)
			return false
		case 'p':
			if r.KillRing.HasData() {
				text := r.KillRing.Get()
				r.insertText(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "paste.normal", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			}
			return false
		case 'o':
			if r.Buf != nil {
				start, end := r.currentLineBounds()
				// Insert the new line before the existing newline (if present)
				// so the cursor lands on exactly one new line below, not past it.
				pos := end
				if end > start && r.Buf.RuneAt(end-1) == '\n' {
					pos = end - 1
				}
				r.Cursor = pos
				r.insertText("\n")
			}
			r.Mode = ModeInsert
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'd':
			if r.PendingD {
				r.PendingD = false
				if r.Buf != nil {
					start, end := r.currentLineBounds()
					if end > start {
						text := string(r.Buf.Slice(start, end))
						_ = r.deleteRange(start, end, text)
						r.KillRing.Set(text)
						r.recomputeCursorLine()
						if r.Logger != nil {
							r.Logger.Event("action", map[string]any{"name": "delete.line", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
						}
						if r.Screen != nil {
							r.draw(nil)
						}
					}
				}
			} else {
				r.PendingD = true
			}
			return false
		case 'x':
			// Cut the character at the cursor in normal mode
			if r.Buf != nil && r.Cursor < r.Buf.Len() {
				start := r.Cursor
				end := start + 1
				text := string(r.Buf.Slice(start, end))
				_ = r.deleteRange(start, end, text)
				r.KillRing.Set(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "cut.normal", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			}
			return false
		case 'G':
			if r.Buf != nil && r.Buf.Len() > 0 {
				r.Cursor = r.Buf.Len() - 1
				lines := r.Buf.Lines()
				last := len(lines) - 1
				if last > 0 && len(lines[last]) == 0 {
					last--
				}
				r.CursorLine = last
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'w':
			if r.Buf != nil {
				r.Cursor = buffer.NextWordStart(r.Buf, r.Cursor)
				r.recomputeCursorLine()
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'b':
			if r.Buf != nil {
				r.Cursor = buffer.WordStart(r.Buf, r.Cursor)
				r.recomputeCursorLine()
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'e':
			if r.Buf != nil {
				r.Cursor = buffer.WordEnd(r.Buf, r.Cursor)
				r.recomputeCursorLine()
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'g':
			if r.PendingG {
				r.PendingG = false
				if r.Buf != nil {
					r.Cursor = 0
					r.CursorLine = 0
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			} else {
				r.PendingG = true
			}
			return false
		case '$':
			if r.Buf != nil {
				_, end := r.currentLineBounds()
				if end > 0 && r.Buf.RuneAt(end-1) == '\n' {
					end--
				}
				r.Cursor = end
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case '0':
			if r.Buf != nil {
				start, _ := r.currentLineBounds()
				r.Cursor = start
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case ' ':
			return r.runMnemonicMenu()
		}
	}
	if r.Mode == ModeInsert && ev.Key() == tcell.KeyRune && ev.Rune() == 'm' && ev.Modifiers() == tcell.ModAlt {
		return r.runMnemonicMenu()
	}
	if r.Mode == ModeVisual && ev.Key() == tcell.KeyRune && ev.Rune() == 'v' && ev.Modifiers() == 0 {
		r.Mode = ModeNormal
		r.VisualStart = -1
		r.VisualLine = false
		r.PendingG = false
		r.draw(nil)
		return false
	}
	// Command keybindings
	if r.matchCommand(ev, "quit") {
		if r.Dirty {
			return r.runQuitPrompt()
		}
		return true
	}
	if r.matchCommand(ev, "save") {
		if r.FilePath == "" {
			r.runSaveAsPrompt()
		} else {
			if err := r.Save(); err == nil {
				r.showDialog("Saved " + r.FilePath)
			}
		}
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "save", "file": r.FilePath})
		}
		return false
	}
	if r.matchCommand(ev, "search") {
		r.runSearchPrompt()
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "search.prompt"})
		}
		return false
	}
	if r.matchCommand(ev, "menu") {
		return r.runCommandMenu()
	}
	// Ctrl+O -> open file prompt (handle both rune+Ctrl and dedicated control key)
	if (ev.Key() == tcell.KeyRune && ev.Rune() == 'o' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlO {
		r.runOpenPrompt()
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "open.prompt"})
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	// Ctrl+PgUp/PgDn -> switch buffers
	if ev.Key() == tcell.KeyPgUp && ev.Modifiers() == tcell.ModCtrl {
		r.saveBufferState()
		bs := r.Ed.Prev()
		r.FilePath, r.Buf, r.Cursor, r.Dirty = bs.FilePath, bs.Buf, bs.Cursor, bs.Dirty
		r.recomputeCursorLine()
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if ev.Key() == tcell.KeyPgDn && ev.Modifiers() == tcell.ModCtrl {
		r.saveBufferState()
		bs := r.Ed.Next()
		r.FilePath, r.Buf, r.Cursor, r.Dirty = bs.FilePath, bs.Buf, bs.Cursor, bs.Dirty
		r.recomputeCursorLine()
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	// Ctrl+Z -> undo (handle both rune+Ctrl and dedicated control key)
	if (ev.Key() == tcell.KeyRune && ev.Rune() == 'z' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlZ {
		r.performUndo("undo")
		return false
	}
	// Ctrl+Y -> yank in insert mode, redo otherwise
	if (ev.Key() == tcell.KeyRune && ev.Rune() == 'y' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlY {
		if r.Mode == ModeInsert {
			if r.KillRing.HasData() {
				text := r.KillRing.Get()
				r.insertText(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "yank", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			}
		} else {
			r.performRedo("redo")
		}
		return false
	}
	// Ctrl+R -> redo in normal/visual modes
	if (ev.Key() == tcell.KeyRune && ev.Rune() == 'r' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlR {
		if r.Mode != ModeInsert {
			r.performRedo("redo.ctrlr")
		}
		return false
	}
	// Alt+G -> go-to line (Alt modifier)
	if ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == tcell.ModAlt {
		r.runGoToPrompt()
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "goto.prompt"})
		}
		return false
	}
	// Show help on F1 or Ctrl+H (support dedicated control key)
	if ev.Key() == tcell.KeyF1 || (ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == tcell.ModCtrl) || (ev.Key() == tcell.KeyCtrlH && r.Mode != ModeInsert) {
		r.ShowHelp = true
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "help.show"})
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}

	// Half-page down/up in normal mode (Ctrl+D / Ctrl+U)
	if r.Mode != ModeInsert {
		// Ctrl+D
		if (ev.Key() == tcell.KeyRune && ev.Rune() == 'd' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlD {
			lines := 10
			if r.Screen != nil {
				_, h := r.Screen.Size()
				if h > 0 {
					lines = h / 2
				}
			}
			r.moveCursorVertical(lines)
			r.recomputeCursorLine()
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		}
		// Ctrl+U
		if (ev.Key() == tcell.KeyRune && ev.Rune() == 'u' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlU {
			lines := 10
			if r.Screen != nil {
				_, h := r.Screen.Size()
				if h > 0 {
					lines = h / 2
				}
			}
			r.moveCursorVertical(-lines)
			r.recomputeCursorLine()
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		}
	}

	if r.Mode == ModeVisual {
		return r.handleVisualKey(ev)
	}

	// Ctrl+A/Ctrl+E in insert mode -> line start/end
	if r.Mode == ModeInsert && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'a' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlA) {
		start, _ := r.currentLineBounds()
		r.Cursor = start
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if r.Mode == ModeInsert && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'e' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlE) {
		_, end := r.currentLineBounds()
		if end > 0 && r.Buf.RuneAt(end-1) == '\n' {
			end--
		}
		r.Cursor = end
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}

	// Arrow keys and basic cursor movement (Ctrl+B/F for left/right, Ctrl+P/N for up/down, hjkl in normal mode)
	if ev.Key() == tcell.KeyLeft || ev.Key() == tcell.KeyCtrlB || (ev.Key() == tcell.KeyRune && ev.Rune() == 'b' && ev.Modifiers() == tcell.ModCtrl) || (r.Mode != ModeInsert && ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == 0) {
		if r.Cursor > 0 {
			if r.Buf != nil && r.Buf.RuneAt(r.Cursor-1) == '\n' {
				r.CursorLine--
				if r.CursorLine < 0 {
					r.CursorLine = 0
				}
			}
			r.Cursor--
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if ev.Key() == tcell.KeyRight || ev.Key() == tcell.KeyCtrlF || (ev.Key() == tcell.KeyRune && ev.Rune() == 'f' && ev.Modifiers() == tcell.ModCtrl) || (r.Mode != ModeInsert && ev.Key() == tcell.KeyRune && ev.Rune() == 'l' && ev.Modifiers() == 0) {
		if r.Buf != nil && r.Cursor < r.Buf.Len() {
			if r.Buf.RuneAt(r.Cursor) == '\n' {
				r.CursorLine++
			}
			r.Cursor++
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if ev.Key() == tcell.KeyUp || ev.Key() == tcell.KeyCtrlP || (ev.Key() == tcell.KeyRune && ev.Rune() == 'p' && ev.Modifiers() == tcell.ModCtrl) || (r.Mode != ModeInsert && ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == 0) {
		r.moveCursorVertical(-1)
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if ev.Key() == tcell.KeyDown || ev.Key() == tcell.KeyCtrlN || (ev.Key() == tcell.KeyRune && ev.Rune() == 'n' && ev.Modifiers() == tcell.ModCtrl) || (r.Mode != ModeInsert && ev.Key() == tcell.KeyRune && ev.Rune() == 'j' && ev.Modifiers() == 0) {
		r.moveCursorVertical(1)
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}

	// Insert typed rune only in insert mode
	if r.Mode == ModeInsert && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		text := string(ev.Rune())
		r.insertText(text)
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "insert", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}

	// Backspace
	if r.Mode == ModeInsert && (ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2) {
		if r.Cursor > 0 {
			// capture deleted rune
			del := string(r.Buf.Slice(r.Cursor-1, r.Cursor))
			_ = r.deleteRange(r.Cursor-1, r.Cursor, del)
			if r.Logger != nil {
				r.Logger.Event("action", map[string]any{"name": "backspace", "deleted": del, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
			}
			if r.Screen != nil {
				r.draw(nil)
			}
		}
		return false
	}

	// Delete (forward)
	if r.Mode == ModeInsert && ev.Key() == tcell.KeyDelete {
		if r.Cursor < r.Buf.Len() {
			del := string(r.Buf.Slice(r.Cursor, r.Cursor+1))
			_ = r.deleteRange(r.Cursor, r.Cursor+1, del)
			if r.Logger != nil {
				r.Logger.Event("action", map[string]any{"name": "delete", "deleted": del, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
			}
			if r.Screen != nil {
				r.draw(nil)
			}
		}
		return false
	}

	// Enter -> newline
	if r.Mode == ModeInsert && ev.Key() == tcell.KeyEnter {
		r.insertText("\n")
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "newline", "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}

	// Ctrl+K -> kill (cut) from cursor to end of line in insert mode
	if r.Mode == ModeInsert && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlK) {
		start := r.Cursor
		_, lineEnd := r.currentLineBounds()
		end := lineEnd
		if start < end {
			if r.Buf.RuneAt(start) == '\n' {
				end = start + 1
			} else if end > start && r.Buf.RuneAt(end-1) == '\n' {
				end = end - 1
			}
			if end > start {
				text := string(r.Buf.Slice(start, end))
				_ = r.deleteRange(start, end, text)
				r.KillRing.Set(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "cut.insert", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			}
		}
		return false
	}
	// Ctrl+U -> yank (paste) from kill ring in insert mode
	if r.Mode == ModeInsert && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'u' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlU) {
		if r.KillRing.HasData() {
			text := r.KillRing.Get()
			r.insertText(text)
			if r.Logger != nil {
				r.Logger.Event("action", map[string]any{"name": "yank", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
			}
			if r.Screen != nil {
				r.draw(nil)
			}
		}
		return false
	}

	return false
}

// handleVisualKey processes key events while in visual mode.
func (r *Runner) handleVisualKey(ev *tcell.EventKey) bool {
	if r.PendingG && !(ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0) {
		r.PendingG = false
	}
	switch {
	case r.PendingG && ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0:
		r.PendingG = false
		if r.Buf != nil {
			r.Cursor = 0
			r.CursorLine = 0
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0:
		r.PendingG = true
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'G':
		if r.Buf != nil && r.Buf.Len() > 0 {
			r.Cursor = r.Buf.Len() - 1
			lines := r.Buf.Lines()
			last := len(lines) - 1
			if last > 0 && len(lines[last]) == 0 {
				last--
			}
			r.CursorLine = last
		}
		r.draw(nil)
		return false
	case (ev.Key() == tcell.KeyRune && ev.Rune() == 'd' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlD:
		lines := 10
		if r.Screen != nil {
			_, h := r.Screen.Size()
			if h > 0 {
				lines = h / 2
			}
		}
		r.moveCursorVertical(lines)
		r.recomputeCursorLine()
		r.draw(nil)
		return false
	case (ev.Key() == tcell.KeyRune && ev.Rune() == 'u' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlU:
		lines := 10
		if r.Screen != nil {
			_, h := r.Screen.Size()
			if h > 0 {
				lines = h / 2
			}
		}
		r.moveCursorVertical(-lines)
		r.recomputeCursorLine()
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyLeft || (ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == 0):
		if r.Cursor > 0 {
			if r.Buf != nil && r.Buf.RuneAt(r.Cursor-1) == '\n' {
				r.CursorLine--
				if r.CursorLine < 0 {
					r.CursorLine = 0
				}
			}
			r.Cursor--
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRight || (ev.Key() == tcell.KeyRune && ev.Rune() == 'l' && ev.Modifiers() == 0):
		if r.Buf != nil && r.Cursor < r.Buf.Len() {
			if r.Buf.RuneAt(r.Cursor) == '\n' {
				r.CursorLine++
			}
			r.Cursor++
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'w' && ev.Modifiers() == 0:
		if r.Buf != nil {
			r.Cursor = buffer.NextWordStart(r.Buf, r.Cursor)
			r.recomputeCursorLine()
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'b' && ev.Modifiers() == 0:
		if r.Buf != nil {
			r.Cursor = buffer.WordStart(r.Buf, r.Cursor)
			r.recomputeCursorLine()
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'e' && ev.Modifiers() == 0:
		if r.Buf != nil {
			r.Cursor = buffer.WordEnd(r.Buf, r.Cursor)
			r.recomputeCursorLine()
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyUp || (ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == 0):
		r.moveCursorVertical(-1)
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyDown || (ev.Key() == tcell.KeyRune && ev.Rune() == 'j' && ev.Modifiers() == 0):
		r.moveCursorVertical(1)
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == '$' && ev.Modifiers() == 0:
		if r.Buf != nil {
			_, end := r.currentLineBounds()
			if end > 0 && r.Buf.RuneAt(end-1) == '\n' {
				end--
			}
			r.Cursor = end
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == '0' && ev.Modifiers() == 0:
		if r.Buf != nil {
			start, _ := r.currentLineBounds()
			r.Cursor = start
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'o' && ev.Modifiers() == 0:
		if r.Buf != nil {
			start, end := r.currentLineBounds()
			pos := end
			if end > start && r.Buf.RuneAt(end-1) == '\n' {
				pos = end - 1
			}
			r.Cursor = pos
			r.insertText("\n")
		}
		r.Mode = ModeInsert
		r.VisualStart = -1
		r.VisualLine = false
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'y' && ev.Modifiers() == 0:
		if r.Buf != nil {
			start, end := r.visualSelectionBounds()
			if start < end {
				text := string(r.Buf.Slice(start, end))
				r.KillRing.Set(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "yank.visual", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				r.Cursor = start
			}
		}
		r.Mode = ModeNormal
		r.VisualStart = -1
		r.VisualLine = false
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'x' && ev.Modifiers() == 0:
		if r.Buf != nil {
			start, end := r.visualSelectionBounds()
			if start < end {
				text := string(r.Buf.Slice(start, end))
				_ = r.deleteRange(start, end, text)
				r.KillRing.Set(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "cut.visual", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				r.Cursor = start
			}
		}
		r.Mode = ModeNormal
		r.VisualStart = -1
		r.VisualLine = false
		r.draw(nil)
		return false
	}
	// ignore other keys in visual mode
	return false
}

func (r *Runner) matchCommand(ev *tcell.EventKey, name string) bool {
	if r.Keymap == nil {
		r.Keymap = config.DefaultKeymap()
	}
	kb, ok := r.Keymap[name]
	if !ok {
		return false
	}
	return kb.Matches(ev)
}

func (r *Runner) performUndo(action string) {
	if r.History == nil {
		return
	}
	if err := r.History.Undo(r.Buf, &r.Cursor); err == nil {
		r.editSeq++
	}
	r.recomputeCursorLine()
	r.Dirty = true
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": action, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
}

func (r *Runner) performRedo(action string) {
	if r.History == nil {
		return
	}
	if err := r.History.Redo(r.Buf, &r.Cursor); err == nil {
		r.editSeq++
	}
	r.recomputeCursorLine()
	r.Dirty = true
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": action, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
}
