package app

import (
	"os"
	"strings"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"example.com/texteditor/pkg/history"
	"example.com/texteditor/pkg/logs"
	"github.com/gdamore/tcell/v2"
)

// Mode represents the current editor mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
)

// Runner owns the terminal lifecycle and a minimal event loop.
type Runner struct {
	Screen      tcell.Screen
	FilePath    string
	Buf         *buffer.GapBuffer
	Cursor      int // cursor position in runes
	Dirty       bool
	ShowHelp    bool
	Mode        Mode
	VisualStart int
	History     *history.History
	KillRing    history.KillRing
	Logger      *logs.Logger
	MiniBuf     []string
	Keymap      map[string]config.Keybinding
}

func (r *Runner) setMiniBuffer(lines []string) {
	r.MiniBuf = lines
}

func (r *Runner) clearMiniBuffer() {
	r.MiniBuf = nil
}

// New creates an empty Runner.
func New() *Runner {
	return &Runner{Buf: buffer.NewGapBuffer(0), History: history.New(), Mode: ModeNormal, VisualStart: -1, Keymap: config.DefaultKeymap()}
}

// LoadFile loads a file into the runner's buffer.
func (r *Runner) LoadFile(path string) error {
	if path == "" {
		return nil
	}
	if r.Logger != nil {
		r.Logger.Event("open.attempt", map[string]any{"file": path})
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if r.Logger != nil {
			r.Logger.Event("open.error", map[string]any{"file": path, "error": err.Error()})
		}
		return err
	}
	r.FilePath = path
	// Normalize CRLF to LF for internal buffer storage
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	r.Buf = buffer.NewGapBufferFromString(normalized)
	r.Cursor = r.Buf.Len()
	r.Dirty = false
	if r.Logger != nil {
		r.Logger.Event("open.success", map[string]any{"file": path, "bytes": len(data), "runes": r.Buf.Len()})
	}
	return nil
}

// Save writes the buffer contents to the current FilePath and clears Dirty.
func (r *Runner) Save() error {
	if r.FilePath == "" {
		return os.ErrInvalid
	}
	data := []byte(r.Buf.String())
	if err := os.WriteFile(r.FilePath, data, 0644); err != nil {
		return err
	}
	r.Dirty = false
	return nil
}

// InitScreen initializes a tcell screen if one is not already set.
func (r *Runner) InitScreen() error {
	if r.Screen != nil {
		return nil
	}
	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	s.SetStyle(tcell.StyleDefault)
	s.Clear()
	r.Screen = s
	return nil
}

// Fini finalizes the screen if initialized.
func (r *Runner) Fini() {
	if r.Screen != nil {
		r.Screen.Fini()
		r.Screen = nil
	}
	if r.Logger != nil {
		r.Logger.Close()
	}
}

// Run starts the event loop. It will initialize the screen if needed and
// return when the user requests quit (Ctrl+Q).
func (r *Runner) Run() error {
	if r.Screen == nil {
		if err := r.InitScreen(); err != nil {
			return err
		}
		defer r.Fini()
	}

	// Initialize logger from env (no-op if disabled)
	if r.Logger == nil {
		r.Logger = logs.NewFromEnv()
	}
	if r.Logger != nil {
		r.Logger.Event("run.start", map[string]any{"file": r.FilePath})
		defer r.Logger.Event("run.end", map[string]any{"file": r.FilePath})
	}

	// initial draw
	if r.Buf != nil && r.Buf.Len() > 0 {
		r.draw(nil)
	} else {
		drawUI(r.Screen)
	}

	for {
		ev := r.Screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if r.Logger != nil {
				r.Logger.Event("key", map[string]any{
					"type":      "EventKey",
					"key":       int(ev.Key()),
					"rune":      string(ev.Rune()),
					"modifiers": int(ev.Modifiers()),
				})
			}
			// If help is currently shown, consume this key to dismiss it
			if r.ShowHelp {
				r.ShowHelp = false
				if r.Buf != nil && r.Buf.Len() > 0 {
					r.draw(nil)
				} else {
					drawUI(r.Screen)
				}
				continue
			}
			// Otherwise, handle the key normally; if it requests quit, exit
			if r.handleKeyEvent(ev) {
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "quit"})
				}
				return nil
			}
		case *tcell.EventResize:
			r.Screen.Sync()
			if r.ShowHelp {
				drawHelp(r.Screen)
			} else {
				if r.Buf != nil && r.Buf.Len() > 0 {
					r.draw(nil)
				} else {
					drawUI(r.Screen)
				}
			}
		}
	}
}
