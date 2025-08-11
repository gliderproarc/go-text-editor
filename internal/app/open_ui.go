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
		lines := []string{"Open: " + input}
		if errMsg != "" {
			lines = append(lines, errMsg)
		} else {
			lines = append(lines, "Enter to open, Esc to cancel")
		}
		r.setMiniBuffer(lines)
		r.draw(nil)

		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			// Cancel
			if ev.Key() == tcell.KeyEsc {
				r.clearMiniBuffer()
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
				r.clearMiniBuffer()
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
