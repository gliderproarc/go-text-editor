package app

import (
	"fmt"
	"os"

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
				{key: 'a', name: "save as", action: func() bool {
					r.runSaveAsPrompt()
					return false
				}},
			},
		},
		{
			key:  'p',
			name: "spell",
			children: []*mnemonicNode{
				{key: 't', name: "toggle", action: func() bool {
					if r.Spell != nil && r.Spell.Enabled {
						r.DisableSpellCheck()
						r.showDialog("Spell checking disabled")
						return false
					}
					cmd := os.Getenv("TEXTEDITOR_SPELL")
					var err error
					if cmd != "" {
						err = r.EnableSpellCheck(cmd)
					} else {
						if err = r.EnableSpellCheck("./aspellbridge"); err != nil {
							err = r.EnableSpellCheck("./spellmock")
						}
					}
					if err != nil {
						r.showDialog("Spell enable failed: " + err.Error())
					} else {
						r.showDialog("Spell checking enabled")
					}
					return false
				}},
				{key: 'r', name: "recheck", action: func() bool { r.updateSpellAsync(); return false }},
				{key: 'c', name: "check word", action: func() bool { r.CheckWordAtCursor(); return false }},
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
			key:  'c',
			name: "clipboard",
			children: []*mnemonicNode{
				{key: 'c', name: "cycle kill ring", action: func() bool { r.runKillRingCycle(); return false }},
			},
		},
		{
			key:  'm',
			name: "menu",
			children: []*mnemonicNode{
				{key: 'c', name: "command menu", action: func() bool { return r.runCommandMenu() }},
				{key: 'm', name: "multi-edit", action: func() bool { r.toggleMultiEdit(); return false }},
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
		{key: 'h', name: "toggle help", action: func() bool {
			r.ShowHelp = !r.ShowHelp
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
	// Show menu overlay so status bar displays <M>
	r.Overlay = OverlayMenu
	defer func() { r.Overlay = OverlayNone }()
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
			case r.isCancelKey(kev):
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
