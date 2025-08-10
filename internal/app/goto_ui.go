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
	s := r.Screen
	input := ""
	for {
		// redraw buffer and draw prompt
		r.draw(nil)
		_, height := s.Size()
		prompt := "Go to line: " + input
		for i, ch := range prompt {
			s.SetContent(i, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
		}
		s.Show()

		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEsc {
				r.draw(nil)
				return
			}
			if ev.Key() == tcell.KeyEnter {
				// parse input
				n := 0
				if input != "" {
					v, err := strconv.Atoi(strings.TrimSpace(input))
					if err == nil && v > 0 {
						n = v
					}
				}
				if n <= 0 {
					// invalid number; keep prompt open
					continue
				}
				// move cursor to start of line n (1-based)
				text := r.Buf.String()
				lines := strings.Split(text, "\n")
				if n > len(lines) {
					n = len(lines)
				}
				// compute rune index at start of line n
				pos := 0
				for i := 0; i < n-1 && i < len(lines); i++ {
					// +1 for the newline byte
					pos += len([]rune(lines[i])) + 1
				}
				// convert byte offset pos to rune index
				r.Cursor = byteOffsetToRuneIndex(text, pos)
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
