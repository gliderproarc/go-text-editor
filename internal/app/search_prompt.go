package app

import (
	"fmt"

	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

// runSearchPrompt runs a simple modal prompt at the status line allowing the
// user to type a query. Typing updates the prompt; Enter jumps to the current
// match; Esc or Ctrl+G cancels. This is a synchronous helper that polls events from the
// runner's screen.
func (r *Runner) runSearchPrompt() {
	r.runSearchPromptCase(true)
}

// runSearchPromptCase runs the search prompt with optional case sensitivity.
func (r *Runner) runSearchPromptCase(caseSensitive bool) {
	if r.Screen == nil {
		return
	}
	// Show search overlay so status bar displays <S>
	r.Overlay = OverlaySearch
	defer func() { r.Overlay = OverlayNone }()
	query := ""
	sel := 0 // selected match index within current results
	for {
		text := r.Buf.String()
		var raw []search.Range
		if query != "" {
			raw = search.SearchAllCase(text, query, caseSensitive)
		}
		// compute default selection relative to cursor when needed
		if query == "" || len(raw) == 0 {
			sel = 0
		} else if sel < 0 || sel >= len(raw) {
			// clamp and/or recompute next from cursor
			bytePos := runeIndexToByteOffset(text, r.Cursor)
			idx := search.SearchNext(raw, bytePos)
			if idx >= 0 {
				sel = idx
			} else {
				sel = 0
			}
		}

		ranges := buildSearchHighlights(raw, sel)

		// build minibuffer lines with match list (show up to 10)
		lines := []string{"Search: " + query}
		if query != "" {
			if len(raw) > 0 {
				lines = append(lines, fmt.Sprintf("%d matches; use Ctrl+N/P or arrows", len(raw)))
				// show matches
				max := len(raw)
				if max > 10 {
					max = 10
				}
				// prepare optional width for truncation
				width, _ := r.Screen.Size()
				for i := 0; i < max; i++ {
					m := raw[i]
					entry := buildSearchPromptLine(text, m, i == sel)
					// truncate to screen width
					if width > 0 {
						if rw := []rune(entry); len(rw) > width {
							entry = string(rw[:width])
						}
					}
					lines = append(lines, entry)
				}
				if len(raw) > 10 {
					lines = append(lines, fmt.Sprintf("  … and %d more", len(raw)-10))
				}
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
			if r.isCancelKey(ev) {
				// redraw main view without highlights
				r.clearMiniBuffer()
				r.draw(nil)
				return
			}

			// Accept -> jump to selected match
			if ev.Key() == tcell.KeyEnter {
				if query == "" {
					r.clearMiniBuffer()
					r.draw(nil)
					return
				}
				if len(raw) == 0 {
					continue
				}
				idx := sel
				if idx >= 0 && idx < len(raw) {
					// move cursor to start of match (convert bytes->runes)
					r.Cursor = byteOffsetToRuneIndex(text, raw[idx].Start)
					// update CursorLine from byte position (count newlines before start)
					prefix := text[:raw[idx].Start]
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
			// Navigation: Ctrl+P/Up and Ctrl+N/Down
			if (ev.Key() == tcell.KeyCtrlP) || (ev.Key() == tcell.KeyUp) || (ev.Key() == tcell.KeyRune && ev.Rune() == 'p' && ev.Modifiers() == tcell.ModCtrl) {
				if len(raw) > 0 {
					sel = (sel - 1 + len(raw)) % len(raw)
				}
				continue
			}
			if (ev.Key() == tcell.KeyCtrlN) || (ev.Key() == tcell.KeyDown) || (ev.Key() == tcell.KeyRune && ev.Rune() == 'n' && ev.Modifiers() == tcell.ModCtrl) {
				if len(raw) > 0 {
					sel = (sel + 1) % len(raw)
				}
				continue
			}
			// Backspace
			if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
				if len(query) > 0 {
					query = query[:len(query)-1]
					sel = 0
				}
				continue
			}
			// Type
			if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
				query += string(ev.Rune())
				// recompute selection to first/next match
				sel = 0
				continue
			}
		}
	}
}
