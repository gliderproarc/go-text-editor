package app

import (
	"os"
	"strings"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/search"
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
}

// New creates an empty Runner.
func New() *Runner { return &Runner{Buf: buffer.NewGapBuffer(0)} }

// LoadFile loads a file into the runner's buffer.
func (r *Runner) LoadFile(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	r.FilePath = path
	r.Buf = buffer.NewGapBufferFromString(strings.ReplaceAll(string(data), "\r\n", "\n"))
	r.Cursor = r.Buf.Len()
	r.Dirty = false
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
			// If handleKeyEvent returns true, we should quit
			if r.handleKeyEvent(ev) {
				return nil
			}
			// If we were showing help, any key dismisses it and we redraw
			if r.ShowHelp {
				r.ShowHelp = false
				if r.Buf != nil && r.Buf.Len() > 0 {
					drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
				} else {
					drawUI(r.Screen)
				}
				continue
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
	// Ctrl+S -> save
	if ev.Key() == tcell.KeyRune && ev.Rune() == 's' && ev.Modifiers() == tcell.ModCtrl {
		_ = r.Save()
		if r.Screen != nil {
			if r.Buf != nil && r.Buf.Len() > 0 {
				drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
			} else {
				drawUI(r.Screen)
			}
		}
		return false
	}
	// Ctrl+W -> incremental search prompt
	if ev.Key() == tcell.KeyRune && ev.Rune() == 'w' && ev.Modifiers() == tcell.ModCtrl {
		r.runSearchPrompt()
		return false
	}
	// Alt+G -> go-to line (Alt modifier)
	if ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == tcell.ModAlt {
		r.runGoToPrompt()
		return false
	}
	// Show help on 'h'
	if ev.Key() == tcell.KeyRune && ev.Rune() == 'h' {
		r.ShowHelp = true
		if r.Screen != nil {
			drawHelp(r.Screen)
		}
		return false
	}

	// Insert typed rune (simple handling: any rune with no Ctrl/Alt)
	if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		r.Buf.Insert(r.Cursor, []rune{ev.Rune()})
		r.Cursor++
		r.Dirty = true
		if r.Screen != nil {
			drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
		}
		return false
	}

	// Backspace
	if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
		if r.Cursor > 0 {
			r.Buf.Delete(r.Cursor-1, r.Cursor)
			if r.Cursor > 0 {
				r.Cursor--
			}
			r.Dirty = true
			if r.Screen != nil {
				drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
			}
		}
		return false
	}

	// Delete (forward)
	if ev.Key() == tcell.KeyDelete {
		if r.Cursor < r.Buf.Len() {
			r.Buf.Delete(r.Cursor, r.Cursor+1)
			r.Dirty = true
			if r.Screen != nil {
				drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
			}
		}
		return false
	}

	// Enter -> newline
	if ev.Key() == tcell.KeyEnter {
		r.Buf.Insert(r.Cursor, []rune{'\n'})
		r.Cursor++
		r.Dirty = true
		if r.Screen != nil {
			drawBuffer(r.Screen, r.Buf, r.FilePath, nil)
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
		"- Press Ctrl+Q or 'q' to exit",
		"- Press 'h' for help",
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
