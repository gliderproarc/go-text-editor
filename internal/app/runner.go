package app

import (
    "os"
    "strings"

    "example.com/texteditor/pkg/buffer"
    "example.com/texteditor/pkg/history"
    "example.com/texteditor/pkg/search"
    "example.com/texteditor/pkg/logs"
    "github.com/gdamore/tcell/v2"
)

// Runner owns the terminal lifecycle and a minimal event loop.
type Runner struct {
    Screen   tcell.Screen
    FilePath string
    Buf      *buffer.GapBuffer
    Cursor   int // cursor position in runes
    Dirty    bool
    ShowHelp bool
    History  *history.History
    KillRing history.KillRing
    Logger   *logs.Logger
}

// New creates an empty Runner.
func New() *Runner { return &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()} }

// LoadFile loads a file into the runner's buffer.
func (r *Runner) LoadFile(path string) error {
    if path == "" {
        return nil
    }
    if r.Logger != nil {
        r.Logger.Event("open.attempt", map[string]any{"file": path})
    }
    data, err := os.ReadFile(path)
    if err != nil {
        if r.Logger != nil {
            r.Logger.Event("open.error", map[string]any{"file": path, "error": err.Error()})
        }
        return err
    }
    r.FilePath = path
    // Normalize CRLF to LF for internal buffer storage
    normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
    r.Buf = buffer.NewGapBufferFromString(normalized)
    r.Cursor = r.Buf.Len()
    r.Dirty = false
    if r.Logger != nil {
        r.Logger.Event("open.success", map[string]any{"file": path, "bytes": len(data), "runes": r.Buf.Len()})
    }
    return nil
}

// Save writes the buffer contents to the current FilePath and clears Dirty.
func (r *Runner) Save() error {
	if r.FilePath == "" {
		return os.ErrInvalid
	}
	data := []byte(r.Buf.String())
	if err := os.WriteFile(r.FilePath, data, 0644); err != nil {
		return err
	}
	r.Dirty = false
	return nil
}

// InitScreen initializes a tcell screen if one is not already set.
func (r *Runner) InitScreen() error {
	if r.Screen != nil {
		return nil
	}
	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	s.SetStyle(tcell.StyleDefault)
	s.Clear()
	r.Screen = s
	return nil
}

// Fini finalizes the screen if initialized.
func (r *Runner) Fini() {
    if r.Screen != nil {
        r.Screen.Fini()
        r.Screen = nil
    }
    if r.Logger != nil {
        r.Logger.Close()
    }
}

// Run starts the event loop. It will initialize the screen if needed and
// return when the user requests quit (Ctrl+Q).
func (r *Runner) Run() error {
    if r.Screen == nil {
        if err := r.InitScreen(); err != nil {
            return err
        }
        defer r.Fini()
    }

    // Initialize logger from env (no-op if disabled)
    if r.Logger == nil {
        r.Logger = logs.NewFromEnv()
    }
    if r.Logger != nil {
        r.Logger.Event("run.start", map[string]any{"file": r.FilePath})
        defer r.Logger.Event("run.end", map[string]any{"file": r.FilePath})
    }

	// initial draw
	if r.Buf != nil && r.Buf.Len() > 0 {
		drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
	} else {
		drawUI(r.Screen)
	}

	for {
		ev := r.Screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if r.Logger != nil {
				r.Logger.Event("key", map[string]any{
					"type":      "EventKey",
					"key":       int(ev.Key()),
					"rune":      string(ev.Rune()),
					"modifiers": int(ev.Modifiers()),
				})
			}
			// If help is currently shown, consume this key to dismiss it
			if r.ShowHelp {
				r.ShowHelp = false
				if r.Buf != nil && r.Buf.Len() > 0 {
					drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
				} else {
					drawUI(r.Screen)
				}
				continue
			}
			// Otherwise, handle the key normally; if it requests quit, exit
			if r.handleKeyEvent(ev) {
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "quit"})
				}
				return nil
			}
		case *tcell.EventResize:
			r.Screen.Sync()
			if r.ShowHelp {
				drawHelp(r.Screen)
			} else {
				if r.Buf != nil && r.Buf.Len() > 0 {
					drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
				} else {
					drawUI(r.Screen)
				}
			}
		}
	}
}

// handleKeyEvent processes a key event. It returns true if the event signals
// the runner should quit.
func (r *Runner) handleKeyEvent(ev *tcell.EventKey) bool {
	// Ctrl+Q via rune + Ctrl
	if ev.Key() == tcell.KeyRune && ev.Rune() == 'q' && ev.Modifiers() == tcell.ModCtrl {
		return true
	}
	// Some platforms expose a dedicated CtrlQ key
	if ev.Key() == tcell.KeyCtrlQ {
		return true
	}
    // Ctrl+S -> save (handle both rune+Ctrl and dedicated control key)
    if (ev.Key() == tcell.KeyRune && ev.Rune() == 's' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlS {
        if r.FilePath == "" {
            // No file path set; prompt for Save As
            r.runSaveAsPrompt()
        } else {
            _ = r.Save()
        }
        if r.Logger != nil {
            r.Logger.Event("action", map[string]any{"name": "save", "file": r.FilePath})
        }
        if r.Screen != nil {
            if r.Buf != nil && r.Buf.Len() > 0 {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            } else {
                drawUI(r.Screen)
            }
        }
        return false
    }
    // Ctrl+O -> open file prompt (handle both rune+Ctrl and dedicated control key)
    if (ev.Key() == tcell.KeyRune && ev.Rune() == 'o' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlO {
        r.runOpenPrompt()
        if r.Logger != nil {
            r.Logger.Event("action", map[string]any{"name": "open.prompt"})
        }
        if r.Screen != nil {
            if r.Buf != nil && r.Buf.Len() > 0 {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            } else {
                drawUI(r.Screen)
            }
        }
        return false
    }
    // Ctrl+Z -> undo
    if ev.Key() == tcell.KeyRune && ev.Rune() == 'z' && ev.Modifiers() == tcell.ModCtrl {
        if r.History != nil {
            _ = r.History.Undo(r.Buf, &r.Cursor)
            r.Dirty = true
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "undo", "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
            }
            if r.Screen != nil {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            }
        }
        return false
    }
    // Ctrl+Y -> redo
    if ev.Key() == tcell.KeyRune && ev.Rune() == 'y' && ev.Modifiers() == tcell.ModCtrl {
        if r.History != nil {
            _ = r.History.Redo(r.Buf, &r.Cursor)
            r.Dirty = true
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "redo", "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
            }
            if r.Screen != nil {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            }
        }
        return false
    }
	// Ctrl+W -> incremental search prompt
    if ev.Key() == tcell.KeyRune && ev.Rune() == 'w' && ev.Modifiers() == tcell.ModCtrl {
        r.runSearchPrompt()
        if r.Logger != nil {
            r.Logger.Event("action", map[string]any{"name": "search.prompt"})
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
    // Show help on F1 or Ctrl+H (note: many terminals map Ctrl+H to Backspace)
    if ev.Key() == tcell.KeyF1 || (ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == tcell.ModCtrl) || ev.Key() == tcell.KeyCtrlH {
        r.ShowHelp = true
        if r.Logger != nil {
            r.Logger.Event("action", map[string]any{"name": "help.show"})
        }
        if r.Screen != nil {
            drawHelp(r.Screen)
        }
        return false
    }

    // Insert typed rune (simple handling: any rune with no Ctrl/Alt)
    if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
        text := string(ev.Rune())
        r.insertText(text)
        if r.Logger != nil {
            r.Logger.Event("action", map[string]any{"name": "insert", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
        }
        if r.Screen != nil {
            drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
        }
        return false
    }

	// Backspace
    if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
        if r.Cursor > 0 {
            // capture deleted rune
            del := string(r.Buf.Slice(r.Cursor-1, r.Cursor))
            _ = r.deleteRange(r.Cursor-1, r.Cursor, del)
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "backspace", "deleted": del, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
            }
            if r.Screen != nil {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            }
        }
        return false
    }

	// Delete (forward)
    if ev.Key() == tcell.KeyDelete {
        if r.Cursor < r.Buf.Len() {
            del := string(r.Buf.Slice(r.Cursor, r.Cursor+1))
            _ = r.deleteRange(r.Cursor, r.Cursor+1, del)
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "delete", "deleted": del, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
            }
            if r.Screen != nil {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            }
        }
        return false
    }

	// Enter -> newline
    if ev.Key() == tcell.KeyEnter {
        r.insertText("\n")
        if r.Logger != nil {
            r.Logger.Event("action", map[string]any{"name": "newline", "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
        }
        if r.Screen != nil {
            drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
        }
        return false
    }

    // Ctrl+K -> kill (cut) current line to kill ring
    if ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == tcell.ModCtrl {
        start, end := r.currentLineBounds()
        if end > start {
            text := string(r.Buf.Slice(start, end))
            _ = r.deleteRange(start, end, text)
            r.KillRing.Set(text)
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "kill.line", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
            }
            // Move cursor to start (now at next line)
            r.Cursor = start
            if r.Screen != nil {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            }
        }
        return false
    }
    // Ctrl+U -> yank (paste) from kill ring
    if ev.Key() == tcell.KeyRune && ev.Rune() == 'u' && ev.Modifiers() == tcell.ModCtrl {
        if r.KillRing.HasData() {
            text := r.KillRing.Get()
            r.insertText(text)
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "yank", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
            }
            if r.Screen != nil {
                drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
            }
        }
        return false
    }

	return false
}

// Minimal UI helpers (kept here so runner does not depend on package main)
func drawUI(s tcell.Screen) {
	width, height := s.Size()
	msg := "TextEditor: No File"
	msgX := (width - len(msg)) / 2
	msgY := height / 2
	for i, r := range msg {
		s.SetContent(msgX+i, msgY, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}
	status := "Press Ctrl+Q to exit"
	sbX := (width - len(status)) / 2
	sbY := height - 1
	for i, r := range status {
		s.SetContent(sbX+i, sbY, r, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
	}
	s.Show()
}

func drawHelp(s tcell.Screen) {
    width, height := s.Size()
    s.Clear()
    s.SetStyle(tcell.StyleDefault)
    lines := []string{
        "Help:",
        "- F1: Show this help (recommended)",
        "- Ctrl+H: Show help (if terminal supports)",
        "- Ctrl+Q: Quit",
        "- Ctrl+O: Open file",
        "- Ctrl+S: Save (Save As if no file)",
        "- Ctrl+W: Search",
        "- Alt+G: Go to line",
        "- Ctrl+K: Cut line",
        "- Ctrl+U: Paste",
        "- Ctrl+Z / Ctrl+Y: Undo / Redo",
        "- Enter: New line; Backspace/Delete: Remove",
        "- Typing: Inserts characters",
    }
    y := (height - len(lines)) / 2
    for i, line := range lines {
        x := (width - len(line)) / 2
        for j, r := range line {
            s.SetContent(x+j, y+i, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
        }
    }
    s.Show()
}

func drawBuffer(s tcell.Screen, buf *buffer.GapBuffer, fname string, highlights []search.Range) {
	if buf == nil {
		drawFile(s, fname, []string{}, highlights)
		return
	}
	content := buf.String()
	lines := strings.Split(content, "\n")
	drawFile(s, fname, lines, highlights)
}

// insertText inserts text at the current cursor, records history, and updates state.
func (r *Runner) insertText(text string) {
    if text == "" {
        return
    }
    _ = r.Buf.Insert(r.Cursor, []rune(text))
    if r.History != nil {
        r.History.RecordInsert(r.Cursor, text)
    }
    r.Cursor += len([]rune(text))
    r.Dirty = true
}

// deleteRange deletes [start,end) with provided text for history and updates cursor.
func (r *Runner) deleteRange(start, end int, text string) error {
    if start < 0 {
        start = 0
    }
    if end > r.Buf.Len() {
        end = r.Buf.Len()
    }
    if start >= end {
        return nil
    }
    if err := r.Buf.Delete(start, end); err != nil {
        return err
    }
    if r.History != nil {
        r.History.RecordDelete(start, text)
    }
    // adjust cursor
    if r.Cursor > end {
        r.Cursor -= (end - start)
    } else if r.Cursor > start {
        r.Cursor = start
    }
    r.Dirty = true
    return nil
}

// currentLineBounds returns the rune start and end indices for the current cursor's line.
func (r *Runner) currentLineBounds() (start, end int) {
    // compute line index by counting newlines up to cursor
    line := 0
    for i := 0; i < r.Cursor && i < r.Buf.Len(); i++ {
        if string(r.Buf.Slice(i, i+1)) == "\n" {
            line++
        }
    }
    return r.Buf.LineAt(line)
}

func drawFile(s tcell.Screen, fname string, lines []string, highlights []search.Range) {
	width, height := s.Size()
	s.Clear()
	maxLines := height - 1
	if maxLines < 0 {
		maxLines = 0
	}
	lineStart := 0 // byte offset of start of current line
	for i := 0; i < maxLines && i < len(lines); i++ {
		line := lines[i]
		runes := []rune(line)
		// compute highlights for this line as rune index intervals
		hl := make([]bool, len(runes))
		if len(highlights) > 0 {
			lineBytesLen := len(line)
			lineStartByte := lineStart
			lineEndByte := lineStartByte + lineBytesLen
			for _, h := range highlights {
				if h.Start < lineEndByte && h.End > lineStartByte {
					overlapStart := h.Start
					if overlapStart < lineStartByte {
						overlapStart = lineStartByte
					}
					overlapEnd := h.End
					if overlapEnd > lineEndByte {
						overlapEnd = lineEndByte
					}
					// convert byte offsets relative to line to rune indices
					startRune := 0
					endRune := 0
					if overlapStart-lineStartByte > 0 {
						startRune = len([]rune(line[:overlapStart-lineStartByte]))
					}
					if overlapEnd-lineStartByte > 0 {
						endRune = len([]rune(line[:overlapEnd-lineStartByte]))
					}
					if startRune < 0 {
						startRune = 0
					}
					if endRune > len(runes) {
						endRune = len(runes)
					}
					for ri := startRune; ri < endRune && ri < len(hl); ri++ {
						hl[ri] = true
					}
				}
			}
		}
		for j := 0; j < width && j < len(runes); j++ {
			ch := runes[j]
			if j < len(hl) && hl[j] {
				s.SetContent(j, i, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow))
			} else {
				s.SetContent(j, i, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
			}
		}
		// advance lineStart by bytes in line + 1 for the newline
		lineStart += len([]byte(line)) + 1
	}
	status := fname + " — Press Ctrl+Q to exit"
	if len(status) > width {
		status = string([]rune(status)[:width])
	}
	for i, r := range status {
		s.SetContent(i, height-1, r, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
	}
	s.Show()
}
