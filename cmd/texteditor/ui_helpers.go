package main

import (
    "github.com/gdamore/tcell/v2"
)

// drawUI renders the main UI: a centered message and a status bar.
func drawUI(s tcell.Screen) {
    width, height := s.Size()
    msg := "TextEditor: No File"
    msgX := (width - len(msg)) / 2
    msgY := height / 2
    for i, r := range msg {
        s.SetContent(msgX+i, msgY, r, nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
    }
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

