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
    // Show search overlay so status bar displays <S>
    r.Overlay = OverlaySearch
    defer func() { r.Overlay = OverlayNone }()
    query := ""
    sel := 0 // selected match index within current results
    for {
        text := r.Buf.String()
        var raw []search.Range
        if query != "" {
            raw = search.SearchAll(text, query)
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

        // decorate highlights: selected -> blue, others -> yellow
        var ranges []search.Range
        for i, rge := range raw {
            if i == sel {
                ranges = append(ranges, search.Range{Start: rge.Start, End: rge.End, Group: "bg.search.current"})
            } else {
                ranges = append(ranges, search.Range{Start: rge.Start, End: rge.End, Group: "bg.search"})
            }
        }

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
                    // compute 1-based line number by counting newlines before start
                    prefix := text[:m.Start]
                    ln := 1
                    for j := 0; j < len(prefix); j++ {
                        if prefix[j] == '\n' {
                            ln++
                        }
                    }
                    prefixStr := "  "
                    if i == sel {
                        prefixStr = "> "
                    }
                    entry := fmt.Sprintf("%s%d: %q", prefixStr, ln, text[m.Start:m.End])
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
            if ev.Key() == tcell.KeyEsc {
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
