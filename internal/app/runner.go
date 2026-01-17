package app

import (
	"os"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/config"
	"example.com/texteditor/pkg/editor"
	"example.com/texteditor/pkg/history"
	"example.com/texteditor/pkg/logs"
	"example.com/texteditor/pkg/plugins"
	"example.com/texteditor/pkg/search"
	"github.com/gdamore/tcell/v2"
)

// Mode represents the current editor mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
	ModeMultiEdit
)

// Overlay represents transient UI states that override status indicator
// such as search or menu prompts shown in the mini-buffer.
type Overlay int

const (
	OverlayNone Overlay = iota
	OverlaySearch
	OverlayMenu
)

// Runner owns the terminal lifecycle and a minimal event loop.
type Runner struct {
	Screen            tcell.Screen
	FilePath          string
	Buf               *buffer.GapBuffer
	Cursor            int // cursor position in runes
	CursorLine        int // 0-based current line index (maintained incrementally)
	TopLine           int // first visible line index
	Dirty             bool
	Ed                *editor.Editor
	ShowHelp          bool
	Mode              Mode
	VisualStart       int
	VisualLine        bool
	History           *history.History
	KillRing          history.KillRing
	Logger            *logs.Logger
	MiniBuf           []string
	Keymap            map[string]config.Keybinding
	Theme             config.Theme
	themeList         []themeEntry
	themeIndex        int
	EventCh           chan tcell.Event
	RenderCh          chan renderState
	PendingG          bool
	PendingD          bool
	PendingY          bool
	PendingC          bool
	PendingTextObject bool
	TextObjectAround  bool
	PendingCount      int
	lastChange        *repeatableChange
	insertCapture     *insertCapture
	Syntax            plugins.Highlighter
	syntaxSrc         string
	syntaxCache       []search.Range
	// Async syntax highlighting state
	SyntaxAsync *SyntaxState
	// current transient overlay (search/menu) to inform status bar
	Overlay Overlay
	// Spell checking subsystem state
	Spell *SpellState
	// Macro recording info shown in status bar
	MacroStatus string
	// Multi-edit mode state (nil when inactive)
	MultiEdit *multiEditState
	// Monotonic edit sequence; increments on any buffer mutation (insert/delete/undo/redo).
	editSeq int64
	// Last yank (paste) range for yank-pop.
	lastYankStart int
	lastYankEnd   int
	lastYankCount int
	lastYankValid bool
	// True while performing yank/paste operations to avoid clearing yank state.
	yankInProgress bool
	// Macro recording/playback state.
	macroRegisters      map[string][]macroEvent
	macroRecording      bool
	macroRecordRegister string
	macroPendingRecord  bool
	macroPendingPlay    bool
	macroPlayback       []macroEvent
	macroPlaying        bool
	macroLastRegister   string
}

func (r *Runner) setMiniBuffer(lines []string) {
	r.MiniBuf = lines
}

func (r *Runner) clearMiniBuffer() {
	r.MiniBuf = nil
}

func (r *Runner) isCancelKey(ev *tcell.EventKey) bool {
	return ev.Key() == tcell.KeyEsc || ev.Key() == tcell.KeyCtrlG || (ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == tcell.ModCtrl)
}

func (r *Runner) isInsertMode() bool {
	return r.Mode == ModeInsert || r.Mode == ModeMultiEdit
}

// waitEvent returns the next terminal event either from the runner's event
// channel (if present) or by polling the screen directly. It blocks until an
// event is available or returns nil if no screen is attached.
func (r *Runner) waitEvent() tcell.Event {
	if r.macroPlaying {
		if ev, ok := r.consumeMacroEvent(); ok {
			return ev
		}
	}
	var ev tcell.Event
	if r.EventCh != nil {
		var ok bool
		ev, ok = <-r.EventCh
		if !ok {
			return nil
		}
	} else if r.Screen != nil {
		ev = r.Screen.PollEvent()
	}
	if kev, ok := ev.(*tcell.EventKey); ok {
		if r.shouldRecordMacroEvent(kev) {
			r.recordMacroEvent(kev)
		}
	}
	return ev
}

// New creates an empty Runner.
func New() *Runner {
	ed := editor.New()
	bs := editor.BufferState{Buf: buffer.NewGapBuffer(0)}
	ed.AddBuffer(bs)
	// Start with terminal-compliant theme so the app follows terminal colors
	return &Runner{
		Buf:            bs.Buf,
		History:        history.New(),
		Mode:           ModeNormal,
		VisualStart:    -1,
		Keymap:         config.DefaultKeymap(),
		Theme:          config.TerminalTheme(),
		Ed:             ed,
		lastYankStart:  -1,
		lastYankEnd:    -1,
		macroRegisters: map[string][]macroEvent{},
	}
}

// cursorLine returns the current 0-based line index of the cursor.
func (r *Runner) cursorLine() int {
	if r.Buf == nil {
		return 0
	}
	// Fallback calculation (avoids expensive Slice by using RuneAt)
	line := 0
	for i := 0; i < r.Cursor && i < r.Buf.Len(); i++ {
		if r.Buf.RuneAt(i) == '\n' {
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
	// Use maintained CursorLine to avoid rescanning the buffer each draw.
	line := r.CursorLine
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
	r.syntaxSrc = ""
	// Initialize CursorLine from cached lines
	if r.Buf != nil {
		lines := r.Buf.Lines()
		if len(lines) > 0 {
			r.CursorLine = len(lines) - 1
		} else {
			r.CursorLine = 0
		}
	} else {
		r.CursorLine = 0
	}
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

// recomputeCursorLine recalculates CursorLine from Cursor and buffer contents.
// It uses the cached line slice for efficiency and should be called sparingly
// when Cursor is set directly without incremental updates.
func (r *Runner) recomputeCursorLine() {
	if r.Buf == nil {
		r.CursorLine = 0
		return
	}
	lines := r.Buf.Lines()
	idx := 0
	runesSoFar := 0
	for idx < len(lines) {
		lineRunes := len([]rune(lines[idx]))
		if r.Cursor <= runesSoFar+lineRunes { // cursor within this line
			r.CursorLine = idx
			return
		}
		// account for newline rune between lines
		runesSoFar += lineRunes + 1
		idx++
	}
	// if beyond last line, clamp to last
	if len(lines) > 0 {
		r.CursorLine = len(lines) - 1
	} else {
		r.CursorLine = 0
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
		ev := r.waitEvent()
		if ev == nil {
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
