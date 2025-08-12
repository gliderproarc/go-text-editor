package app

import (
	"strings"
	"testing"
	"time"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/history"
	"github.com/gdamore/tcell/v2"
)

// TestRunner_LargeFilePerformance simulates rendering and key handling on a
// large buffer and records the throughput to catch regressions with long files.
func TestRunner_LargeFilePerformance(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("initializing simulation screen failed: %v", err)
	}
	defer s.Fini()

	// Generate a large file: 2000 lines of sample text.
	var sb strings.Builder
	for i := 0; i < 2000; i++ {
		sb.WriteString("This is a line in a very large file used for performance testing.\n")
	}
	buf := buffer.NewGapBufferFromString(sb.String())

	r := &Runner{Screen: s, Buf: buf, History: history.New()}

	// Measure draw frame rate.
	const frameCount = 10
	start := time.Now()
	for i := 0; i < frameCount; i++ {
		r.draw(nil)
	}
	dur := time.Since(start)
	fps := float64(frameCount) / dur.Seconds()
	t.Logf("draw FPS on large file: %.2f", fps)
	if fps < 20 {
		t.Fatalf("draw FPS too low: %.2f", fps)
	}

	// Measure key handling rate (down arrow which triggers redraw).
	const events = 20
	start = time.Now()
	for i := 0; i < events; i++ {
		r.handleKeyEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	}
	dur = time.Since(start)
	eps := float64(events) / dur.Seconds()
	t.Logf("key events per second on large file: %.2f", eps)
	if eps < 5 {
		t.Fatalf("key handling rate too low: %.2f", eps)
	}
}
