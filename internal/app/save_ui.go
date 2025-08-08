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
    s := r.Screen
    input := r.FilePath // prefill with current path if any
    errMsg := ""
    for {
        drawBuffer(s, r.Buf, r.FilePath, nil)
        width, height := s.Size()
        // Clear status line
        for i := 0; i < width; i++ {
            s.SetContent(i, height-1, ' ', nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
        }
        prompt := "Save As: " + input
        for i, ch := range prompt {
            s.SetContent(i, height-1, ch, nil, tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
        }
        if errMsg != "" {
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
            if ev.Key() == tcell.KeyEsc {
                drawBuffer(s, r.Buf, r.FilePath, nil)
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
                drawBuffer(s, r.Buf, r.FilePath, nil)
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

