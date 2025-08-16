package app

import (
    "os"
    "testing"
    "time"

    "example.com/texteditor/internal/testhelpers"
    "example.com/texteditor/pkg/buffer"
    "example.com/texteditor/pkg/history"
    "github.com/gdamore/tcell/v2"
)

// TestRun_Mnemonic_SpellCheckWord_OK_Simulation simulates pressing "SPC p c"
// with the cursor on a correct word and asserts the runner remains responsive
// (i.e., no freeze/deadlock in the mnemonic action path).
func TestRun_Mnemonic_SpellCheckWord_OK_Simulation(t *testing.T) {
    // Use the simple test checker to avoid external dependencies
    bin := testhelpers.BuildBin(t, "simplechecker", "./internal/testhelpers/simplechecker")
    old := os.Getenv("TEXTEDITOR_SPELL")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL", old) })
    if err := os.Setenv("TEXTEDITOR_SPELL", bin); err != nil {
        t.Fatalf("setenv: %v", err)
    }

    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil { t.Fatalf("init sim: %v", err) }
    defer s.Fini()

    // Buffer contains a correct word at cursor position
    r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString("Hello mispelt ok\n"), History: history.New()}
    r.Cursor = 1 // within "Hello"

    done := make(chan error, 1)
    go func(){ done <- r.Run() }()

    time.Sleep(10 * time.Millisecond) // allow event loop to start

    // Open mnemonic menu, navigate to "spell" (p), then "check word" (c)
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
    // Quit to end the loop; if the mnemonic action deadlocks, this won’t be processed
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

    select {
    case err := <-done:
        if err != nil { t.Fatalf("runner returned error: %v", err) }
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for runner to quit after SPC p c (possible freeze)")
    }
}

// TestRun_Mnemonic_SpellCheckWord_Misspelled_Simulation simulates pressing
// "SPC p c" with the cursor on a misspelled word. It should also remain
// responsive (no freeze) even when the checker reports a misspelling.
func TestRun_Mnemonic_SpellCheckWord_Misspelled_Simulation(t *testing.T) {
    bin := testhelpers.BuildBin(t, "simplechecker", "./internal/testhelpers/simplechecker")
    old := os.Getenv("TEXTEDITOR_SPELL")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL", old) })
    if err := os.Setenv("TEXTEDITOR_SPELL", bin); err != nil {
        t.Fatalf("setenv: %v", err)
    }

    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil { t.Fatalf("init sim: %v", err) }
    defer s.Fini()

    // Cursor at the start of the misspelled word "mispelt"
    r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString("Hello mispelt ok\n"), History: history.New()}
    r.Cursor = len([]rune("Hello ")) // position at start of mispelt

    done := make(chan error, 1)
    go func(){ done <- r.Run() }()

    time.Sleep(10 * time.Millisecond)

    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

    select {
    case err := <-done:
        if err != nil { t.Fatalf("runner returned error: %v", err) }
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for runner to quit after SPC p c (possible freeze)")
    }
}

// TestRun_Mnemonic_SpellCheckWord_AspellBridge_Simulation routes through the
// aspell bridge using a fake aspell binary on PATH to ensure the menu path and
// bridge interaction remain responsive.
func TestRun_Mnemonic_SpellCheckWord_AspellBridge_Simulation(t *testing.T) {
    // Build fake aspell and place it on PATH as "aspell"
    asp := testhelpers.BuildBin(t, "aspellfake", "./internal/testhelpers/aspellfake")
    dir, err := os.MkdirTemp("", "aspfakedir-")
    if err != nil { t.Fatalf("mktemp: %v", err) }
    t.Cleanup(func(){ _ = os.RemoveAll(dir) })
    data, err := os.ReadFile(asp)
    if err != nil { t.Fatalf("read aspellfake: %v", err) }
    aspellPath := dir + string(os.PathSeparator) + "aspell"
    if err := os.WriteFile(aspellPath, data, 0o755); err != nil { t.Fatalf("write aspell: %v", err) }

    oldPath := os.Getenv("PATH")
    t.Cleanup(func(){ _ = os.Setenv("PATH", oldPath) })
    if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath); err != nil {
        t.Fatalf("set PATH: %v", err)
    }

    // Point the editor to the bridge which invokes our fake aspell
    old := os.Getenv("TEXTEDITOR_SPELL")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL", old) })
    if err := os.Setenv("TEXTEDITOR_SPELL", "./aspellbridge"); err != nil {
        t.Fatalf("setenv: %v", err)
    }

    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil { t.Fatalf("init sim: %v", err) }
    defer s.Fini()

    r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString("Hello mispelt ok\n"), History: history.New()}
    r.Cursor = len([]rune("Hello ")) // on mispelt

    done := make(chan error, 1)
    go func(){ done <- r.Run() }()

    time.Sleep(10 * time.Millisecond)

    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

    select {
    case err := <-done:
        if err != nil { t.Fatalf("runner returned error: %v", err) }
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for runner to quit after SPC p c (possible freeze)")
    }
}

// TestRun_Mnemonic_SpellCheckWord_Timeout_Simulation ensures that a very slow
// checker does not freeze the UI; quitting still works after invoking SPC p c.
func TestRun_Mnemonic_SpellCheckWord_Timeout_Simulation(t *testing.T) {
    slow := testhelpers.BuildBin(t, "slowchecker", "./internal/testhelpers/slowchecker")
    old := os.Getenv("TEXTEDITOR_SPELL")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL", old) })
    if err := os.Setenv("TEXTEDITOR_SPELL", slow); err != nil {
        t.Fatalf("setenv: %v", err)
    }
    // Short timeout, long delay
    oldT := os.Getenv("TEXTEDITOR_SPELL_TIMEOUT_MS")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL_TIMEOUT_MS", oldT) })
    _ = os.Setenv("TEXTEDITOR_SPELL_TIMEOUT_MS", "50")
    oldS := os.Getenv("SLOW_MS")
    t.Cleanup(func(){ _ = os.Setenv("SLOW_MS", oldS) })
    _ = os.Setenv("SLOW_MS", "2000")

    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil { t.Fatalf("init sim: %v", err) }
    defer s.Fini()

    r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString("Hello mispelt ok\n"), History: history.New()}
    r.Cursor = len([]rune("Hello ")) // on mispelt

    done := make(chan error, 1)
    go func(){ done <- r.Run() }()

    time.Sleep(10 * time.Millisecond)

    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
    s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl))

    select {
    case err := <-done:
        if err != nil { t.Fatalf("runner returned error: %v", err) }
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for runner to quit after SPC p c with slow checker")
    }
}
