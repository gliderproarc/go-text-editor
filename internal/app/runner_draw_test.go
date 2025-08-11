package app

import (
	"strings"
	"testing"

	"example.com/texteditor/pkg/buffer"
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
	drawFile(s, "f.txt", lines, ranges, -1, false, ModeInsert, nil)

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
	drawBuffer(s, buf, "f.txt", nil, 0, true, ModeInsert, nil)

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
	drawBuffer(s, buf, "f.txt", nil, 0, false, ModeInsert, mini)

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
