package app

import (
	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"github.com/gdamore/tcell/v2"
	"unicode"
)

// handleKeyEvent processes a key event. It returns true if the event signals
// the runner should quit.
func (r *Runner) handleKeyEvent(ev *tcell.EventKey) bool {
	switch r.Mode {
	case ModeNormal:
		if r.PendingG && !(ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0) {
			r.PendingG = false
		}
		if r.PendingTextObject {
			if ev.Key() != tcell.KeyRune || ev.Modifiers() != 0 || !isTextObjectDelimiter(ev.Rune()) {
				r.PendingTextObject = false
				r.TextObjectAround = false
				r.PendingD = false
				r.PendingC = false
				r.PendingY = false
			}
		}
		if r.PendingD {
			if !r.PendingTextObject {
				if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
					if ev.Rune() != 'd' && ev.Rune() != 'w' && ev.Rune() != 'i' && ev.Rune() != 'a' && !unicode.IsDigit(ev.Rune()) {
						r.PendingD = false
					}
				} else {
					r.PendingD = false
				}
			}
		}
		if r.PendingC {
			if !r.PendingTextObject {
				if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
					if ev.Rune() != 'w' && ev.Rune() != 'i' && ev.Rune() != 'a' && !unicode.IsDigit(ev.Rune()) {
						r.PendingC = false
					}
				} else {
					r.PendingC = false
				}
			}
		}
		if r.PendingY {
			if !r.PendingTextObject {
				if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
					if ev.Rune() != 'y' && ev.Rune() != 'i' && ev.Rune() != 'a' && !unicode.IsDigit(ev.Rune()) {
						r.PendingY = false
					}
				} else {
					r.PendingY = false
				}
			}
		}
	case ModeInsert, ModeMultiEdit:
		r.PendingG = false
		r.PendingD = false
		r.PendingY = false
		r.PendingC = false
		r.PendingTextObject = false
		r.TextObjectAround = false
	}
	// Count prefixes in normal mode (digits)
	if r.handleMacroPending(ev) {
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		if ev.Rune() >= '1' && ev.Rune() <= '9' {
			r.PendingCount = r.PendingCount*10 + int(ev.Rune()-'0')
			return false
		}
		if ev.Rune() == '0' && r.PendingCount > 0 {
			r.PendingCount = r.PendingCount * 10
			return false
		}
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		if (r.PendingD || r.PendingC || r.PendingY) && (ev.Rune() == 'i' || ev.Rune() == 'a') {
			r.PendingTextObject = true
			r.TextObjectAround = ev.Rune() == 'a'
			return false
		}
		if r.PendingTextObject && isTextObjectDelimiter(ev.Rune()) {
			count := r.consumeCount()
			delim := ev.Rune()
			around := r.TextObjectAround
			r.PendingTextObject = false
			r.TextObjectAround = false
			switch {
			case r.PendingD:
				r.PendingD = false
				r.deleteTextObject(delim, around, count)
				return false
			case r.PendingC:
				r.PendingC = false
				r.deleteTextObject(delim, around, count)
				r.Mode = ModeInsert
				r.beginInsertCapture(count, func(text string, c int) {
					r.setLastChange(func(times int) {
						r.changeTextObject(delim, around, times, text)
					}, c)
				})
				if r.Screen != nil {
					r.draw(nil)
				}
				return false
			case r.PendingY:
				r.PendingY = false
				r.yankTextObject(delim, around, count)
				return false
			}
		}
	}
	// Mode transitions similar to Vim
	if r.isCancelKey(ev) {
		switch r.Mode {
		case ModeInsert:
			r.finalizeInsertCapture()
			r.Mode = ModeNormal
			r.PendingY = false
			r.PendingTextObject = false
			r.TextObjectAround = false
			r.PendingCount = 0
			r.draw(nil)
			return false
		case ModeMultiEdit:
			r.exitMultiEdit()
			return false
		case ModeVisual:
			r.Mode = ModeNormal
			r.VisualStart = -1
			r.VisualLine = false
			r.PendingG = false
			r.PendingY = false
			r.PendingTextObject = false
			r.TextObjectAround = false
			r.PendingCount = 0
			r.draw(nil)
			return false
		default:
			return false
		}
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		switch ev.Rune() {
		case 'q':
			if r.macroRecording {
				r.stopMacroRecording()
				if r.Screen != nil {
					r.draw(nil)
				}
				return false
			}
			if r.startMacroRecording("") == macroStartPrepareRegister {
				if r.Screen != nil {
					r.draw(nil)
				}
			}
			return false
		case '@':
			if r.macroRecording {
				return false
			}
			register := r.lastMacroRegister()
			if register != "" {
				if r.startMacroPlayback(register) {
					if r.Screen != nil {
						r.draw(nil)
					}
				}
				return false
			}
			if r.beginMacroPlayback("") {
				if r.Screen != nil {
					r.draw(nil)
				}
			}
			return false
		case 'i':
			count := r.consumeCount()
			r.Mode = ModeInsert
			r.beginInsertCapture(count, func(text string, _ int) {
				if text == "" {
					r.setLastChange(nil, 0)
					return
				}
				r.setLastChange(func(c int) {
					for i := 0; i < c; i++ {
						r.insertText(text)
					}
					if r.Screen != nil {
						r.draw(nil)
					}
				}, 1)
			})
			r.draw(nil)
			return false
		case 'a':
			count := r.consumeCount()
			if r.Buf != nil && r.Cursor < r.Buf.Len() {
				if r.Buf.RuneAt(r.Cursor) == '\n' {
					r.CursorLine++
				}
				r.Cursor++
			}
			r.Mode = ModeInsert
			r.beginInsertCapture(count, func(text string, _ int) {
				if text == "" {
					r.setLastChange(nil, 0)
					return
				}
				r.setLastChange(func(c int) {
					for i := 0; i < c; i++ {
						if r.Buf != nil && r.Cursor < r.Buf.Len() {
							if r.Buf.RuneAt(r.Cursor) == '\n' {
								r.CursorLine++
							}
							r.Cursor++
						}
						r.insertText(text)
					}
					if r.Screen != nil {
						r.draw(nil)
					}
				}, 1)
			})
			r.draw(nil)
			return false
		case 'u':
			count := r.consumeCount()
			for i := 0; i < count; i++ {
				r.performUndo("undo.normal")
			}
			return false
		case 'v':
			_ = r.consumeCount()
			r.Mode = ModeVisual
			r.VisualStart = r.Cursor
			r.VisualLine = false
			r.draw(nil)
			return false
		case 'V':
			_ = r.consumeCount()
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
			count := r.consumeCount()
			if r.KillRing.HasData() {
				text := r.KillRing.Get()
				if r.Buf != nil && r.Cursor < r.Buf.Len() {
					// paste after the cursor position
					if r.Buf.RuneAt(r.Cursor) == '\n' {
						r.CursorLine++
					}
					r.Cursor++
				}
				start := r.beginYankTracking()
				for i := 0; i < count; i++ {
					r.insertText(text)
				}
				r.endYankTracking(start, count)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "paste.normal", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			}
			return false
		case 'P':
			count := r.consumeCount()
			if r.KillRing.HasData() {
				text := r.KillRing.Get()
				start := r.beginYankTracking()
				for i := 0; i < count; i++ {
					r.insertText(text)
				}
				r.endYankTracking(start, count)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "paste.before", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			}
			return false
		case 'o':
			count := r.consumeCount()
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
			r.beginInsertCapture(count, func(text string, c int) {
				r.setLastChange(func(times int) {
					for i := 0; i < times*c; i++ {
						if r.Buf != nil {
							start, end := r.currentLineBounds()
							pos := end
							if end > start && r.Buf.RuneAt(end-1) == '\n' {
								pos = end - 1
							}
							r.Cursor = pos
						}
						r.insertText("\n")
						if text != "" {
							r.insertText(text)
						}
					}
					if r.Screen != nil {
						r.draw(nil)
					}
				}, c)
			})
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'c':
			if r.PendingC {
				r.PendingC = false
			} else {
				r.PendingC = true
			}
			return false
		case 'd':
			if r.PendingD {
				r.PendingD = false
				count := r.consumeCount()
				r.deleteLines(count)
			} else {
				r.PendingD = true
			}
			return false
		case '.':
			count := 0
			if r.PendingCount > 0 {
				count = r.consumeCount()
			}
			r.repeatLastChange(count)
			return false
		case 'y':
			if r.PendingY {
				r.PendingY = false
				if r.Buf != nil {
					count := r.consumeCount()
					start, end := r.currentLineBounds()
					for i := 1; i < count && end < r.Buf.Len(); i++ {
						for end < r.Buf.Len() && r.Buf.RuneAt(end) != '\n' {
							end++
						}
						if end < r.Buf.Len() {
							end++
						}
					}
					text := string(r.Buf.Slice(start, end))
					r.clearYankState()
					r.KillRing.Push(text)
					if r.Logger != nil {
						r.Logger.Event("action", map[string]any{"name": "yank.line", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
					}
				}
			} else {
				r.PendingY = true
			}
			return false
		case 'Y':
			count := r.consumeCount()
			if r.Buf != nil {
				start, end := r.currentLineBounds()
				for i := 1; i < count && end < r.Buf.Len(); i++ {
					for end < r.Buf.Len() && r.Buf.RuneAt(end) != '\n' {
						end++
					}
					if end < r.Buf.Len() {
						end++
					}
				}
				text := string(r.Buf.Slice(start, end))
				r.clearYankState()
				r.KillRing.Push(text)
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "yank.line", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
				}
			}
			r.PendingY = false
			return false
		case 'x':
			// Cut the character(s) at the cursor in normal mode
			count := r.consumeCount()
			r.deleteChars(count)
			return false
		case 'G':
			hasCount := r.PendingCount > 0
			count := r.consumeCount()
			if r.Buf != nil && r.Buf.Len() > 0 {
				if hasCount {
					lines := r.Buf.Lines()
					line := count - 1
					if line < 0 {
						line = 0
					}
					if len(lines) == 0 {
						line = 0
					} else if line > len(lines)-1 {
						line = len(lines) - 1
					}
					start, _ := r.Buf.LineAt(line)
					r.Cursor = start
					r.CursorLine = line
				} else {
					r.Cursor = r.Buf.Len() - 1
					lines := r.Buf.Lines()
					last := len(lines) - 1
					if last > 0 && len(lines[last]) == 0 {
						last--
					}
					r.CursorLine = last
				}
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'w':
			count := r.consumeCount()
			if r.PendingD {
				r.PendingD = false
				r.deleteWords(count)
				return false
			}
			if r.PendingC {
				r.PendingC = false
				r.deleteWords(count)
				r.Mode = ModeInsert
				r.beginInsertCapture(count, func(text string, c int) {
					r.setLastChange(func(times int) {
						r.changeWords(times, text)
					}, c)
				})
				if r.Screen != nil {
					r.draw(nil)
				}
				return false
			}
			if r.Buf != nil {
				for i := 0; i < count; i++ {
					r.Cursor = buffer.NextWordStart(r.Buf, r.Cursor)
				}
				r.recomputeCursorLine()
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'b':
			count := r.consumeCount()
			if r.Buf != nil {
				for i := 0; i < count; i++ {
					r.Cursor = buffer.WordStart(r.Buf, r.Cursor)
				}
				r.recomputeCursorLine()
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'e':
			count := r.consumeCount()
			if r.Buf != nil {
				for i := 0; i < count; i++ {
					r.Cursor = buffer.WordEnd(r.Buf, r.Cursor)
				}
				r.recomputeCursorLine()
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'h':
			count := r.consumeCount()
			for i := 0; i < count && r.Cursor > 0; i++ {
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
		case 'l':
			count := r.consumeCount()
			for i := 0; i < count; i++ {
				if r.Buf != nil && r.Cursor < r.Buf.Len() {
					if r.Buf.RuneAt(r.Cursor) == '\n' {
						r.CursorLine++
					}
					r.Cursor++
				}
			}
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'k':
			count := r.consumeCount()
			r.moveCursorVertical(-count)
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'j':
			count := r.consumeCount()
			r.moveCursorVertical(count)
			if r.Screen != nil {
				r.draw(nil)
			}
			return false
		case 'g':
			if r.PendingG {
				r.PendingG = false
				hasCount := r.PendingCount > 0
				count := r.consumeCount()
				if r.Buf != nil {
					if hasCount {
						line := count - 1
						if line < 0 {
							line = 0
						}
						lines := r.Buf.Lines()
						if len(lines) == 0 {
							line = 0
						} else if line > len(lines)-1 {
							line = len(lines) - 1
						}
						start, _ := r.Buf.LineAt(line)
						r.Cursor = start
						r.CursorLine = line
					} else {
						r.Cursor = 0
						r.CursorLine = 0
					}
				}
				if r.Screen != nil {
					r.draw(nil)
				}
			} else {
				r.PendingG = true
			}
			return false
		case '$':
			_ = r.consumeCount()
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
			_ = r.consumeCount()
			return r.runMnemonicMenu()
		}
	}
	if r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'm' && ev.Modifiers() == tcell.ModAlt {
		return r.runMnemonicMenu()
	}
	if r.Mode == ModeVisual && ev.Key() == tcell.KeyRune && ev.Rune() == 'v' && ev.Modifiers() == 0 {
		r.Mode = ModeNormal
		r.VisualStart = -1
		r.VisualLine = false
		r.PendingG = false
		r.PendingTextObject = false
		r.TextObjectAround = false
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
	if r.matchCommand(ev, "multi-edit") {
		r.toggleMultiEdit()
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "multi-edit.toggle"})
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
		if r.isInsertMode() {
			if r.KillRing.HasData() {
				text := r.KillRing.Get()
				start := r.beginYankTracking()
				r.insertText(text)
				r.endYankTracking(start, 1)
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
		if !r.isInsertMode() {
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
	if ev.Key() == tcell.KeyF1 || (ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == tcell.ModCtrl) || (ev.Key() == tcell.KeyCtrlH && !r.isInsertMode()) {
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
	if !r.isInsertMode() {
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
	if r.isInsertMode() && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'a' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlA) {
		start, _ := r.currentLineBounds()
		r.Cursor = start
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if r.isInsertMode() && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'e' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlE) {
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
	if ev.Key() == tcell.KeyLeft || ev.Key() == tcell.KeyCtrlB || (ev.Key() == tcell.KeyRune && ev.Rune() == 'b' && ev.Modifiers() == tcell.ModCtrl) || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == 0) {
		if r.Mode == ModeMultiEdit && r.MultiEdit != nil && r.Cursor <= r.MultiEdit.primaryStart {
			return false
		}
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
	if ev.Key() == tcell.KeyRight || ev.Key() == tcell.KeyCtrlF || (ev.Key() == tcell.KeyRune && ev.Rune() == 'f' && ev.Modifiers() == tcell.ModCtrl) || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'l' && ev.Modifiers() == 0) {
		if r.Mode == ModeMultiEdit && r.MultiEdit != nil && r.Cursor >= r.MultiEdit.primaryEnd {
			return false
		}
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
	if ev.Key() == tcell.KeyUp || ev.Key() == tcell.KeyCtrlP || (ev.Key() == tcell.KeyRune && ev.Rune() == 'p' && ev.Modifiers() == tcell.ModCtrl) || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == 0) {
		r.moveCursorVertical(-1)
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	if ev.Key() == tcell.KeyDown || ev.Key() == tcell.KeyCtrlN || (ev.Key() == tcell.KeyRune && ev.Rune() == 'n' && ev.Modifiers() == tcell.ModCtrl) || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'j' && ev.Modifiers() == 0) {
		r.moveCursorVertical(1)
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}

	// Insert typed rune only in insert mode
	if r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
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
	if r.isInsertMode() && (ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2) {
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
	if r.isInsertMode() && ev.Key() == tcell.KeyDelete {
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
	if r.isInsertMode() && ev.Key() == tcell.KeyEnter {
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
	if r.isInsertMode() && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlK) {
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
				r.KillRing.Push(text)
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
	if r.isInsertMode() && ((ev.Key() == tcell.KeyRune && ev.Rune() == 'u' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlU) {
		if r.KillRing.HasData() {
			text := r.KillRing.Get()
			start := r.beginYankTracking()
			r.insertText(text)
			r.endYankTracking(start, 1)
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
	if r.PendingTextObject {
		if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 && isTextObjectDelimiter(ev.Rune()) {
			_ = r.consumeCount()
			start, end, ok := r.textObjectBounds(ev.Rune(), r.TextObjectAround)
			r.PendingTextObject = false
			r.TextObjectAround = false
			if ok {
				r.VisualStart = start
				r.VisualLine = false
				r.Cursor = end - 1
				r.draw(nil)
			}
			return false
		}
		if ev.Key() != tcell.KeyRune || ev.Modifiers() != 0 || !isTextObjectDelimiter(ev.Rune()) {
			r.PendingTextObject = false
			r.TextObjectAround = false
		}
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
	case ev.Key() == tcell.KeyRune && (ev.Rune() == 'i' || ev.Rune() == 'a') && ev.Modifiers() == 0:
		r.PendingTextObject = true
		r.TextObjectAround = ev.Rune() == 'a'
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
	case ev.Key() == tcell.KeyRune && ev.Rune() == ' ' && ev.Modifiers() == 0:
		return r.runMnemonicMenu()
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'y' && ev.Modifiers() == 0:
		if r.Buf != nil {
			start, end := r.visualSelectionBounds()
			if start < end {
				text := string(r.Buf.Slice(start, end))
				r.clearYankState()
				r.KillRing.Push(text)
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
				r.KillRing.Push(text)
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
