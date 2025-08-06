package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

// drawUI renders the main UI: a centered message and a status bar.
func drawUI(s tcell.Screen) {
	width, height := s.Size()

	// Center message
	msg := "TextEditor: No File"
	msgX := (width - len(msg)) / 2
	msgY := height / 2
	for i, r := range msg {
		s.SetContent(msgX+i, msgY, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}

	// Status bar at bottom
	status := "Press Ctrl+Q to exit"
	sbX := (width - len(status)) / 2
	sbY := height - 1
	for i, r := range status {
		s.SetContent(sbX+i, sbY, r, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
	}
	s.Show()
}

// drawHelp renders a help screen.
func drawHelp(s tcell.Screen) {
	width, height := s.Size()
	s.Clear()
	s.SetStyle(tcell.StyleDefault)
	lines := []string{
		"Help:",
		"- Press Ctrl+Q or 'q' to exit",
		"- Press 'h' for help",
	}
	y := (height - len(lines)) / 2
	for i, line := range lines {
		x := (width - len(line)) / 2
		for j, r := range line {
			s.SetContent(x+j, y+i, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
		}
	}
	s.Show()
}

// editorApp represents a minimal editor application.
// It initializes a terminal screen using tcell, renders a
// status bar and waits for a Ctrl+Q keypress to exit.
func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating screen: %v\n", err)
		os.Exit(1)
	}
	if err = s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer s.Fini()

	s.SetStyle(tcell.StyleDefault)
	s.Clear()

	showHelp := false

	// Initial draw
	drawUI(s)

	for {
		ev := s.PollEvent()
		if showHelp {
			// any key exits help and redraw normal UI
			showHelp = false
			drawUI(s)
			continue
		}
		switch ev := ev.(type) {
		case *tcell.EventKey:
			// Ctrl+Q checks rune 'q' with ModCtrl
			if ev.Key() == tcell.KeyRune && ev.Rune() == 'q' && ev.Modifiers() == tcell.ModCtrl {
				return
			}
			// On Windows, Ctrl+Q might show as KeyCtrlQ
			if ev.Key() == tcell.KeyCtrlQ {
				return
			}
			// Show help on 'h'
			if ev.Key() == tcell.KeyRune && ev.Rune() == 'h' {
				showHelp = true
				drawHelp(s)
				continue
			}
		case *tcell.EventResize:
			s.Sync()
			if showHelp {
				drawHelp(s)
			} else {
				drawUI(s)
			}
		}
	}
}
