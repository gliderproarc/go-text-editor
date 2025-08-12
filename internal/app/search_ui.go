package app

import (
	"fmt"

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
	query := ""
	for {
		text := r.Buf.String()
		var ranges []search.Range
		if query != "" {
			ranges = search.SearchAll(text, query)
		}
		lines := []string{"Search: " + query}
		if query != "" {
			if len(ranges) > 0 {
				lines = append(lines, fmt.Sprintf("%d matches", len(ranges)))
			} else {
				lines = append(lines, "No matches")
			}
		}
		r.setMiniBuffer(lines)
		r.draw(ranges)

		ev := r.waitEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			// Cancel
			if ev.Key() == tcell.KeyEsc {
				// redraw main view without highlights
				r.clearMiniBuffer()
				r.draw(nil)
				return
			}
			// Accept
			if ev.Key() == tcell.KeyEnter {
				// perform search and jump to next match
				if query == "" {
					r.clearMiniBuffer()
					r.draw(nil)
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
					// update CursorLine from byte position (count newlines before start)
					prefix := text[:ranges[idx].Start]
					count := 0
					for i := 0; i < len(prefix); i++ {
						if prefix[i] == '\n' {
							count++
						}
					}
					r.CursorLine = count
					// after jumping we redraw and return
					r.clearMiniBuffer()
					r.draw(nil)
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
