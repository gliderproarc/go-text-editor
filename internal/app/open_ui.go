package app

import (
	"github.com/gdamore/tcell/v2"
)

// runOpenPrompt prompts for a file path and loads it into the buffer.
// Esc cancels; Enter attempts to load. On error, shows a brief message and remains in the prompt.
func (r *Runner) runOpenPrompt() {
	if r.Screen == nil {
		return
	}
	s := r.Screen
	input := ""
	errMsg := ""
	for {
		// redraw buffer and draw prompt/status
		r.draw(nil)
		width, height := s.Size()
		// Clear status line
		for i := 0; i < width; i++ {
			s.SetContent(i, height-1, ' ', nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
		}
		prompt := "Open: " + input
		for i, ch := range prompt {
			s.SetContent(i, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
		}
		if errMsg != "" {
			// show error right-aligned
			start := width - len([]rune(errMsg))
			if start < len([]rune(prompt))+1 {
				start = len([]rune(prompt)) + 1
			}
			idx := 0
			for _, ch := range errMsg {
				if start+idx < width {
					s.SetContent(start+idx, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorWhite))
				}
				idx++
			}
		}
		s.Show()

		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			// Cancel
			if ev.Key() == tcell.KeyEsc {
				r.draw(nil)
				return
			}
			// Accept
			if ev.Key() == tcell.KeyEnter {
				path := input
				if path == "" {
					errMsg = "path required"
					continue
				}
				if r.Logger != nil {
					r.Logger.Event("open.prompt.submit", map[string]any{"file": path})
				}
				if err := r.LoadFile(path); err != nil {
					errMsg = err.Error()
					if r.Logger != nil {
						r.Logger.Event("open.prompt.error", map[string]any{"file": path, "error": err.Error()})
					}
					continue
				}
				if r.Logger != nil {
					r.Logger.Event("open.prompt.success", map[string]any{"file": path})
				}
				r.draw(nil)
				return
			}
			// Backspace
			if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
				if len(input) > 0 {
					input = input[:len(input)-1]
				}
				continue
			}
			// Type
			if ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
				input += string(ev.Rune())
				errMsg = ""
				continue
			}
		}
	}
}
