package app

import (
	"strings"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

// renderState captures a snapshot of editor state for the renderer goroutine.
type renderState struct {
	lines       []string
	filePath    string
	cursor      int
	dirty       bool
	mode        Mode
	overlay     Overlay
	macroStatus string
	topLine     int
	miniBuf     []string
	highlights  []search.Range
	showHelp    bool
	bufLen      int
	theme       config.Theme
	view        View
}

// Minimal UI helpers (kept here so runner does not depend on package main)
func drawUI(s tcell.Screen, th config.Theme, macroStatus string) {
	width, height := s.Size()
	msg := "TextEditor: No File"
	msgX := (width - len(msg)) / 2
	msgY := height / 2
	for i, r := range msg {
		s.SetContent(msgX+i, msgY, r, nil, tcell.StyleDefault.Foreground(th.TextDefault))
	}
	status := "Press Ctrl+Q to exit"
	status = appendMacroStatus(status, macroStatus)
	sbX := (width - len(status)) / 2
	statusRow := height - 1
	for i, r := range status {
		s.SetContent(sbX+i, statusRow, r, nil, tcell.StyleDefault.Foreground(th.StatusForeground).Background(th.StatusBackground))
	}
	drawMacroRecordingIndicator(s, th, statusRow, width, macroStatus)
	s.Show()
}

func appendMacroStatus(status string, macroStatus string) string {
	if macroStatus == "" {
		return status
	}
	return status + " | " + macroStatus
}

func drawHelp(s tcell.Screen, th config.Theme) {
	_, height := s.Size()
	s.Clear()
	s.SetStyle(tcell.StyleDefault.Foreground(th.UIForeground).Background(th.UIBackground))
	lines := []string{
		"Help:",
		"- F1: Show this help (recommended)",
		"- Ctrl+H: Show help (if terminal supports)",
		"- Ctrl+Q: Quit",
		"- Ctrl+O: Open file",
		"- Ctrl+S: Save (Save As if no file)",
		"- Ctrl+W: Search",
		"- Ctrl+L: Multi-edit",
		"- Alt+G: Go to line",
		"- Ctrl+K: Cut to end of line",
		"- Ctrl+U/Ctrl+Y: Paste",
		"- Ctrl+Z / Ctrl+Y: Undo / Redo",
		"- Ctrl+A/Ctrl+E: Line start/end (insert)",
		"- Macros: q{reg} record, q stop, @{reg} play, @@ replay",
		"- Modes: Normal (default), Insert (i), Visual (v)",
		"- Normal mode: p paste, a append, dd delete line",
		"- Visual mode: y copy, x cut, o open line",
		"- Arrow keys or Ctrl+B/F/P/N: Move cursor",
		"- Enter: New line; Backspace/Delete: Remove",
		"- Typing: Inserts characters",
	}
	leftPadding := 4
	y := (height - len(lines)) / 2
	for i, line := range lines {
		x := leftPadding
		for j, r := range line {
			s.SetContent(x+j, y+i, r, nil, tcell.StyleDefault.Foreground(th.TextDefault))
		}
	}
	s.Show()
}

// showDialog displays a message in the mini-buffer and waits for a key press.
// After dismissal it redraws the current buffer or default UI.
func (r *Runner) showDialog(message string) {
	r.showDialogLines([]string{message})
}

// showDialogLines displays multiple lines in the mini-buffer and waits for a key press.
func (r *Runner) showDialogLines(lines []string) {
	if r.Screen == nil {
		return
	}
	lines = append(lines, "Press any key to continue")
	r.setMiniBuffer(lines)
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

func drawBuffer(s tcell.Screen, buf *buffer.GapBuffer, fname string, highlights []search.Range, cursor int, dirty bool, mode Mode, overlay Overlay, topLine int, minibuf []string, th config.Theme, macroStatus string) {
	if buf == nil {
		drawFile(s, fname, []string{}, highlights, cursor, dirty, mode, overlay, topLine, minibuf, th, macroStatus)
		return
	}
	drawFile(s, fname, buf.Lines(), highlights, cursor, dirty, mode, overlay, topLine, minibuf, th, macroStatus)
}

// renderSnapshot captures the current runner state into a renderState.
func (r *Runner) renderSnapshot(highlights []search.Range) renderState {
	r.ensureCursorVisible()
	// kick background updates based on current viewport/content (non-blocking)
	if r.View != ViewFileManager {
		r.updateSpellAsync()
		r.updateSyntaxAsync()
	}
	if vh := r.visualHighlightRange(); len(vh) > 0 {
		highlights = append(highlights, vh...)
	}
	if mh := r.multiEditHighlights(); len(mh) > 0 {
		highlights = append(highlights, mh...)
	}
	if sh := r.syntaxHighlightsCached(); len(sh) > 0 {
		highlights = append(highlights, sh...)
	}
	if sp := r.spellHighlights(); len(sp) > 0 {
		highlights = append(highlights, sp...)
	}
	mini := append([]string(nil), r.MiniBuf...)
	hs := append([]search.Range(nil), highlights...)
	macroStatus := r.MacroStatus
	var lines []string
	bufLen := 0
	if r.Buf != nil {
		lines = r.Buf.Lines()
		bufLen = r.Buf.Len()
	}
	if r.View == ViewFileManager && r.Buf == nil {
		bufLen = 1
	}
	return renderState{
		lines:       lines,
		filePath:    r.FilePath,
		cursor:      r.Cursor,
		dirty:       r.Dirty,
		mode:        r.Mode,
		overlay:     r.Overlay,
		macroStatus: macroStatus,
		topLine:     r.TopLine,
		miniBuf:     mini,
		highlights:  hs,
		showHelp:    r.ShowHelp,
		bufLen:      bufLen,
		theme:       r.Theme,
		view:        r.View,
	}
}

// renderToScreen draws the provided snapshot to the tcell screen.
func renderToScreen(s tcell.Screen, st renderState) {
	if st.showHelp {
		drawHelp(s, st.theme)
		return
	}
	if st.bufLen > 0 || st.view == ViewFileManager {
		drawFile(s, st.filePath, st.lines, st.highlights, st.cursor, st.dirty, st.mode, st.overlay, st.topLine, st.miniBuf, st.theme, st.macroStatus)
		return
	}
	drawUI(s, st.theme, st.macroStatus)
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
		// Coalesce renders: drop any pending frame so the latest wins.
		// This avoids showing stale intermediate frames (e.g., pre-theme/syntax)
		// which can appear as brief color changes.
		select {
		case <-r.RenderCh:
			// dropped an older pending frame
		default:
		}
		r.RenderCh <- snapshot
		return
	}
	renderToScreen(r.Screen, snapshot)
}

func drawFile(s tcell.Screen, fname string, lines []string, highlights []search.Range, cursor int, dirty bool, mode Mode, overlay Overlay, topLine int, minibuf []string, th config.Theme, macroStatus string) {
	width, height := s.Size()
	s.Clear()
	// set default UI style
	s.SetStyle(tcell.StyleDefault.Foreground(th.UIForeground).Background(th.UIBackground))
	statusLineCount := 1
	mbHeight := len(minibuf)
	maxLines := height - statusLineCount - mbHeight
	if maxLines < 0 {
		maxLines = 0
	}
	lineStart := 0     // byte offset of start of current line
	lineStartRune := 0 // rune offset of start of current line
	for i := 0; i < topLine && i < len(lines); i++ {
		lineStart += len([]byte(lines[i])) + 1
		lineStartRune += len([]rune(lines[i])) + 1
	}
	cursorColor := th.CursorNormalBG
	switch mode {
	case ModeInsert, ModeMultiEdit:
		cursorColor = th.CursorInsertBG
	case ModeVisual:
		cursorColor = th.CursorVisualBG
	case ModeNormal:
		cursorColor = th.CursorNormalBG
	}
	cursorStyle := tcell.StyleDefault.Foreground(th.CursorText).Background(cursorColor).Attributes(tcell.AttrBlink)
	// syntax colors from theme (fallbacks handled below)

	for i := 0; i < maxLines && topLine+i < len(lines); i++ {
		line := lines[topLine+i]
		runes := []rune(line)
		// compute highlights for this line:
		// - bgHL marks background highlights (search/selection)
		// - bgGroup stores background highlight category (e.g., "bg.search.current")
		// - fgGroup stores syntax group per rune (colored foreground)
		// - ulHL marks underline attribute (used for spell-check misspellings)
		bgHL := make([]bool, len(runes))
		bgGroup := make([]string, len(runes))
		fgGroup := make([]string, len(runes))
		ulHL := make([]bool, len(runes))
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
						switch h.Group {
						case "":
							// default background highlight (search/selection)
							bgHL[ri] = true
							bgGroup[ri] = "bg.search"
						case "bg.search", "bg.search.current", "bg.select", "bg.multiedit", "bg.multiedit.current":
							// explicit background highlight kinds
							bgHL[ri] = true
							bgGroup[ri] = h.Group

						case "bg.spell":
							// Spell-check: visually underline characters (no bg)
							ulHL[ri] = true
						default:
							// syntax foreground coloring
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
				// choose background color based on bgGroup
				bg := th.HighlightSearchBG
				fg := th.HighlightSearchFG
				if g := bgGroup[j]; g == "bg.search.current" || g == "bg.multiedit.current" {
					bg = th.HighlightSearchCurrentBG
					fg = th.HighlightSearchCurrentFG
				} else if g := bgGroup[j]; g == "bg.select" {
					// Subtle visual selection highlight
					bg = th.SelectBG
					fg = th.SelectFG
				}
				style := tcell.StyleDefault.Foreground(fg).Background(bg)
				if j < len(ulHL) && ulHL[j] {
					// Underline and use the configured underline color for fg
					style = style.Foreground(th.HighlightSpellUnderlineFG).Attributes(tcell.AttrUnderline)
				}
				s.SetContent(j, i, ch, nil, style)
			default:
				// syntax foreground coloring if present
				if g := fgGroup[j]; g != "" {
					col, ok := th.SyntaxColors[g]
					if !ok {
						col = th.TextDefault
					}
					style := tcell.StyleDefault.Foreground(col)
					// make comments dimmer for subtlety
					if g == "comment" {
						style = style.Attributes(tcell.AttrDim)
					}
					if g == "function" {
						style = style.Attributes(tcell.AttrBold)
					}
					if j < len(ulHL) && ulHL[j] {
						// Switch to underline color and include underline attribute.
						// Preserve simple bold/dim cases explicitly.
						if g == "comment" {
							style = tcell.StyleDefault.Foreground(th.HighlightSpellUnderlineFG).Attributes(tcell.AttrDim | tcell.AttrUnderline)
						} else if g == "function" {
							style = tcell.StyleDefault.Foreground(th.HighlightSpellUnderlineFG).Attributes(tcell.AttrBold | tcell.AttrUnderline)
						} else {
							style = tcell.StyleDefault.Foreground(th.HighlightSpellUnderlineFG).Attributes(tcell.AttrUnderline)
						}
					}
					s.SetContent(j, i, ch, nil, style)
				} else {
					style := tcell.StyleDefault.Foreground(th.TextDefault)
					if j < len(ulHL) && ulHL[j] {
						style = tcell.StyleDefault.Foreground(th.HighlightSpellUnderlineFG).Attributes(tcell.AttrUnderline)
					}
					s.SetContent(j, i, ch, nil, style)
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
	isFileManager := false
	if len(display) >= len("[File Manager]") && display[:len("[File Manager]")] == "[File Manager]" {
		isFileManager = true
	}
	if isFileManager {
		modeTag := "<FM>"
		status := modeTag + "  " + display + " — Enter to open, Esc to close"
		status = appendMacroStatus(status, macroStatus)
		status = truncateStatusForIndicator(status, width, macroStatus)
		modeColor := tcell.ColorOrange
		statusRow := height - 1
		for i, r := range status {
			style := tcell.StyleDefault.Foreground(th.StatusForeground).Background(th.StatusBackground)
			if i < len(modeTag) {
				style = tcell.StyleDefault.Foreground(modeColor).Background(th.StatusBackground).Attributes(tcell.AttrBold)
			}
			s.SetContent(i, statusRow, r, nil, style)
		}
		drawMacroRecordingIndicator(s, th, statusRow, width, macroStatus)
		// draw mini-buffer lines just above status bar
		for i, line := range minibuf {
			y := height - statusLineCount - mbHeight + i
			runes := []rune(line)
			defStyle := tcell.StyleDefault.Foreground(th.MiniForeground).Background(th.MiniBackground)
			isMnemonic := len(runes) >= 4 && runes[0] == ' ' && runes[2] == ' ' && runes[3] == '-'
			for x := 0; x < width; x++ {
				ch := ' '
				style := defStyle
				if x < len(runes) {
					ch = runes[x]
					if isMnemonic && x == 1 {
						style = tcell.StyleDefault.Foreground(th.MenuKeyForeground).Background(th.MiniBackground)
					}
				}
				s.SetContent(x, y, ch, nil, style)
			}
		}
		s.Show()
		return
	}
	// status mode indicator
	modeTag := "<N>"
	switch overlay {
	case OverlaySearch:
		modeTag = "<S>"
	case OverlayMenu:
		modeTag = "<M>"
	default:
		switch mode {
		case ModeInsert:
			modeTag = "<I>"
		case ModeVisual:
			modeTag = "<V>"
		case ModeMultiEdit:
			modeTag = "<ME>"
		default:
			modeTag = "<N>"
		}
	}
	status := modeTag + "  " + display + " — Press Ctrl+Q to exit"
	status = appendMacroStatus(status, macroStatus)
	status = truncateStatusForIndicator(status, width, macroStatus)
	// Colorize mode indicators: <N>, <V>, <I> match cursor; <M>=orange; <S>=red
	modeColor := th.StatusForeground
	switch overlay {
	case OverlaySearch:
		modeColor = tcell.ColorRed
	case OverlayMenu:
		modeColor = tcell.ColorOrange
	default:
		// match cursor color for mode
		modeColor = cursorColor
	}
	statusRow := height - 1
	for i, r := range status {
		style := tcell.StyleDefault.Foreground(th.StatusForeground).Background(th.StatusBackground)
		if i < len(modeTag) {
			style = tcell.StyleDefault.Foreground(modeColor).Background(th.StatusBackground).Attributes(tcell.AttrBold)
		}
		s.SetContent(i, statusRow, r, nil, style)
	}
	drawMacroRecordingIndicator(s, th, statusRow, width, macroStatus)
	// draw mini-buffer lines just above status bar
	for i, line := range minibuf {
		y := height - statusLineCount - mbHeight + i
		runes := []rune(line)
		// default style for mini-buffer text
		defStyle := tcell.StyleDefault.Foreground(th.MiniForeground).Background(th.MiniBackground)
		// special-case mnemonic menu entry lines: " <key> - <name>"
		isMnemonic := len(runes) >= 4 && runes[0] == ' ' && runes[2] == ' ' && runes[3] == '-'
		for x := 0; x < width; x++ {
			ch := ' '
			style := defStyle
			if x < len(runes) {
				ch = runes[x]
				if isMnemonic && x == 1 { // color the key selector
					style = tcell.StyleDefault.Foreground(th.MenuKeyForeground).Background(th.MiniBackground)
				}
			}
			s.SetContent(x, y, ch, nil, style)
		}
	}
	s.Show()
}

const macroRecordingIndicator = "<R>"

func isMacroRecording(macroStatus string) bool {
	return strings.HasPrefix(macroStatus, "Recording macro")
}

func truncateStatusForIndicator(status string, width int, macroStatus string) string {
	if width <= 0 {
		return ""
	}
	maxWidth := width
	if isMacroRecording(macroStatus) {
		indicatorRunes := len([]rune(macroRecordingIndicator))
		if width > indicatorRunes {
			maxWidth = width - indicatorRunes
		} else {
			maxWidth = 0
		}
	}
	statusRunes := []rune(status)
	if len(statusRunes) > maxWidth {
		status = string(statusRunes[:maxWidth])
	}
	return status
}

func drawMacroRecordingIndicator(s tcell.Screen, th config.Theme, row int, width int, macroStatus string) {
	if !isMacroRecording(macroStatus) {
		return
	}
	indicatorRunes := []rune(macroRecordingIndicator)
	if width < len(indicatorRunes) {
		return
	}
	startX := width - len(indicatorRunes)
	style := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(th.StatusBackground)
	for i, r := range indicatorRunes {
		s.SetContent(startX+i, row, r, nil, style)
	}
}
