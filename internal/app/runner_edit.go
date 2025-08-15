package app

import "example.com/texteditor/pkg/search"

// visualHighlightRange returns the current visual selection as byte offsets.
func (r *Runner) visualSelectionBounds() (start, end int) {
	start = r.VisualStart
	end = r.Cursor
	if start > end {
		start, end = end, start
	}
	if r.Buf != nil {
		if r.VisualLine {
			for start > 0 && r.Buf.RuneAt(start-1) != '\n' {
				start--
			}
			for end < r.Buf.Len() && r.Buf.RuneAt(end) != '\n' {
				end++
			}
			if end < r.Buf.Len() {
				end++
			}
		} else if end < r.Buf.Len() {
			// Ensure the character under the cursor is included
			end++
		}
	}
	return
}

func (r *Runner) visualHighlightRange() []search.Range {
	if r.Mode != ModeVisual || r.VisualStart < 0 || r.Buf == nil {
		return nil
	}
	start, end := r.visualSelectionBounds()
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
    // Tag as visual selection so renderer can style it subtly
    return []search.Range{{Start: startBytes, End: endBytes, Group: "bg.select"}}
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
	// Update line index based on inserted newlines
	for _, ch := range text {
		if ch == '\n' {
			r.CursorLine++
		}
	}
    r.Dirty = true
    r.syntaxSrc = ""
    // Mark buffer content changed for spell re-check coalescing
    r.editSeq++
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
	// remember original cursor to decide line adjustments
	origCursor := r.Cursor
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
	// adjust current line by number of removed newlines that were before or at cursor position
	if text != "" && start < origCursor { // only adjust if deletion was before the cursor
		removed := 0
		for _, ch := range text {
			if ch == '\n' {
				removed++
			}
		}
		r.CursorLine -= removed
		if r.CursorLine < 0 {
			r.CursorLine = 0
		}
	}
    r.Dirty = true
    r.syntaxSrc = ""
    // Mark buffer content changed for spell re-check coalescing
    r.editSeq++
    return nil
}

// moveCursorVertical moves the cursor up or down by delta lines, preserving the column when possible.
func (r *Runner) moveCursorVertical(delta int) {
	if r.Buf == nil || r.Buf.Len() == 0 || delta == 0 {
		return
	}
	// find start of current line and column by scanning backwards
	start := r.Cursor
	for start > 0 && r.Buf.RuneAt(start-1) != '\n' {
		start--
	}
	col := r.Cursor - start
	pos := start
	if delta > 0 {
		for i := 0; i < delta && pos < r.Buf.Len(); i++ {
			for pos < r.Buf.Len() && r.Buf.RuneAt(pos) != '\n' {
				pos++
			}
			if pos < r.Buf.Len() {
				pos++
			}
		}
	} else {
		for i := 0; i > delta && pos > 0; i-- {
			pos--
			for pos > 0 && r.Buf.RuneAt(pos-1) != '\n' {
				pos--
			}
		}
	}
	end := pos
	for end < r.Buf.Len() && r.Buf.RuneAt(end) != '\n' {
		end++
	}
	lineLen := end - pos
	if col > lineLen {
		col = lineLen
	}
	r.Cursor = pos + col
	// Update line index conservatively using cached line count
	if r.Buf != nil {
		total := len(r.Buf.Lines())
		if total < 1 {
			total = 1
		}
		r.CursorLine += delta
		if r.CursorLine < 0 {
			r.CursorLine = 0
		}
		if r.CursorLine > total-1 {
			r.CursorLine = total - 1
		}
	}
	if r.Screen != nil {
		r.ensureCursorVisible()
	}
}

// currentLineBounds returns the rune start and end indices for the current cursor's line.
func (r *Runner) currentLineBounds() (start, end int) {
	if r.Buf == nil || r.Buf.Len() == 0 {
		return 0, 0
	}
	start = r.Cursor
	for start > 0 && r.Buf.RuneAt(start-1) != '\n' {
		start--
	}
	end = r.Cursor
	for end < r.Buf.Len() && r.Buf.RuneAt(end) != '\n' {
		end++
	}
	if end < r.Buf.Len() {
		end++
	}
	return start, end
}
