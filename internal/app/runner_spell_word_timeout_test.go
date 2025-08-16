package app

import (
    "os"
    "testing"

    "example.com/texteditor/internal/testhelpers"
    "example.com/texteditor/pkg/buffer"
    "example.com/texteditor/pkg/history"
)

func TestCheckWordAtCursor_Timeout(t *testing.T) {
    bin := testhelpers.BuildBin(t, "slowchecker", "./internal/testhelpers/slowchecker")
    old := os.Getenv("TEXTEDITOR_SPELL")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL", old) })
    if err := os.Setenv("TEXTEDITOR_SPELL", bin); err != nil {
        t.Fatalf("setenv: %v", err)
    }
    oldT := os.Getenv("TEXTEDITOR_SPELL_TIMEOUT_MS")
    t.Cleanup(func(){ _ = os.Setenv("TEXTEDITOR_SPELL_TIMEOUT_MS", oldT) })
    if err := os.Setenv("TEXTEDITOR_SPELL_TIMEOUT_MS", "50"); err != nil {
        t.Fatalf("setenv timeout: %v", err)
    }
    // Also set delay to be long to ensure timeout is hit
    oldS := os.Getenv("SLOW_MS")
    t.Cleanup(func(){ _ = os.Setenv("SLOW_MS", oldS) })
    _ = os.Setenv("SLOW_MS", "2000")

    r := &Runner{Buf: buffer.NewGapBufferFromString("Hello mispelt ok\n"), History: history.New()}
    r.Cursor = len([]rune("Hello ")) // on mispelt

    got := r.CheckWordAtCursor()
    if got != "Spell check timed out" {
        t.Fatalf("expected timeout message, got %q", got)
    }
}

