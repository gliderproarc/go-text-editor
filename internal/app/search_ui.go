package app

import (
	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

// runSearchPrompt runs a simple modal prompt at the status line allowing the
// user to type a query. Typing updates the prompt; Enter jumps to the current
// match; Esc cancels. This is a synchronous helper that polls events from the
// runner's screen.
func (r *Runner) runSearchPrompt() {
	if r.Screen == nil {
		return
	}
	s := r.Screen
	query := ""
	for {
		// compute highlight ranges for current query
		text := r.Buf.String()
		var ranges []search.Range
		if query != "" {
			ranges = search.SearchAll(text, query)
		}
		// redraw buffer (with highlights) and draw prompt
		drawBuffer(s, r.Buf, r.FilePath, ranges)
		_, height := s.Size()
		prompt := "Search: " + query
		for i, ch := range prompt {
			s.SetContent(i, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
		}
		s.Show()

		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			// Cancel
			if ev.Key() == tcell.KeyEsc {
				// redraw main view without highlights
				drawBuffer(s, r.Buf, r.FilePath, nil)
				return
			}
			// Accept
			if ev.Key() == tcell.KeyEnter {
				// perform search and jump to next match
				if query == "" {
					return
				}
				text := r.Buf.String()
				ranges := search.SearchAll(text, query)
				if len(ranges) == 0 {
					// no match: keep prompt open
					continue
				}
				bytePos := runeIndexToByteOffset(text, r.Cursor)
				idx := search.SearchNext(ranges, bytePos)
				if idx >= 0 && idx < len(ranges) {
					// move cursor to start of match (convert bytes->runes)
					r.Cursor = byteOffsetToRuneIndex(text, ranges[idx].Start)
					// after jumping we redraw and return
					drawBuffer(s, r.Buf, r.FilePath, nil)
					return
				}
			}
			// Backspace
			if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
				if len(query) > 0 {
					query = query[:len(query)-1]
				}
				continue
			}
			// Type
			if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
				query += string(ev.Rune())
				continue
			}
		}
	}
}

// runeIndexToByteOffset converts a rune index (count of runes from start) to a byte offset in s.
func runeIndexToByteOffset(s string, runeIndex int) int {
	runes := []rune(s)
	if runeIndex <= 0 {
		return 0
	}
	if runeIndex >= len(runes) {
		return len(s)
	}
	return len(string(runes[:runeIndex]))
}

// byteOffsetToRuneIndex converts a byte offset into s to the corresponding rune index.
func byteOffsetToRuneIndex(s string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(s) {
		return len([]rune(s))
	}
	return len([]rune(s[:byteOffset]))
}
