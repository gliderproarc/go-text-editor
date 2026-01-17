package app

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func (r *Runner) killRingPreview(text string) string {
	if text == "" {
		return "(empty)"
	}
	flat := strings.ReplaceAll(text, "\n", "\\n")
	flat = strings.ReplaceAll(flat, "\t", "\\t")
	max := 60
	if len([]rune(flat)) > max {
		runes := []rune(flat)
		flat = string(runes[:max]) + "…"
	}
	return flat
}

func (r *Runner) killRingStatusLines() []string {
	if !r.KillRing.HasData() {
		return []string{"Kill ring is empty"}
	}
	lines := []string{
		"Kill ring (Esc/Ctrl+G or Enter to accept)",
		"Use Ctrl+N/P or arrows to cycle",
	}
	entries := r.KillRing.EntriesFromCurrent()
	width := 0
	height := 0
	if r.Screen != nil {
		width, height = r.Screen.Size()
	}
	maxEntries := len(entries)
	if height > 0 {
		maxEntries = height - len(lines) - 1
	}
	if maxEntries < 1 {
		maxEntries = 1
	}
	if maxEntries > len(entries) {
		maxEntries = len(entries)
	}
	for i := 0; i < maxEntries; i++ {
		prefix := "  "
		if i == 0 {
			prefix = "> "
		}
		entry := prefix + r.killRingPreview(entries[i])
		if width > 0 {
			runes := []rune(entry)
			if len(runes) > width {
				entry = string(runes[:width])
			}
		}
		lines = append(lines, entry)
	}
	if len(entries) > maxEntries {
		lines = append(lines, fmt.Sprintf("  … and %d more", len(entries)-maxEntries))
	}
	return lines
}

func (r *Runner) showKillRingStatus() {
	if r.Screen == nil {
		return
	}
	lines := r.killRingStatusLines()
	r.setMiniBuffer(lines)
	r.draw(nil)
}

func (r *Runner) yankPop(direction int) {
	if r.Buf == nil {
		return
	}
	rotate := r.KillRing.Rotate
	if direction < 0 {
		rotate = r.KillRing.RotatePrev
	}
	if !r.lastYankValid {
		if !rotate() {
			r.showDialog("Kill ring has no alternate entries")
			return
		}
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "yank.pop.select", "text": r.KillRing.Get(), "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
		}
		if r.Screen != nil {
			r.draw(nil)
		}
		r.showKillRingStatus()
		return
	}
	if !rotate() {
		r.showDialog("Kill ring has no alternate entries")
		return
	}
	start := r.lastYankStart
	end := r.lastYankEnd
	count := r.lastYankCount
	if count < 1 {
		count = 1
	}
	if start < 0 || end < start || end > r.Buf.Len() {
		r.clearYankState()
		r.showDialog("Unable to cycle yank")
		return
	}
	r.yankInProgress = true
	r.Cursor = start
	removed := string(r.Buf.Slice(start, end))
	_ = r.deleteRange(start, end, removed)
	text := r.KillRing.Get()
	for i := 0; i < count; i++ {
		r.insertText(text)
	}
	r.yankInProgress = false
	r.lastYankStart = start
	r.lastYankEnd = r.Cursor
	r.lastYankCount = count
	r.lastYankValid = true
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": "yank.pop", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.showKillRingStatus()
}

func (r *Runner) runKillRingCycle() {
	if !r.KillRing.HasData() {
		r.showDialog("Kill ring is empty")
		return
	}
	if r.Screen == nil {
		return
	}
	defer func() {
		r.clearMiniBuffer()
		r.draw(nil)
	}()
	r.yankPop(1)
	for {
		ev := r.waitEvent()
		if ev == nil {
			return
		}
		kev, ok := ev.(*tcell.EventKey)
		if !ok {
			continue
		}
		switch {
		case r.isCancelKey(kev) || kev.Key() == tcell.KeyEnter:
			return
		case kev.Key() == tcell.KeyCtrlN || kev.Key() == tcell.KeyDown || kev.Key() == tcell.KeyRight:
			r.yankPop(1)
		case kev.Key() == tcell.KeyCtrlP || kev.Key() == tcell.KeyUp || kev.Key() == tcell.KeyLeft:
			r.yankPop(-1)
		case kev.Key() == tcell.KeyRune && kev.Rune() == 'n' && kev.Modifiers() == tcell.ModCtrl:
			r.yankPop(1)
		case kev.Key() == tcell.KeyRune && kev.Rune() == 'p' && kev.Modifiers() == tcell.ModCtrl:
			r.yankPop(-1)
		}
	}
}
