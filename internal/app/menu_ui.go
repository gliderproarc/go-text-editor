package app

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

type command struct {
	name   string
	action func() bool
}

func (r *Runner) commandList() []command {
    return []command{
        {name: "open file", action: func() bool { r.runOpenPrompt(); return false }},
        {name: "save", action: func() bool {
            if r.FilePath == "" {
                r.runSaveAsPrompt()
            } else {
                if err := r.Save(); err == nil {
                    r.showDialog("Saved " + r.FilePath)
                }
            }
            if r.Logger != nil {
                r.Logger.Event("action", map[string]any{"name": "save", "file": r.FilePath})
            }
            return false
        }},
        {name: "theme: next", action: func() bool { r.NextTheme(); return false }},
        {name: "theme: previous", action: func() bool { r.PrevTheme(); return false }},
        {name: "search", action: func() bool { r.runSearchPrompt(); return false }},
        {name: "go to line", action: func() bool { r.runGoToPrompt(); return false }},
        {name: "help", action: func() bool { r.ShowHelp = true; r.draw(nil); return false }},
        {name: "quit", action: func() bool {
            if r.Dirty {
                return r.runQuitPrompt()
            }
            return true
        }},
    }
}

// runCommandMenu opens a mini-buffer menu listing commands. It supports
// fuzzy filtering by typing and navigation with Ctrl+P/Ctrl+N. Enter executes
// the highlighted command. It returns true if the command requests to quit.
func (r *Runner) runCommandMenu() bool {
	if r.Screen == nil {
		return false
	}
	cmds := r.commandList()
	query := ""
	sel := 0
	filtered := cmds
	for {
		if query != "" {
			tmp := make([]command, 0, len(cmds))
			for _, c := range cmds {
				if strings.Contains(strings.ToLower(c.name), strings.ToLower(query)) {
					tmp = append(tmp, c)
				}
			}
			filtered = tmp
		} else {
			filtered = cmds
		}
		if len(filtered) == 0 {
			filtered = []command{}
		}
		if sel >= len(filtered) {
			sel = len(filtered) - 1
		}
		if sel < 0 {
			sel = 0
		}
		lines := []string{"Command: " + query}
		// show up to first 10 commands
		max := len(filtered)
		if max > 10 {
			max = 10
		}
		for i := 0; i < max; i++ {
			prefix := "  "
			if i == sel {
				prefix = "> "
			}
			lines = append(lines, prefix+filtered[i].name)
		}
		r.setMiniBuffer(lines)
		r.draw(nil)

		ev := r.waitEvent()
		if ev == nil {
			r.clearMiniBuffer()
			r.draw(nil)
			return false
		}
		if kev, ok := ev.(*tcell.EventKey); ok {
			switch {
			case kev.Key() == tcell.KeyEsc:
				r.clearMiniBuffer()
				r.draw(nil)
				return false
			case kev.Key() == tcell.KeyEnter:
				r.clearMiniBuffer()
				r.draw(nil)
				if len(filtered) > 0 {
					return filtered[sel].action()
				}
				return false
			case kev.Key() == tcell.KeyBackspace || kev.Key() == tcell.KeyBackspace2:
				if len(query) > 0 {
					query = query[:len(query)-1]
					sel = 0
				}
			case kev.Key() == tcell.KeyCtrlP || (kev.Key() == tcell.KeyRune && kev.Rune() == 'p' && kev.Modifiers() == tcell.ModCtrl):
				if sel > 0 {
					sel--
				}
			case kev.Key() == tcell.KeyCtrlN || (kev.Key() == tcell.KeyRune && kev.Rune() == 'n' && kev.Modifiers() == tcell.ModCtrl):
				if sel < len(filtered)-1 {
					sel++
				}
			case kev.Key() == tcell.KeyRune && kev.Modifiers() == 0:
				query += string(kev.Rune())
				sel = 0
			}
		}
	}
}
