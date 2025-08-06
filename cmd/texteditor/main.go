package main

import (
    "fmt"
    "os"

    "github.com/gdamore/tcell/v2"
)

// editorApp represents a minimal editor application.
// It initializes a terminal screen using tcell, renders a
// status bar and waits for a Ctrl+Q keypress to exit.
// Future enhancements will add file loading, editing, etc.
func main() {
    // Initialize the tcell screen.
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

    // Ensure the screen uses a clean exit.
    s.SetStyle(tcell.StyleDefault)
    s.Clear()

    // Draw a simple UI: a center message and a status bar.
    width, height := s.Size()

    // Center message
    msg := "TextEditor: No File"
    msgX := (width - len(msg)) / 2
    msgY := height / 2
    for i, r := range msg {
        s.SetContent(msgX+i, msgY, tcell.Rune(r), nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
    }

    // Status bar at bottom
    status := "Press Ctrl+Q to exit"
    sbX := (width - len(status)) / 2
    sbY := height - 1
    for i, r := range status {
        s.SetContent(sbX+i, sbY, tcell.Rune(r), nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
    }
    s.Show()

    // Event loop
    for {
        ev := s.PollEvent()
        switch ev := ev.(type) {
        case *tcell.EventKey:
            // Ctrl+Q checks rune 'q' with ModCtrl
            if ev.Key() == tcell.KeyRune && ev.Rune() == 'q' && ev.Modifiers() == tcell.ModCtrl {
                // Clean exit
                return
            }
            // On Windows, Ctrl+Q might show as KeyCtrlQ
            if ev.Key() == tcell.KeyCtrlQ {
                return
            }
            // For debugging: show pressed key
            // Commented out to keep UI clean
            // s.SetContent(0, height-2, rune(ev.Rune()), nil, tcell.StyleDefault)
        case *tcell.EventResize:
            s.Sync()
        }
    }
}
