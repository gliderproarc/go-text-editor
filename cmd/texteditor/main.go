package main

import (
	"fmt"
	"os"

	"example.com/texteditor/internal/app"
	"example.com/texteditor/pkg/config"
	"example.com/texteditor/pkg/logs"
)

// main wires the CLI to the application runner which supports typing,
// saving, search, and other keybindings. It optionally loads a file
// provided as the first argument and starts the event loop.
func main() {
	r := app.New()
	// Initialize logger from env for CLI runs
	r.Logger = logs.NewFromEnv()

	if cfg, err := config.LoadDefault(); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
	} else {
		r.Keymap = cfg.Keymap
	}

	// Load optional file path argument
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if r.Logger != nil {
			r.Logger.Event("cli.open.arg", map[string]any{"file": arg})
		}
		if err := r.LoadFile(arg); err != nil {
			// Non-fatal: start editor with empty buffer and report error to stderr
			fmt.Fprintf(os.Stderr, "failed to load %s: %v\n", arg, err)
			if r.Logger != nil {
				r.Logger.Event("cli.open.error", map[string]any{"file": arg, "error": err.Error()})
			}
		}
	}

	if err := r.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
