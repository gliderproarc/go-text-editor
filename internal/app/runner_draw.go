package app

import (
	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

// renderState captures a snapshot of editor state for the renderer goroutine.
type renderState struct {
	lines      []string
	filePath   string
	cursor     int
	dirty      bool
	mode       Mode
	topLine    int
	miniBuf    []string
	highlights []search.Range
	showHelp   bool
	bufLen     int
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
		"- Ctrl+K: Cut to end of line",
		"- Ctrl+U/Ctrl+Y: Paste",
		"- Ctrl+Z / Ctrl+Y: Undo / Redo",
		"- Ctrl+A/Ctrl+E: Line start/end (insert)",
		"- Modes: Normal (default), Insert (i), Visual (v)",
		"- Normal mode: p paste, a append, dd delete line",
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
	// Wait for a single event before dismissing the dialog.
	if r.EventCh != nil {
		<-r.EventCh
	} else {
		s := r.Screen
		for {
			if ev := s.PollEvent(); ev != nil {
				if _, ok := ev.(*tcell.EventKey); ok {
					break
				}
			}
		}
	}
	r.clearMiniBuffer()
	r.draw(nil)
}

func drawBuffer(s tcell.Screen, buf *buffer.GapBuffer, fname string, highlights []search.Range, cursor int, dirty bool, mode Mode, topLine int, minibuf []string) {
	if buf == nil {
		drawFile(s, fname, []string{}, highlights, cursor, dirty, mode, topLine, minibuf)
		return
	}
	drawFile(s, fname, buf.Lines(), highlights, cursor, dirty, mode, topLine, minibuf)
}

// renderSnapshot captures the current runner state into a renderState.
func (r *Runner) renderSnapshot(highlights []search.Range) renderState {
	r.ensureCursorVisible()
	if vh := r.visualHighlightRange(); len(vh) > 0 {
		highlights = append(highlights, vh...)
	}
	if sh := r.syntaxHighlights(); len(sh) > 0 {
		highlights = append(highlights, sh...)
	}
	mini := append([]string(nil), r.MiniBuf...)
	hs := append([]search.Range(nil), highlights...)
	var lines []string
	bufLen := 0
	if r.Buf != nil {
		lines = r.Buf.Lines()
		bufLen = r.Buf.Len()
	}
	return renderState{
		lines:      lines,
		filePath:   r.FilePath,
		cursor:     r.Cursor,
		dirty:      r.Dirty,
		mode:       r.Mode,
		topLine:    r.TopLine,
		miniBuf:    mini,
		highlights: hs,
		showHelp:   r.ShowHelp,
		bufLen:     bufLen,
	}
}

// renderToScreen draws the provided snapshot to the tcell screen.
func renderToScreen(s tcell.Screen, st renderState) {
	if st.showHelp {
		drawHelp(s)
		return
	}
	if st.bufLen > 0 {
		drawFile(s, st.filePath, st.lines, st.highlights, st.cursor, st.dirty, st.mode, st.topLine, st.miniBuf)
	} else {
		drawUI(s)
	}
}

// draw renders the buffer with optional highlights and current visual selection.
// If a render channel is configured, the snapshot is sent to the renderer
// goroutine; otherwise it is drawn synchronously.
func (r *Runner) draw(highlights []search.Range) {
	if r.Screen == nil {
		return
	}
	snapshot := r.renderSnapshot(highlights)
	if r.RenderCh != nil {
		r.RenderCh <- snapshot
		return
	}
	renderToScreen(r.Screen, snapshot)
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
    // simple theme mapping for syntax groups
    theme := map[string]tcell.Color{
        "keyword":  tcell.ColorRed,
        "string":   tcell.ColorGreen,
        "comment":  tcell.ColorGray,
        "number":   tcell.ColorYellow,
        "type":     tcell.ColorBlue,
        "function": tcell.ColorBlue,
    }

    for i := 0; i < maxLines && topLine+i < len(lines); i++ {
        line := lines[topLine+i]
        runes := []rune(line)
        // compute highlights for this line:
        // - bgHL marks background highlights (search/selection)
        // - fgGroup stores syntax group per rune (colored foreground)
        bgHL := make([]bool, len(runes))
        fgGroup := make([]string, len(runes))
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
                    for ri := startRune; ri < endRune && ri < len(runes); ri++ {
                        if h.Group == "" {
                            bgHL[ri] = true
                        } else {
                            fgGroup[ri] = h.Group
                        }
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
            case j < len(bgHL) && bgHL[j]:
                s.SetContent(j, i, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow))
            default:
                // syntax foreground coloring if present
                if g := fgGroup[j]; g != "" {
                    col, ok := theme[g]
                    if !ok {
                        col = tcell.ColorWhite
                    }
                    style := tcell.StyleDefault.Foreground(col)
                    // make comments dimmer for subtlety
                    if g == "comment" {
                        style = style.Attributes(tcell.AttrDim)
                    }
                    if g == "function" {
                        style = style.Attributes(tcell.AttrBold)
                    }
                    s.SetContent(j, i, ch, nil, style)
                } else {
                    s.SetContent(j, i, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
                }
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
