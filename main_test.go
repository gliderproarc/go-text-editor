package main

import (
    "testing"

    "github.com/gdamore/tcell/v2"
)

// TestDrawUI checks that drawUI does not panic and sets expected content.
func TestDrawUI(t *testing.T) {
    // Use a simulation screen to avoid /dev/tty dependencies in CI/sandbox.
    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil {
        t.Fatalf("initializing screen failed: %v", err)
    }
    defer s.Fini()
    // Ensure a reasonable non-zero size for assertions.
    s.SetSize(80, 24)

    width, height := s.Size()
	// Ensure screen has non-zero size
	if width <= 0 || height <= 0 {
		t.Fatalf("unexpected screen size: %d x %d", width, height)
	}

	drawUI(s)
	// The message should be centered
	msg := "TextEditor: No File"
	msgX := (width - len(msg)) / 2
	msgY := height / 2
	for i, r := range msg {
		cr, _, _, _ := s.GetContent(msgX+i, msgY)
		if cr != r {
			t.Fatalf("content mismatch at (%d,%d): expected %q got %q", msgX+i, msgY, string(r), string(cr))
		}
	}
}

// TestDrawHelp checks that drawHelp does not panic.
func TestDrawHelp(t *testing.T) {
    s := tcell.NewSimulationScreen("UTF-8")
    if err := s.Init(); err != nil {
        t.Fatalf("initializing screen failed: %v", err)
    }
    defer s.Fini()

	drawHelp(s)
	// No specific content check, just ensure no panic.
}
