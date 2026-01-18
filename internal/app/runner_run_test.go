package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/history"
	"github.com/gdamore/tcell/v2"
)

// TestRun_TypingSaveQuit_Simulation reflects README behavior: type directly, save with Ctrl+S, quit with Ctrl+Q.
func TestRun_TypingSaveQuit_Simulation(t *testing.T) {
	// Set up simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	// Temp file for save
	tmp, err := os.CreateTemp("", "texteditor_run_*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New(), FilePath: tmp.Name()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// Give the loop a moment to start
	time.Sleep(10 * time.Millisecond)

	// Enter insert mode and type 'a', 'b' (avoid 'h' which opens help per README)
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	// Save via Ctrl+S
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModCtrl))
	// Dismiss save dialog
	s.PostEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// Quit via Ctrl+Q (use rune+ctrl for portability)
	s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	// Verify buffer content and saved file per README expectations
	if got := r.Buf.String(); got != "ab" {
		t.Fatalf("expected buffer 'ab', got %q", got)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "ab" {
		t.Fatalf("expected saved content 'ab', got %q", string(data))
	}
	if r.Dirty {
		t.Fatalf("expected Dirty=false after save")
	}
}

func TestRunner_LoadFile_NormalizesCRLF(t *testing.T) {
	f, err := os.CreateTemp("", "texteditor_crlf_*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	path := f.Name()
	_, _ = f.WriteString("a\r\nb\r\n")
	_ = f.Close()
	defer os.Remove(path)

	r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
	if err := r.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	t.Logf("LoadFile OK: path=%s, buf_len=%d", path, r.Buf.Len())
	if got := r.Buf.String(); got != "a\nb\n" {
		t.Fatalf("expected normalized newlines, got %q", got)
	}
}

func TestRunner_SaveAs_WritesAndClearsDirty(t *testing.T) {
	f, err := os.CreateTemp("", "texteditor_saveas_*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	path := f.Name()
	_ = f.Close()
	defer os.Remove(path)

	r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	// type 'a','b'
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	r.handleKeyEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0))
	if !r.Dirty {
		t.Fatalf("expected Dirty after typing")
	}
	if err := r.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved: %v", err)
	}
	if string(data) != "ab" {
		t.Fatalf("expected 'ab', got %q", string(data))
	}
	if r.Dirty {
		t.Fatalf("expected Dirty=false after save")
	}
	if r.FilePath != path {
		t.Fatalf("expected FilePath %q, got %q", path, r.FilePath)
	}
}

// TestRun_OpenFilePrompt_Simulation verifies that pressing Ctrl+O opens the
// file manager; selecting a file opens it into the buffer.
func TestRun_OpenFilePrompt_Simulation(t *testing.T) {
	// Prepare a temp file to open
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "open.txt")
	content := "hello\nworld\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}

	// Set up simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// Give the loop a moment to start
	time.Sleep(10 * time.Millisecond)

	// Open file manager via dedicated control key (as emitted in real logs)
	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlO, 0, 0))
	for i := 0; i < 50; i++ {
		if r.View == ViewFileManager {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r.View != ViewFileManager {
		t.Fatalf("expected file manager view")
	}
	if r.FileManager == nil {
		t.Fatalf("expected file manager state")
	}
	r.FileManager.Dir = tmpDir
	if err := r.loadFileManagerDir(tmpDir); err != nil {
		t.Fatalf("load dir: %v", err)
	}

	// Move to file entry and open it.
	s.PostEventWait(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// Quit to end the loop
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	// Assert file got loaded
	if r.FilePath != path {
		t.Fatalf("expected FilePath=%q after open, got %q", path, r.FilePath)
	}
	if got := r.Buf.String(); got != content {
		t.Fatalf("expected buffer to equal file content, got %q", got)
	}
}

// TestRun_FileManager_OpenEntry_Simulation verifies Ctrl+O opens the file manager
// and Enter opens the first file entry.
func TestRun_FileManager_OpenEntry_Simulation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.txt")
	content := "hello file manager"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	time.Sleep(10 * time.Millisecond)

	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlO, 0, 0))
	// Wait for file manager to initialize and list entries.
	for i := 0; i < 50; i++ {
		if r.View == ViewFileManager {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r.View != ViewFileManager {
		t.Fatalf("expected file manager view")
	}
	if r.FileManager == nil {
		t.Fatalf("expected file manager state")
	}
	r.FileManager.Dir = tmpDir
	if err := r.loadFileManagerDir(tmpDir); err != nil {
		t.Fatalf("load dir: %v", err)
	}
	// Move to the next entry (first file after ..).
	s.PostEventWait(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	if r.FilePath != filePath {
		t.Fatalf("expected FilePath=%q after open, got %q", filePath, r.FilePath)
	}
	if got := r.Buf.String(); got != content {
		t.Fatalf("expected buffer to equal file content, got %q", got)
	}
}

// TestRun_FileManager_Rename_Simulation verifies renaming a file from insert mode
// prompts for confirmation and applies the change.
func TestRun_FileManager_Rename_Simulation(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "alpha.txt")
	newPath := filepath.Join(tmpDir, "beta.txt")
	if err := os.WriteFile(oldPath, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	time.Sleep(10 * time.Millisecond)

	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlO, 0, 0))
	for i := 0; i < 50; i++ {
		if r.View == ViewFileManager {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r.View != ViewFileManager {
		t.Fatalf("expected file manager view")
	}
	if r.FileManager == nil {
		t.Fatalf("expected file manager state")
	}
	r.FileManager.Dir = tmpDir
	if err := r.loadFileManagerDir(tmpDir); err != nil {
		t.Fatalf("load dir: %v", err)
	}

	// Move to file entry and enter insert mode.
	s.PostEventWait(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	// Ensure cursor is at the start of the filename field.
	if r.FileManager == nil {
		t.Fatalf("expected file manager state")
	}
	nameStart := r.fileManagerNameStartPos(1)
	for r.Cursor > nameStart {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	}

	// Remove "alpha" and type "beta".
	for i := 0; i < len("alpha"); i++ {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyDelete, 0, 0))
	}
	for _, ch := range "beta" {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}

	// Exit insert mode and confirm rename.
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'y', 0))

	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected renamed file at %q: %v", newPath, err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old file to be missing")
	}
}

// TestRun_SearchPrompt_Simulation verifies that the search prompt moves the cursor
// to the first match of the query when Enter is pressed.
func TestRun_SearchPrompt_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("hello world hello")
	r := &Runner{Screen: s, Buf: buf, History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// allow event loop to start
	time.Sleep(10 * time.Millisecond)

	// open search prompt via Ctrl+W
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModCtrl))
	// type query
	for _, ch := range "world" {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}
	// accept
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// quit
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	if r.Cursor != len([]rune("hello ")) {
		t.Fatalf("expected cursor at start of 'world', got %d", r.Cursor)
	}
}

func TestRun_MnemonicMenu_FromVisualSelection(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("alpha beta")
	r := &Runner{Screen: s, Buf: buf, History: history.New(), Mode: ModeVisual, VisualStart: 0}
	r.Cursor = 5

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	time.Sleep(10 * time.Millisecond)

	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}
}

// TestRun_MnemonicMenuMacroRecord_Simulation verifies starting a macro via the
// mnemonic menu shows the macro status indicator.
func TestRun_MnemonicMenuMacroRecord_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()
	s.SetSize(60, 10)

	r := &Runner{Screen: s, Buf: buffer.NewGapBuffer(0), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// allow event loop to start
	time.Sleep(10 * time.Millisecond)

	// open mnemonic menu: Space, then m r
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'm', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'r', 0))

	waitForMacroStatus(t, r, "Macro record: choose register")
	// select register
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'b', 0))

	statusLine := "Recording macro @b"
	waitForMacroStatus(t, r, statusLine)
	assertMacroStatusRow(t, s, statusLine)

	t.Logf("macro status snapshot: %q", r.MacroStatus)

	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}
}

func waitForMacroStatus(t *testing.T, r *Runner, want string) {
	t.Helper()
	deadline := time.After(500 * time.Millisecond)
	for {
		if r.MacroStatus == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for macro status, got %q", r.MacroStatus)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func assertMacroStatusRow(t *testing.T, s tcell.Screen, statusLine string) {
	t.Helper()
	deadline := time.After(500 * time.Millisecond)
	for {
		_, height := s.Size()
		match := true
		for i, r := range statusLine {
			cr, _, _, _ := s.GetContent(i, height-1)
			if cr != r {
				match = false
				break
			}
		}
		if match {
			return
		}
		select {
		case <-deadline:
			_, height := s.Size()
			cr, _, _, _ := s.GetContent(0, height-1)
			t.Fatalf("expected macro status %q at 0, got %q", string(statusLine[0]), string(cr))
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestRun_MacroRecordAndPlayback_Simulation records a simple macro and replays it.
func TestRun_MacroRecordAndPlayback_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()
	s.SetSize(60, 10)

	r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString("start\n"), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	time.Sleep(10 * time.Millisecond)

	// start recording into register a
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	// insert 'a' in insert mode and return to normal
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'a', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	// stop recording
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', 0))

	waitForMacroStatus(t, r, "")

	// open a new line below and replay the macro
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'o', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, '@', 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'a', 0))

	deadline := time.After(500 * time.Millisecond)
	for {
		if r.Buf.String() == "astart\na\n" {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for macro playback, got %q", r.Buf.String())
		case <-time.After(10 * time.Millisecond):
		}
	}

	s.PostEventWait(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0))
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'y', 0))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}
}

// TestRun_SearchPrompt_CtrlKey_Simulation verifies the search prompt also opens
// when the terminal emits the dedicated Ctrl+W key (as seen in real logs), and
// that it moves the cursor on Enter.
func TestRun_SearchPrompt_CtrlKey_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("hello world hello")
	r := &Runner{Screen: s, Buf: buf, History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	// allow event loop to start
	time.Sleep(10 * time.Millisecond)

	// open search prompt via dedicated Ctrl+W key
	s.PostEventWait(tcell.NewEventKey(tcell.KeyCtrlW, 0, 0))
	// type query
	for _, ch := range "world" {
		s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ch, 0))
	}
	// accept search
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// quit editor
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	expected := len([]rune("hello "))
	if r.Cursor != expected {
		t.Fatalf("expected cursor %d after search, got %d", expected, r.Cursor)
	}
	if got := r.Buf.String(); got != "hello world hello" {
		t.Fatalf("buffer modified during search: %q", got)
	}
}

// TestRun_GoToPrompt_Simulation verifies that the go-to prompt jumps to the specified line.
func TestRun_GoToPrompt_Simulation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()

	content := "one\ntwo\nthree\n"
	r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString(content), History: history.New()}

	done := make(chan error, 1)
	go func() { done <- r.Run() }()

	time.Sleep(10 * time.Millisecond)

	// open go-to prompt via Alt+G
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModAlt))
	// type line number
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, '3', 0))
	// accept
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	// quit editor
	s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for runner to quit")
	}

	expected := len([]rune("one\ntwo\n"))
	if r.Cursor != expected {
		t.Fatalf("expected cursor %d after go-to, got %d", expected, r.Cursor)
	}
}
