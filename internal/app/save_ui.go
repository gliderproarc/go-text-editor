package app

import (
	"os"

	"github.com/gdamore/tcell/v2"
)

// SaveAs writes the current buffer to the given path and updates FilePath.
func (r *Runner) SaveAs(path string) error {
	if path == "" {
		return os.ErrInvalid
	}
	r.FilePath = path
	return r.Save()
}

// runSaveAsPrompt prompts for a file path and saves the current buffer there.
// Esc cancels; Enter attempts to write. Overwrites existing files.
func (r *Runner) runSaveAsPrompt() {
	if r.Screen == nil {
		return
	}
	input := r.FilePath // prefill with current path if any
	errMsg := ""
	for {
		lines := []string{"Save As: " + input}
		if errMsg != "" {
			lines = append(lines, errMsg)
		} else {
			lines = append(lines, "Enter to save, Esc to cancel")
		}
		r.setMiniBuffer(lines)
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
				if input == "" {
					errMsg = "path required"
					continue
				}
				if err := r.SaveAs(input); err != nil {
					errMsg = err.Error()
					continue
				}
				r.clearMiniBuffer()
				r.showDialog("Saved " + input)
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
				errMsg = ""
				continue
			}
		}
	}
}
