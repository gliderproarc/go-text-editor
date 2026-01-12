package app

import "github.com/gdamore/tcell/v2"

// runQuitPrompt shows a confirmation mini-buffer when the buffer is dirty.
// It returns true if the user confirms quit.
func (r *Runner) runQuitPrompt() bool {
	if r.Screen == nil {
		return true
	}
	r.setMiniBuffer([]string{"Unsaved changes. Quit without saving? (y/n)"})
	r.draw(nil)
	for {
		ev := r.waitEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if r.isCancelKey(ev) || (ev.Key() == tcell.KeyRune && (ev.Rune() == 'n' || ev.Rune() == 'N')) {
				r.clearMiniBuffer()
				r.draw(nil)
				return false
			}
			if ev.Key() == tcell.KeyRune && (ev.Rune() == 'y' || ev.Rune() == 'Y') {
				r.clearMiniBuffer()
				return true
			}
		}
	}
}
