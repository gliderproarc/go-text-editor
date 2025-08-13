package app

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type mnemonicNode struct {
	key      rune
	name     string
	action   func() bool
	children []*mnemonicNode
}

func (r *Runner) mnemonicMenu() []*mnemonicNode {
    return []*mnemonicNode{
        {
            key:  'f',
            name: "file",
            children: []*mnemonicNode{
                {key: 'o', name: "open file", action: func() bool {
                    r.runOpenPrompt()
                    return false
                }},
                {key: 's', name: "save", action: func() bool {
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
            },
        },
        {
            key:  't',
            name: "theme",
            children: []*mnemonicNode{
                {key: 'n', name: "next", action: func() bool { r.NextTheme(); return false }},
                {key: 'p', name: "previous", action: func() bool { r.PrevTheme(); return false }},
            },
        },
        {
            key:  's',
            name: "search",
            children: []*mnemonicNode{
                {key: 's', name: "search", action: func() bool {
                    r.runSearchPrompt()
                    return false
                }},
            },
        },
		{
			key:  'g',
			name: "go to",
			children: []*mnemonicNode{
				{key: 'l', name: "line", action: func() bool {
					r.runGoToPrompt()
					return false
				}},
			},
		},
		{key: 'h', name: "help", action: func() bool {
			r.ShowHelp = true
			r.draw(nil)
			return false
		}},
		{key: 'q', name: "quit", action: func() bool {
			if r.Dirty {
				return r.runQuitPrompt()
			}
			return true
		}},
	}
}

func (r *Runner) runMnemonicMenu() bool {
	if r.Screen == nil {
		return false
	}
	root := &mnemonicNode{children: r.mnemonicMenu()}
	node := root
	path := ""
	for {
		lines := []string{"Keys: " + path}
		for _, child := range node.children {
			lines = append(lines, fmt.Sprintf(" %c - %s", child.key, child.name))
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
			case kev.Key() == tcell.KeyRune && kev.Rune() == ' ' && kev.Modifiers() == 0:
				r.clearMiniBuffer()
				r.draw(nil)
				return r.runCommandMenu()
			case kev.Key() == tcell.KeyRune && kev.Modifiers() == 0:
				ch := kev.Rune()
				var next *mnemonicNode
				for _, child := range node.children {
					if child.key == ch {
						next = child
						break
					}
				}
				if next != nil {
					if len(next.children) > 0 {
						node = next
						path += string(ch)
					} else if next.action != nil {
						r.clearMiniBuffer()
						r.draw(nil)
						return next.action()
					}
				}
			}
		}
	}
}
