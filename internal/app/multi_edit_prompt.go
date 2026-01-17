package app

import (
	"fmt"

	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

func (r *Runner) runMultiEditPrompt() string {
	if r.Screen == nil || r.Buf == nil {
		return ""
	}
	query := ""
	for {
		text := r.Buf.String()
		var raw []search.Range
		if query != "" {
			raw = search.SearchAll(text, query)
		}
		ranges := buildSearchHighlights(raw, -1)
		lines := []string{"Multi-edit: " + query}
		if query != "" {
			if len(raw) > 0 {
				lines = append(lines, fmt.Sprintf("%d matches; Enter to start", len(raw)))
				width, height := r.Screen.Size()
				max := len(raw)
				if height > 0 {
					max = height - len(lines) - 1
				}
				if max < 1 {
					max = 1
				}
				if max > len(raw) {
					max = len(raw)
				}
				for i := 0; i < max; i++ {
					m := raw[i]
					lineNum := lineNumberForByte(text, m.Start)
					entry := fmt.Sprintf("  %d: %q", lineNum, text[m.Start:m.End])
					if width > 0 {
						runes := []rune(entry)
						if len(runes) > width {
							entry = string(runes[:width])
						}
					}
					lines = append(lines, entry)
				}
				if len(raw) > max {
					lines = append(lines, fmt.Sprintf("  … and %d more", len(raw)-max))
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
			if r.isCancelKey(ev) {
				r.clearMiniBuffer()
				r.draw(nil)
				return ""
			}
			if ev.Key() == tcell.KeyEnter {
				if query != "" && len(raw) > 0 {
					r.clearMiniBuffer()
					r.draw(nil)
					return query
				}
				continue
			}
			if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
				if len(query) > 0 {
					query = query[:len(query)-1]
				}
				continue
			}
			if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
				query += string(ev.Rune())
				continue
			}
		}
	}
}
