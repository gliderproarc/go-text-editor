package app

import (
	"strings"
	"testing"
	"time"

	"example.com/texteditor/internal/testhelpers"
	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/history"
	"github.com/gdamore/tcell/v2"
)

// TestRunner_SpellHighlights_Viewport verifies that enabling spell check with a
// checker speaking the whitespace protocol produces spell highlight ranges in
// the rendered snapshot for visible misspelled words.
func TestRunner_SpellHighlights_Viewport(t *testing.T) {
	// Build simple test checker
	bin := testhelpers.BuildBin(t, "simplechecker", "./internal/testhelpers/simplechecker")

	// Simulation screen
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim: %v", err)
	}
	defer s.Fini()
	s.SetSize(80, 8)

	content := "Hello mispelt OK OOPS unknown\nsecond line\n"
	r := &Runner{Screen: s, Buf: buffer.NewGapBufferFromString(content), History: history.New()}
	r.RenderCh = make(chan renderState, 8)
	updates := make(chan renderState, 1)
	go func() {
		for st := range r.RenderCh {
			select {
			case updates <- st:
			default:
				<-updates
				updates <- st
			}
		}
	}()

	// Enable spell check using our helper binary
	if err := r.EnableSpellCheck(bin); err != nil {
		t.Fatalf("EnableSpellCheck: %v", err)
	}
	t.Cleanup(func() {
		if r.Spell != nil && r.Spell.Client != nil {
			r.Spell.Client.Stop()
		}
	})

	// Trigger a draw; updateSpellAsync runs during snapshot
	r.draw(nil)

	// Wait for a frame that contains spell highlights
	var st renderState
	deadline := time.After(6 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	found := false
	for !found {
		select {
		case st = <-updates:
			// Look for any bg.spell ranges
			for _, h := range st.highlights {
				if h.Group == "bg.spell" {
					found = true
					break
				}
			}

		case <-ticker.C:
			r.draw(nil)
		case <-deadline:
			t.Fatalf("timeout waiting for spell highlights")
		}
	}

	// Verify expected misspelled/unknown words are highlighted
	text := strings.ReplaceAll(strings.Join(st.lines, "\n")+"\n", "\r\n", "\n")
	want := map[string]bool{"mispelt": false, "oops": false, "unknown": false}
	gotCount := 0
	for _, h := range st.highlights {
		if h.Group != "bg.spell" {
			continue
		}
		if h.Start < 0 || h.End > len(text) || h.Start >= h.End {
			continue
		}
		w := strings.ToLower(text[h.Start:h.End])
		if _, ok := want[w]; ok {
			if !want[w] {
				gotCount++
			}
			want[w] = true
		}
	}
	if gotCount < 2 { // at least two of the three must be present in viewport
		t.Fatalf("expected at least two spell highlights, got %d (%v)", gotCount, want)
	}

	// Disable and ensure highlights clear on next draw
	r.DisableSpellCheck()
	r.draw(nil)
	select {
	case st = <-updates:
		for _, h := range st.highlights {
			if h.Group == "bg.spell" {
				t.Fatalf("expected no spell highlights after disable")
			}
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for post-disable frame")
	}
}
