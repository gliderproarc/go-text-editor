package app

import "example.com/texteditor/pkg/search"

// visualHighlightRange returns the current visual selection as byte offsets.
func (r *Runner) visualHighlightRange() []search.Range {
	if r.Mode != ModeVisual || r.VisualStart < 0 || r.Buf == nil {
		return nil
	}
	start := r.VisualStart
	end := r.Cursor
	if start > end {
		start, end = end, start
	}
	text := r.Buf.String()
	runes := []rune(text)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	startBytes := len(string(runes[:start]))
	endBytes := len(string(runes[:end]))
	return []search.Range{{Start: startBytes, End: endBytes}}
}

// insertText inserts text at the current cursor, records history, and updates state.
func (r *Runner) insertText(text string) {
	if text == "" {
		return
	}
	_ = r.Buf.Insert(r.Cursor, []rune(text))
	if r.History != nil {
		r.History.RecordInsert(r.Cursor, text)
	}
	r.Cursor += len([]rune(text))
	r.Dirty = true
}

// deleteRange deletes [start,end) with provided text for history and updates cursor.
func (r *Runner) deleteRange(start, end int, text string) error {
	if start < 0 {
		start = 0
	}
	if end > r.Buf.Len() {
		end = r.Buf.Len()
	}
	if start >= end {
		return nil
	}
	if err := r.Buf.Delete(start, end); err != nil {
		return err
	}
	if r.History != nil {
		r.History.RecordDelete(start, text)
	}
	// adjust cursor
	if r.Cursor > end {
		r.Cursor -= (end - start)
	} else if r.Cursor > start {
		r.Cursor = start
	}
	r.Dirty = true
	return nil
}

// moveCursorVertical moves the cursor up or down by delta lines, preserving the column when possible.
func (r *Runner) moveCursorVertical(delta int) {
	if r.Buf == nil || r.Buf.Len() == 0 {
		return
	}
	line := 0
	lineStart := 0
	for i := 0; i < r.Cursor && i < r.Buf.Len(); i++ {
		if string(r.Buf.Slice(i, i+1)) == "\n" {
			line++
			lineStart = i + 1
		}
	}
	col := r.Cursor - lineStart
	start, end := r.Buf.LineAt(line + delta)
	lineLen := end - start
	if lineLen > 0 && string(r.Buf.Slice(end-1, end)) == "\n" {
		lineLen--
	}
	if col > lineLen {
		col = lineLen
	}
	r.Cursor = start + col
	if r.Screen != nil {
		r.ensureCursorVisible()
	}
}

// currentLineBounds returns the rune start and end indices for the current cursor's line.
func (r *Runner) currentLineBounds() (start, end int) {
	// compute line index by counting newlines up to cursor
	line := 0
	for i := 0; i < r.Cursor && i < r.Buf.Len(); i++ {
		if string(r.Buf.Slice(i, i+1)) == "\n" {
			line++
		}
	}
	return r.Buf.LineAt(line)
}
