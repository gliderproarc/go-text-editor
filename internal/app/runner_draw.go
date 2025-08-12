package app

import (
	"strings"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

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
		"- Ctrl+K: Cut to end of line",
		"- Ctrl+U/Ctrl+Y: Paste",
		"- Ctrl+Z / Ctrl+Y: Undo / Redo",
		"- Ctrl+A/Ctrl+E: Line start/end (insert)",
		"- Modes: Normal (default), Insert (i), Visual (v)",
		"- Normal mode: p paste, a append",
		"- Visual mode: y copy, x cut, o open line",
		"- Arrow keys or Ctrl+B/F/P/N: Move cursor",
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

// showDialog displays a message in the mini-buffer and waits for a key press.
// After dismissal it redraws the current buffer or default UI.
func (r *Runner) showDialog(message string) {
	if r.Screen == nil {
		return
	}
	r.setMiniBuffer([]string{message, "Press any key to continue"})
	r.draw(nil)
	s := r.Screen
	for {
		if ev := s.PollEvent(); ev != nil {
			if _, ok := ev.(*tcell.EventKey); ok {
				break
			}
		}
	}
	r.clearMiniBuffer()
	if r.Buf != nil && r.Buf.Len() > 0 {
		r.draw(nil)
	} else {
		drawUI(s)
	}
}

func drawBuffer(s tcell.Screen, buf *buffer.GapBuffer, fname string, highlights []search.Range, cursor int, dirty bool, mode Mode, topLine int, minibuf []string) {
	if buf == nil {
		drawFile(s, fname, []string{}, highlights, cursor, dirty, mode, topLine, minibuf)
		return
	}
	content := buf.String()
	lines := strings.Split(content, "\n")
	drawFile(s, fname, lines, highlights, cursor, dirty, mode, topLine, minibuf)
}

// draw renders the buffer with optional highlights and current visual selection.
func (r *Runner) draw(highlights []search.Range) {
	if r.Screen == nil {
		return
	}
	r.ensureCursorVisible()
	if vh := r.visualHighlightRange(); len(vh) > 0 {
		highlights = append(highlights, vh...)
	}
	drawBuffer(r.Screen, r.Buf, r.FilePath, highlights, r.Cursor, r.Dirty, r.Mode, r.TopLine, r.MiniBuf)
}

func drawFile(s tcell.Screen, fname string, lines []string, highlights []search.Range, cursor int, dirty bool, mode Mode, topLine int, minibuf []string) {
	width, height := s.Size()
	s.Clear()
	mbHeight := len(minibuf)
	maxLines := height - 1 - mbHeight
	if maxLines < 0 {
		maxLines = 0
	}
	lineStart := 0     // byte offset of start of current line
	lineStartRune := 0 // rune offset of start of current line
	for i := 0; i < topLine && i < len(lines); i++ {
		lineStart += len([]byte(lines[i])) + 1
		lineStartRune += len([]rune(lines[i])) + 1
	}
	cursorColor := tcell.ColorWhite
	switch mode {
	case ModeInsert:
		cursorColor = tcell.ColorBlue
	case ModeNormal:
		cursorColor = tcell.ColorGreen
	}
	cursorStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(cursorColor).Attributes(tcell.AttrBlink)
	for i := 0; i < maxLines && topLine+i < len(lines); i++ {
		line := lines[topLine+i]
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
			runeIdx := lineStartRune + j
			switch {
			case runeIdx == cursor:
				s.SetContent(j, i, ch, nil, cursorStyle)
			case j < len(hl) && hl[j]:
				s.SetContent(j, i, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow))
			default:
				s.SetContent(j, i, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
			}
		}
		// if cursor at end of line, draw placeholder cell
		if lineStartRune+len(runes) == cursor && len(runes) < width {
			s.SetContent(len(runes), i, ' ', nil, cursorStyle)
		}
		// advance offsets by bytes/runes in line + 1 for the newline
		lineStart += len([]byte(line)) + 1
		lineStartRune += len(runes) + 1
	}
	display := fname
	if display == "" {
		display = "[No File]"
	}
	if dirty {
		display += " [+]"
	}
	status := display + " — Press Ctrl+Q to exit"
	if len(status) > width {
		status = string([]rune(status)[:width])
	}
	for i, r := range status {
		s.SetContent(i, height-1, r, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
	}
	// draw mini-buffer lines just above status bar
	for i, line := range minibuf {
		y := height - 1 - mbHeight + i
		runes := []rune(line)
		for x := 0; x < width; x++ {
			ch := ' '
			if x < len(runes) {
				ch = runes[x]
			}
			s.SetContent(x, y, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
		}
	}
	s.Show()
}
