package app

import (
	"os"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"example.com/texteditor/pkg/editor"
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
	TopLine     int // first visible line index
	Dirty       bool
	Ed          *editor.Editor
	ShowHelp    bool
	Mode        Mode
	VisualStart int
	History     *history.History
	KillRing    history.KillRing
	Logger      *logs.Logger
	MiniBuf     []string
	Keymap      map[string]config.Keybinding
	EventCh     chan tcell.Event
	RenderCh    chan renderState
}

func (r *Runner) setMiniBuffer(lines []string) {
	r.MiniBuf = lines
}

func (r *Runner) clearMiniBuffer() {
	r.MiniBuf = nil
}

// waitEvent returns the next terminal event either from the runner's event
// channel (if present) or by polling the screen directly. It blocks until an
// event is available or returns nil if no screen is attached.
func (r *Runner) waitEvent() tcell.Event {
	if r.EventCh != nil {
		return <-r.EventCh
	}
	if r.Screen != nil {
		return r.Screen.PollEvent()
	}
	return nil
}

// New creates an empty Runner.
func New() *Runner {
	ed := editor.New()
	bs := editor.BufferState{Buf: buffer.NewGapBuffer(0)}
	ed.AddBuffer(bs)
	return &Runner{Buf: bs.Buf, History: history.New(), Mode: ModeNormal, VisualStart: -1, Keymap: config.DefaultKeymap(), Ed: ed}
}

// cursorLine returns the current 0-based line index of the cursor.
func (r *Runner) cursorLine() int {
	if r.Buf == nil {
		return 0
	}
	line := 0
	for i := 0; i < r.Cursor && i < r.Buf.Len(); i++ {
		if string(r.Buf.Slice(i, i+1)) == "\n" {
			line++
		}
	}
	return line
}

// ensureCursorVisible adjusts TopLine so the cursor lies within the viewport.
func (r *Runner) ensureCursorVisible() {
	if r.Screen == nil {
		return
	}
	_, height := r.Screen.Size()
	mbHeight := len(r.MiniBuf)
	maxLines := height - 1 - mbHeight
	if maxLines <= 0 {
		maxLines = 1
	}
	line := r.cursorLine()
	if line < r.TopLine {
		r.TopLine = line
	} else if line >= r.TopLine+maxLines {
		r.TopLine = line - maxLines + 1
	}
	if r.TopLine < 0 {
		r.TopLine = 0
	}
}

func (r *Runner) saveBufferState() {
	if r.Ed == nil {
		return
	}
	r.Ed.UpdateCurrent(editor.BufferState{FilePath: r.FilePath, Buf: r.Buf, Cursor: r.Cursor, Dirty: r.Dirty})
}

// LoadFile loads a file into the runner's buffer.
func (r *Runner) LoadFile(path string) error {
	if path == "" {
		return nil
	}
	if r.Logger != nil {
		r.Logger.Event("open.attempt", map[string]any{"file": path})
	}
	if r.Ed == nil {
		r.Ed = editor.New()
		r.Ed.AddBuffer(editor.BufferState{FilePath: r.FilePath, Buf: r.Buf, Cursor: r.Cursor, Dirty: r.Dirty})
	}
	r.saveBufferState()
	bs, err := r.Ed.LoadFile(path)
	if err != nil {
		if r.Logger != nil {
			r.Logger.Event("open.error", map[string]any{"file": path, "error": err.Error()})
		}
		return err
	}
	r.FilePath = bs.FilePath
	r.Buf = bs.Buf
	r.Cursor = bs.Cursor
	r.Dirty = bs.Dirty
	if r.Logger != nil {
		r.Logger.Event("open.success", map[string]any{"file": path, "runes": r.Buf.Len(), "bytes": len([]byte(r.Buf.String()))})
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
	r.saveBufferState()
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
	// set up channels for asynchronous events and rendering
	r.EventCh = make(chan tcell.Event, 10)
	r.RenderCh = make(chan renderState, 1)

	// poll events in a separate goroutine
	go func() {
		for {
			ev := r.Screen.PollEvent()
			if ev == nil {
				close(r.EventCh)
				return
			}
			r.EventCh <- ev
		}
	}()

	// renderer goroutine consumes snapshots and draws them
	go func() {
		for st := range r.RenderCh {
			renderToScreen(r.Screen, st)
		}
	}()

	// initial draw
	r.draw(nil)

	for {
		ev, ok := <-r.EventCh
		if !ok {
			return nil
		}
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
			if r.ShowHelp {
				r.ShowHelp = false
				r.draw(nil)
				continue
			}
			if r.handleKeyEvent(ev) {
				if r.Logger != nil {
					r.Logger.Event("action", map[string]any{"name": "quit"})
				}
				return nil
			}
		case *tcell.EventResize:
			r.Screen.Sync()
			r.draw(nil)
		}
	}
}
