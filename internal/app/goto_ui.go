package app

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// runGoToPrompt prompts for a line number and moves the cursor to the start of that line.
// Triggered by Alt+G (or could be Ctrl+_ mapping later).
func (r *Runner) runGoToPrompt() {
	if r.Screen == nil {
		return
	}
	input := ""
	for {
		r.setMiniBuffer([]string{"Go to line: " + input})
		r.draw(nil)

		ev := r.waitEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEsc {
				r.clearMiniBuffer()
				r.draw(nil)
				return
			}
			if ev.Key() == tcell.KeyEnter {
				n := 0
				if input != "" {
					v, err := strconv.Atoi(strings.TrimSpace(input))
					if err == nil && v > 0 {
						n = v
					}
				}
				if n <= 0 {
					continue
				}
				text := r.Buf.String()
				lines := strings.Split(text, "\n")
				if n > len(lines) {
					n = len(lines)
				}
				bytePos := 0
				for i := 0; i < n-1 && i < len(lines); i++ {
					bytePos += len(lines[i]) + 1
				}
				r.Cursor = byteOffsetToRuneIndex(text, bytePos)
				r.clearMiniBuffer()
				r.draw(nil)
				return
			}
			if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
				if len(input) > 0 {
					input = input[:len(input)-1]
				}
				continue
			}
			if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
				input += string(ev.Rune())
				continue
			}
		}
	}
}
