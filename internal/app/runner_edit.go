package app

import (
	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/search"
)

type repeatableChange struct {
	count int
	apply func(count int)
}

type insertCapture struct {
	count   int
	builder func(text string, count int)
	text    []rune
}

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

func (r *Runner) clearYankState() {
	r.lastYankValid = false
	r.lastYankStart = -1
	r.lastYankEnd = -1
	r.lastYankCount = 0
}

func (r *Runner) beginYankTracking() int {
	r.yankInProgress = true
	return r.Cursor
}

func (r *Runner) endYankTracking(start int, count int) {
	r.yankInProgress = false
	if count < 1 {
		count = 1
	}
	r.lastYankStart = start
	r.lastYankEnd = r.Cursor
	r.lastYankCount = count
	r.lastYankValid = start >= 0 && r.lastYankEnd >= start
}

// insertText inserts text at the current cursor, records history, and updates state.
func (r *Runner) insertText(text string) {
	if text == "" {
		return
	}
	if !r.yankInProgress {
		r.clearYankState()
	}
	pos := r.Cursor
	r.captureInsertText(text)
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
	r.handleMultiEditInsert(text, pos)
}

func (r *Runner) captureInsertText(text string) {
	if r.insertCapture == nil || text == "" {
		return
	}
	r.insertCapture.text = append(r.insertCapture.text, []rune(text)...)
}

func (r *Runner) beginInsertCapture(count int, builder func(text string, count int)) {
	if count == 0 {
		count = 1
	}
	r.insertCapture = &insertCapture{count: count, builder: builder}
}

func (r *Runner) finalizeInsertCapture() {
	if r.insertCapture == nil {
		return
	}
	text := string(r.insertCapture.text)
	if r.insertCapture.builder != nil {
		r.insertCapture.builder(text, r.insertCapture.count)
	}
	r.insertCapture = nil
}

func (r *Runner) consumeCount() int {
	if r.PendingCount == 0 {
		return 1
	}
	count := r.PendingCount
	r.PendingCount = 0
	return count
}

func (r *Runner) setLastChange(fn func(count int), count int) {
	if fn == nil {
		r.lastChange = nil
		return
	}
	if count < 1 {
		count = 1
	}
	r.lastChange = &repeatableChange{apply: fn, count: count}
}

func (r *Runner) repeatLastChange(count int) {
	if r.lastChange == nil {
		return
	}
	if count < 1 {
		count = r.lastChange.count
	}
	r.lastChange.apply(count)
}

func (r *Runner) deleteLines(count int) {
	if r.Buf == nil || count < 1 {
		return
	}
	start, _ := r.currentLineBounds()
	end := start
	for i := 0; i < count && end < r.Buf.Len(); i++ {
		for end < r.Buf.Len() && r.Buf.RuneAt(end) != '\n' {
			end++
		}
		if end < r.Buf.Len() {
			end++
		}
	}
	if end <= start {
		return
	}
	text := string(r.Buf.Slice(start, end))
	_ = r.deleteRange(start, end, text)
	r.KillRing.Push(text)
	r.recomputeCursorLine()
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": "delete.line", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.setLastChange(func(c int) { r.deleteLines(c) }, count)
}

func (r *Runner) deleteChars(count int) {
	if r.Buf == nil || count < 1 {
		return
	}
	if r.Cursor >= r.Buf.Len() {
		return
	}
	start := r.Cursor
	end := start + count
	if end > r.Buf.Len() {
		end = r.Buf.Len()
	}
	text := string(r.Buf.Slice(start, end))
	_ = r.deleteRange(start, end, text)
	r.KillRing.Push(text)
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": "cut.normal", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.setLastChange(func(c int) { r.deleteChars(c) }, count)
}

func (r *Runner) deleteWords(count int) string {
	if r.Buf == nil || count < 1 {
		return ""
	}
	start := r.Cursor
	end := start
	for i := 0; i < count; i++ {
		end = buffer.NextWordStart(r.Buf, end)
	}
	if end <= start {
		return ""
	}
	text := string(r.Buf.Slice(start, end))
	_ = r.deleteRange(start, end, text)
	r.KillRing.Push(text)
	r.recomputeCursorLine()
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": "delete.word", "text": text, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.setLastChange(func(c int) { r.deleteWords(c) }, count)
	return text
}

func (r *Runner) changeWords(count int, replacement string) {
	if count < 1 {
		count = 1
	}
	deleted := r.deleteWords(count)
	if replacement != "" {
		r.insertText(replacement)
	}
	if r.Logger != nil {
		r.Logger.Event("action", map[string]any{"name": "change.word", "deleted": deleted, "inserted": replacement, "count": count, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.setLastChange(func(c int) { r.changeWords(c, replacement) }, count)
}

func (r *Runner) yankTextObject(delim rune, around bool, count int) {
	if count < 1 {
		count = 1
	}
	r.clearYankState()
	for i := 0; i < count; i++ {
		start, end, ok := r.textObjectBounds(delim, around)
		if !ok {
			return
		}
		text := string(r.Buf.Slice(start, end))
		r.KillRing.Push(text)
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "yank.textobject", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
		}
	}
}

func (r *Runner) deleteTextObject(delim rune, around bool, count int) {
	if count < 1 {
		count = 1
	}
	for i := 0; i < count; i++ {
		start, end, ok := r.textObjectBounds(delim, around)
		if !ok {
			return
		}
		text := string(r.Buf.Slice(start, end))
		_ = r.deleteRange(start, end, text)
		r.KillRing.Push(text)
		r.recomputeCursorLine()
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "delete.textobject", "text": text, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
		}
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.setLastChange(func(c int) { r.deleteTextObject(delim, around, c) }, count)
}

func (r *Runner) changeTextObject(delim rune, around bool, count int, replacement string) {
	if count < 1 {
		count = 1
	}
	for i := 0; i < count; i++ {
		start, end, ok := r.textObjectBounds(delim, around)
		if !ok {
			return
		}
		deleted := string(r.Buf.Slice(start, end))
		_ = r.deleteRange(start, end, deleted)
		if replacement != "" {
			r.insertText(replacement)
		}
		if r.Logger != nil {
			r.Logger.Event("action", map[string]any{"name": "change.textobject", "deleted": deleted, "inserted": replacement, "cursor": r.Cursor, "buffer_len": r.Buf.Len()})
		}
	}
	if r.Screen != nil {
		r.draw(nil)
	}
	r.setLastChange(func(c int) { r.changeTextObject(delim, around, c, replacement) }, count)
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
	if !r.yankInProgress {
		r.clearYankState()
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
	r.handleMultiEditDelete(start, end, text)
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

func (r *Runner) replaceRange(start, end int, replacement string) {
	if r.Buf == nil {
		return
	}
	if start < 0 {
		start = 0
	}
	if end > r.Buf.Len() {
		end = r.Buf.Len()
	}
	if start >= end {
		return
	}
	deleted := string(r.Buf.Slice(start, end))
	if deleted == replacement {
		return
	}
	if !r.yankInProgress {
		r.clearYankState()
	}
	cursor := r.Cursor
	cursorLine := r.CursorLine
	replacementRunes := len([]rune(replacement))
	deletedRunes := end - start
	if start < cursor {
		if cursor >= end {
			cursor += replacementRunes - deletedRunes
		} else {
			cursor = start + replacementRunes
		}
		cursorLine += countNewlines(replacement) - countNewlines(deleted)
		if cursorLine < 0 {
			cursorLine = 0
		}
	}
	_ = r.Buf.Delete(start, end)
	if replacement != "" {
		_ = r.Buf.Insert(start, []rune(replacement))
	}
	if r.History != nil {
		if deleted != "" {
			r.History.RecordDelete(start, deleted)
		}
		if replacement != "" {
			r.History.RecordInsert(start, replacement)
		}
	}
	r.Cursor = cursor
	r.CursorLine = cursorLine
	r.Dirty = true
	r.syntaxSrc = ""
	r.editSeq++
}

func countNewlines(s string) int {
	count := 0
	for _, ch := range s {
		if ch == '\n' {
			count++
		}
	}
	return count
}
