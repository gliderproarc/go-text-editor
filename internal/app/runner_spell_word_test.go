package app

import (
    "os"
    "testing"

    "example.com/texteditor/internal/testhelpers"
    "example.com/texteditor/pkg/buffer"
    "example.com/texteditor/pkg/history"
    "github.com/gdamore/tcell/v2"
)

// helper to make a runner with a simulation screen
func newSimRunner(t *testing.T, content string) *Runner {
    t.Helper()
    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil { t.Fatalf("init sim: %v", err) }
    s.SetSize(80, 8)
    r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString(content), History: history.New()}
    // make render asynchronous channel small; not needed for return-value tests
    r.RenderCh = make(chan renderState, 2)
    t.Cleanup(func(){ s.Fini() })
    return r
}

func TestCheckWordAtCursor_NoWord(t *testing.T) {
    r := newSimRunner(t, "hello world\n")
    // place cursor on whitespace between words (position after "hello")
    r.Cursor = len([]rune("hello"))
    msg := r.CheckWordAtCursor()
    if msg != "No word found" {
        t.Fatalf("expected 'No word found', got %q", msg)
    }
}

func TestCheckWordAtCursor_OKAndMisspelled(t *testing.T) {
    bin := testhelpers.BuildBin(t, "simplechecker", "./internal/testhelpers/simplechecker")
    // ensure our check uses the helper
    old := os.Getenv("TEXTEDITOR_SPELL")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL", old) })
    if err := os.Setenv("TEXTEDITOR_SPELL", bin); err != nil {
        t.Fatalf("setenv: %v", err)
    }

    r := newSimRunner(t, "Hello mispelt ok\n")

    // Cursor on a correct word: Hello
    r.Cursor = 1 // within "Hello"
    got := r.CheckWordAtCursor()
    if got != "OK: hello" { // lowercased in CheckWordAtCursor
        t.Fatalf("expected OK message, got %q", got)
    }

    // Cursor on a misspelled word: mispelt
    // position at start of mispelt: len("Hello ") == 6
    r.Cursor = 6
    got = r.CheckWordAtCursor()
    if got != "Misspelled: mispelt" {
        t.Fatalf("expected misspelled message, got %q", got)
    }
}

