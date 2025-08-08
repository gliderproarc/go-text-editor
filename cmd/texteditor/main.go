package main

import (
	"fmt"
	"os"
	"strings"

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

// drawFile renders the given lines of text to the screen. It will
// draw lines starting at the top-left and truncate to the screen size.
func drawFile(s tcell.Screen, fname string, lines []string) {
	width, height := s.Size()
	s.Clear()
	// Reserve last line for status bar
	maxLines := height - 1
	if maxLines < 0 {
		maxLines = 0
	}
	for i := 0; i < maxLines && i < len(lines); i++ {
		line := lines[i]
		// Clamp to width
		runes := []rune(line)
		for j := 0; j < width && j < len(runes); j++ {
			s.SetContent(j, i, runes[j], nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
		}
	}
	// Status bar shows filename and exit hint
	status := fmt.Sprintf("%s — Press Ctrl+Q to exit", fname)
	if len(status) > width {
		// truncate status to fit
		status = string([]rune(status)[:width])
	}
	for i, r := range status {
		s.SetContent(i, height-1, r, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
	}
	s.Show()
}

// editorApp represents a minimal editor application.
// It initializes a terminal screen using tcell, renders a
// status bar and waits for a Ctrl+Q keypress to exit.
func main() {
	var fname string
	var fileLines []string
	if len(os.Args) > 1 {
		fname = os.Args[1]
		data, err := os.ReadFile(fname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file %s: %v\n", fname, err)
			// proceed without a file
		} else {
			fileLines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		}
	}

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
	if fileLines != nil {
		drawFile(s, fname, fileLines)
	} else {
		drawUI(s)
	}

	for {
		ev := s.PollEvent()
		if showHelp {
			// any key exits help and redraw normal UI
			showHelp = false
			if fileLines != nil {
				drawFile(s, fname, fileLines)
			} else {
				drawUI(s)
			}
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
				if fileLines != nil {
					drawFile(s, fname, fileLines)
				} else {
					drawUI(s)
				}
			}
		}
	}
}
