package app

import (
    "strings"
    "testing"

    "example.com/texteditor/pkg/buffer"
    "example.com/texteditor/pkg/config"
    "example.com/texteditor/pkg/search"
    "github.com/gdamore/tcell/v2"
)

func TestDrawFile_Highlights(t *testing.T) {
	// Use simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	text := "hello world\nhello"
	lines := strings.Split(text, "\n")
	// compute highlights for "hello"
	ranges := search.SearchAll(text, "hello")

    // draw with highlights
    drawFile(s, "f.txt", lines, ranges, -1, false, ModeInsert, 0, nil, config.DefaultTheme())

	// check first line "hello" at (0,0..4) is highlighted
	for x := 0; x < 5; x++ {
		cr, _, style, _ := s.GetContent(x, 0)
		if cr != rune("hello"[x]) {
			t.Fatalf("expected rune %q at (%d,0) got %q", rune("hello"[x]), x, cr)
		}
		expStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow)
		if style != expStyle {
			t.Fatalf("expected highlighted style at (%d,0) got %v", x, style)
		}
	}

	// check second line "hello" at (0,1..4)
	for x := 0; x < 5; x++ {
		cr, _, style, _ := s.GetContent(x, 1)
		if cr != rune("hello"[x]) {
			t.Fatalf("expected rune %q at (%d,1) got %q", rune("hello"[x]), x, cr)
		}
		expStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow)
		if style != expStyle {
			t.Fatalf("expected highlighted style at (%d,1) got %v", x, style)
		}
	}
}

func TestDrawBuffer_DirtyIndicator(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("hello")
    drawBuffer(s, buf, "f.txt", nil, 0, true, ModeInsert, 0, nil, config.DefaultTheme())

	_, height := s.Size()
	expected := "f.txt [+] — Press Ctrl+Q to exit"
	for i, r := range expected {
		cr, _, _, _ := s.GetContent(i, height-1)
		if cr != r {
			t.Fatalf("expected status %q, got mismatch at %d: %q", expected, i, string(cr))
		}
	}
}

func TestDrawBuffer_MiniBuffer(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	buf := buffer.NewGapBufferFromString("hello")
	mini := []string{"mini", "buffer"}
    drawBuffer(s, buf, "f.txt", nil, 0, false, ModeInsert, 0, mini, config.DefaultTheme())

	_, height := s.Size()
	for i, line := range mini {
		for x, r := range line {
			cr, _, _, _ := s.GetContent(x, height-1-len(mini)+i)
			if cr != r {
				t.Fatalf("expected %q at mini-buffer line %d pos %d, got %q", string(r), i, x, string(cr))
			}
		}
	}
}

// TestDrawFile_Viewport ensures that topLine offsets the rendered lines.
func TestDrawFile_Viewport(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	lines := []string{"l1", "l2", "l3"}
    drawFile(s, "f.txt", lines, nil, -1, false, ModeInsert, 1, nil, config.DefaultTheme())
	cr, _, _, _ := s.GetContent(0, 0)
	if cr != 'l' {
		t.Fatalf("expected 'l' at (0,0) got %q", string(cr))
	}
	cr, _, _, _ = s.GetContent(1, 0)
	if cr != '2' {
		t.Fatalf("expected '2' at (1,0) got %q", string(cr))
	}
	cr, _, _, _ = s.GetContent(0, 1)
	if cr != 'l' {
		t.Fatalf("expected 'l' at (0,1) got %q", string(cr))
	}
	cr, _, _, _ = s.GetContent(1, 1)
	if cr != '3' {
		t.Fatalf("expected '3' at (1,1) got %q", string(cr))
	}
}
