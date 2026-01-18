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
	t.Setenv("TEXTEDITOR_SPELL_TIMEOUT_MS", "2000")
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

	// Enable spell check using our helper binary
	if err := r.EnableSpellCheck(bin); err != nil {
		t.Fatalf("EnableSpellCheck: %v", err)
	}
	t.Cleanup(func() {
		if r.Spell != nil && r.Spell.Client != nil {
			r.Spell.Client.Stop()
		}
	})

	deadline := time.After(6 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if r.Spell != nil && len(r.Spell.ranges) > 0 {
			break
		}
		select {
		case <-ticker.C:
			r.updateSpellAsync()
			if r.Spell == nil || r.Spell.Client == nil {
				t.Fatalf("spell client stopped while waiting for highlights")
			}
		case <-deadline:
			t.Fatalf("timeout waiting for spell highlights")
		}
	}

	st := r.renderSnapshot(nil)

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
	st = r.renderSnapshot(nil)
	for _, h := range st.highlights {
		if h.Group == "bg.spell" {
			t.Fatalf("expected no spell highlights after disable")
		}
	}
}

// TestRunner_SpellHighlights_EOFResponse ensures EOF-terminated responses are treated
// as valid results and still yield highlights.
func TestRunner_SpellHighlights_EOFResponse(t *testing.T) {
	t.Setenv("TEXTEDITOR_SPELL_TIMEOUT_MS", "2000")
	bin := testhelpers.BuildBin(t, "eofchecker", "./internal/testhelpers/eofchecker")

	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim: %v", err)
	}
	defer s.Fini()
	s.SetSize(80, 8)

	content := "Hello mispelt ok\n"
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

	if err := r.EnableSpellCheck(bin); err != nil {
		t.Fatalf("EnableSpellCheck: %v", err)
	}
	t.Cleanup(func() {
		if r.Spell != nil && r.Spell.Client != nil {
			r.Spell.Client.Stop()
		}
	})

	r.draw(nil)

	var st renderState
	deadline := time.After(6 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	found := false
	for !found {
		select {
		case st = <-updates:
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

	text := strings.ReplaceAll(strings.Join(st.lines, "\n")+"\n", "\r\n", "\n")
	seen := false
	for _, h := range st.highlights {
		if h.Group != "bg.spell" {
			continue
		}
		if h.Start < 0 || h.End > len(text) || h.Start >= h.End {
			continue
		}
		w := strings.ToLower(text[h.Start:h.End])
		if w == "mispelt" {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatalf("expected mispelt to be highlighted")
	}
}
