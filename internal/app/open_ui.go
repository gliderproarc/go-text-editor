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
        // redraw buffer and draw prompt/status
        drawBuffer(s, r.Buf, r.FilePath, nil)
        width, height := s.Size()
        // Clear status line
        for i := 0; i < width; i++ {
            s.SetContent(i, height-1, ' ', nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
        }
        prompt := "Open: " + input
        for i, ch := range prompt {
            s.SetContent(i, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
        }
        if errMsg != "" {
            // show error right-aligned
            start := width - len([]rune(errMsg))
            if start < len([]rune(prompt))+1 {
                start = len([]rune(prompt)) + 1
            }
            idx := 0
            for _, ch := range errMsg {
                if start+idx < width {
                    s.SetContent(start+idx, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorWhite))
                }
                idx++
            }
        }
        s.Show()

        ev := s.PollEvent()
        switch ev := ev.(type) {
        case *tcell.EventKey:
            // Cancel
            if ev.Key() == tcell.KeyEsc {
                drawBuffer(s, r.Buf, r.FilePath, nil)
                return
            }
            // Accept
            if ev.Key() == tcell.KeyEnter {
                path := input
                if path == "" {
                    errMsg = "path required"
                    continue
                }
                if err := r.LoadFile(path); err != nil {
                    errMsg = err.Error()
                    continue
                }
                drawBuffer(s, r.Buf, r.FilePath, nil)
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
