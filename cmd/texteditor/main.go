package main

import (
    "fmt"
    "os"

    "example.com/texteditor/internal/app"
)

// main wires the CLI to the application runner which supports typing,
// saving, search, and other keybindings. It optionally loads a file
// provided as the first argument and starts the event loop.
func main() {
    r := app.New()

    // Load optional file path argument
    if len(os.Args) > 1 {
        if err := r.LoadFile(os.Args[1]); err != nil {
            // Non-fatal: start editor with empty buffer and report error to stderr
            fmt.Fprintf(os.Stderr, "failed to load %s: %v\n", os.Args[1], err)
        }
    }

    if err := r.Run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

