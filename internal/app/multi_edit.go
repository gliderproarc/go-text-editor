package app

import (
	"fmt"

	"example.com/texteditor/pkg/search"
)

type multiEditState struct {
	target       string
	primaryStart int
	primaryEnd   int
	applying     bool
	matches      []search.Range
}

func (r *Runner) multiEditPrimaryIndex(text string) int {
	st := r.MultiEdit
	if st == nil || len(st.matches) == 0 {
		return -1
	}
	primaryStart := runeIndexToByteOffset(text, st.primaryStart)
	primaryEnd := runeIndexToByteOffset(text, st.primaryEnd)
	return searchPrimaryIndex(st.matches, primaryStart, primaryEnd)
}

func reorderMatchesForPrimary(matches []search.Range, primaryStart, primaryEnd int) []search.Range {
	if len(matches) == 0 {
		return matches
	}
	primaryIndex := searchPrimaryIndex(matches, primaryStart, primaryEnd)
	if primaryIndex > 0 {
		matches[0], matches[primaryIndex] = matches[primaryIndex], matches[0]
	}
	return matches
}

func (r *Runner) adjustMultiEditMatchesForInsert(pos int, inserted string) {
	st := r.MultiEdit
	if st == nil || r.Buf == nil || inserted == "" {
		return
	}
	text := r.Buf.String()
	posByte := runeIndexToByteOffset(text, pos)
	insertBytes := len([]byte(inserted))
	for i, m := range st.matches {
		switch {
		case posByte <= m.Start:
			m.Start += insertBytes
			m.End += insertBytes
		case posByte < m.End:
			m.End += insertBytes
		}
		st.matches[i] = m
	}
}

func (r *Runner) adjustMultiEditMatchesForDelete(start int, deleted string) {
	st := r.MultiEdit
	if st == nil || r.Buf == nil || deleted == "" {
		return
	}
	text := r.Buf.String()
	startByte := runeIndexToByteOffset(text, start)
	deletedBytes := len([]byte(deleted))
	endByte := startByte + deletedBytes
	for i, m := range st.matches {
		newStart := offsetAfterDelete(m.Start, startByte, endByte)
		newEnd := offsetAfterDelete(m.End, startByte, endByte)
		if newEnd < newStart {
			newEnd = newStart
		}
		m.Start = newStart
		m.End = newEnd
		st.matches[i] = m
	}
}

func offsetAfterDelete(pos, start, end int) int {
	if pos < start {
		return pos
	}
	if pos >= end {
		return pos - (end - start)
	}
	return start
}

func searchPrimaryIndex(matches []search.Range, primaryStart, primaryEnd int) int {
	if len(matches) == 0 {
		return -1
	}
	for i, m := range matches {
		if m.Start <= primaryStart && m.End >= primaryEnd {
			return i
		}
	}
	best := -1
	bestDist := -1
	primaryMid := (primaryStart + primaryEnd) / 2
	for i, m := range matches {
		mid := (m.Start + m.End) / 2
		dist := mid - primaryMid
		if dist < 0 {
			dist = -dist
		}
		if best == -1 || dist < bestDist {
			best = i
			bestDist = dist
		}
	}
	return best
}

func (r *Runner) toggleMultiEdit() {
	if r.Mode == ModeMultiEdit {
		r.exitMultiEdit()
		return
	}
	r.startMultiEdit()
}

func (r *Runner) startMultiEdit() {
	if r.Buf == nil {
		return
	}
	if r.Mode == ModeInsert {
		r.finalizeInsertCapture()
	}
	if r.Mode == ModeVisual && r.VisualStart >= 0 {
		start, end := r.visualSelectionBounds()
		if end <= start {
			r.showDialog("Multi-edit needs a selection")
			return
		}
		text := string(r.Buf.Slice(start, end))
		r.enterMultiEdit(text, start, end)
		return
	}
	query := r.runMultiEditPrompt()
	if query == "" {
		return
	}
	r.enterMultiEditFromQuery(query)
}

func (r *Runner) enterMultiEditFromQuery(query string) {
	if r.Buf == nil || query == "" {
		return
	}
	text := r.Buf.String()
	matches := search.SearchAll(text, query)
	if len(matches) == 0 {
		r.showDialog("No matches for: " + query)
		return
	}
	cursorByte := runeIndexToByteOffset(text, r.Cursor)
	primaryIdx := 0
	for i, m := range matches {
		if cursorByte >= m.Start && cursorByte < m.End {
			primaryIdx = i
			break
		}
	}
	startRune := byteOffsetToRuneIndex(text, matches[primaryIdx].Start)
	endRune := byteOffsetToRuneIndex(text, matches[primaryIdx].End)
	if cursorByte < matches[primaryIdx].Start || cursorByte >= matches[primaryIdx].End {
		r.Cursor = startRune
		r.recomputeCursorLine()
	}
	r.enterMultiEdit(query, startRune, endRune)
}

func (r *Runner) enterMultiEdit(text string, start, end int) {
	if r.Buf == nil || text == "" {
		return
	}
	if start < 0 {
		start = 0
	}
	if end > r.Buf.Len() {
		end = r.Buf.Len()
	}
	if end < start {
		end = start
	}
	r.PendingG = false
	r.PendingD = false
	r.PendingY = false
	r.PendingC = false
	r.PendingTextObject = false
	r.TextObjectAround = false
	r.PendingCount = 0
	r.VisualStart = -1
	r.VisualLine = false
	r.Mode = ModeMultiEdit
	bufText := r.Buf.String()
	matches := search.SearchAll(bufText, text)
	if len(matches) == 0 {
		r.exitMultiEdit()
		r.showDialog("No matches for: " + text)
		return
	}
	primaryStart := runeIndexToByteOffset(bufText, start)
	primaryEnd := runeIndexToByteOffset(bufText, end)
	matches = reorderMatchesForPrimary(matches, primaryStart, primaryEnd)
	r.MultiEdit = &multiEditState{
		target:       text,
		primaryStart: start,
		primaryEnd:   end,
		matches:      matches,
	}
	if r.Screen != nil {
		r.draw(nil)
	}
}

func (r *Runner) exitMultiEdit() {
	r.Mode = ModeNormal
	r.MultiEdit = nil
	r.clearMiniBuffer()
	if r.Screen != nil {
		r.draw(nil)
	}
}

func (r *Runner) refreshMultiEditState() bool {
	st := r.MultiEdit
	if st == nil || r.Buf == nil {
		return false
	}
	text := r.Buf.String()
	matches := search.SearchAll(text, st.target)
	if len(matches) == 0 {
		return false
	}
	primaryStart := runeIndexToByteOffset(text, st.primaryStart)
	primaryEnd := runeIndexToByteOffset(text, st.primaryEnd)
	st.matches = reorderMatchesForPrimary(matches, primaryStart, primaryEnd)
	r.setMiniBuffer(r.multiEditStatusLines(text))
	return true
}

func (r *Runner) updateMultiEditStatus() {
	st := r.MultiEdit
	if st == nil || r.Buf == nil {
		return
	}
	r.setMiniBuffer(r.multiEditStatusLines(r.Buf.String()))
}

func (r *Runner) multiEditHighlights() []search.Range {
	st := r.MultiEdit
	if st == nil || r.Mode != ModeMultiEdit || r.Buf == nil {
		return nil
	}
	if len(st.matches) == 0 {
		return nil
	}
	out := make([]search.Range, 0, len(st.matches))
	for i, m := range st.matches {
		group := "bg.multiedit"
		if i == 0 {
			group = "bg.multiedit.current"
		}
		out = append(out, search.Range{Start: m.Start, End: m.End, Group: group})
	}
	return out
}

func (r *Runner) multiEditStatusLines(text string) []string {
	st := r.MultiEdit
	if st == nil {
		return nil
	}
	lines := []string{
		"Multi-edit: " + st.target,
		fmt.Sprintf("%d matches (Esc/Ctrl+G to exit)", len(st.matches)),
	}
	width := 0
	height := 0
	if r.Screen != nil {
		width, height = r.Screen.Size()
	}
	maxEntries := len(st.matches)
	if height > 0 {
		maxEntries = height - len(lines) - 1
	}
	if maxEntries < 1 {
		maxEntries = 1
	}
	if maxEntries > len(st.matches) {
		maxEntries = len(st.matches)
	}
	for i := 0; i < maxEntries; i++ {
		m := st.matches[i]
		if m.Start < 0 {
			m.Start = 0
		}
		if m.End > len(text) {
			m.End = len(text)
		}
		if m.End < m.Start {
			m.End = m.Start
		}
		prefix := "  "
		if i == 0 {
			prefix = "> "
		}
		lineNum := lineNumberForByte(text, m.Start)
		entry := fmt.Sprintf("%s%d: %q", prefix, lineNum, text[m.Start:m.End])
		if width > 0 {
			runes := []rune(entry)
			if len(runes) > width {
				entry = string(runes[:width])
			}
		}
		lines = append(lines, entry)
	}
	if len(st.matches) > maxEntries {
		lines = append(lines, fmt.Sprintf("  … and %d more", len(st.matches)-maxEntries))
	}
	return lines
}

func (r *Runner) multiEditPrimaryText() string {
	st := r.MultiEdit
	if st == nil || r.Buf == nil {
		return ""
	}
	start := st.primaryStart
	end := st.primaryEnd
	if start < 0 {
		start = 0
	}
	if end > r.Buf.Len() {
		end = r.Buf.Len()
	}
	if end < start {
		end = start
	}
	return string(r.Buf.Slice(start, end))
}

func (r *Runner) handleMultiEditInsert(inserted string, pos int) {
	st := r.MultiEdit
	if st == nil || st.applying || r.Mode != ModeMultiEdit || r.Buf == nil {
		return
	}
	if inserted == "" {
		return
	}
	oldTarget := st.target
	insertLen := len([]rune(inserted))
	if pos < st.primaryStart {
		st.primaryStart += insertLen
		st.primaryEnd += insertLen
	} else if pos <= st.primaryEnd {
		st.primaryEnd += insertLen
	}
	newTarget := r.multiEditPrimaryText()
	if newTarget == oldTarget {
		r.adjustMultiEditMatchesForInsert(pos, inserted)
		r.updateMultiEditStatus()
		return
	}
	if newTarget == "" {
		r.exitMultiEdit()
		return
	}
	st.target = newTarget
	r.applyMultiEditReplace(oldTarget, newTarget)
}

func (r *Runner) handleMultiEditDelete(start, end int, deleted string) {
	st := r.MultiEdit
	if st == nil || st.applying || r.Mode != ModeMultiEdit || r.Buf == nil {
		return
	}
	if start >= end {
		return
	}
	oldTarget := st.target
	deletedLen := end - start
	if end <= st.primaryStart {
		st.primaryStart -= deletedLen
		st.primaryEnd -= deletedLen
	} else if start < st.primaryEnd && end > st.primaryStart {
		if start < st.primaryStart {
			st.primaryStart = start
		}
		overlapStart := start
		if overlapStart < st.primaryStart {
			overlapStart = st.primaryStart
		}
		overlapEnd := end
		if overlapEnd > st.primaryEnd {
			overlapEnd = st.primaryEnd
		}
		st.primaryEnd -= overlapEnd - overlapStart
		if st.primaryEnd < st.primaryStart {
			st.primaryEnd = st.primaryStart
		}
	}
	newTarget := r.multiEditPrimaryText()
	if newTarget == oldTarget {
		r.adjustMultiEditMatchesForDelete(start, deleted)
		r.updateMultiEditStatus()
		return
	}
	if newTarget == "" {
		r.exitMultiEdit()
		return
	}
	st.target = newTarget
	r.applyMultiEditReplace(oldTarget, newTarget)
}

func (r *Runner) applyMultiEditReplace(oldTarget, newTarget string) {
	st := r.MultiEdit
	if st == nil || r.Buf == nil || oldTarget == newTarget {
		return
	}
	text := r.Buf.String()
	primaryStart := runeIndexToByteOffset(text, st.primaryStart)
	primaryEnd := runeIndexToByteOffset(text, st.primaryEnd)
	matches := search.SearchAll(text, oldTarget)
	st.applying = true
	defer func() { st.applying = false }()
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		if m.Start < primaryEnd && m.End > primaryStart {
			continue
		}
		startRune := byteOffsetToRuneIndex(text, m.Start)
		endRune := byteOffsetToRuneIndex(text, m.End)
		r.replaceRange(startRune, endRune, newTarget)
	}
	updatedText := r.Buf.String()
	updatedMatches := search.SearchAll(updatedText, newTarget)
	if len(updatedMatches) == 0 {
		r.exitMultiEdit()
		return
	}
	primaryStart = runeIndexToByteOffset(updatedText, st.primaryStart)
	primaryEnd = runeIndexToByteOffset(updatedText, st.primaryEnd)
	primaryIndex := searchPrimaryIndex(updatedMatches, primaryStart, primaryEnd)
	if primaryIndex < 0 {
		primaryStart = runeIndexToByteOffset(updatedText, r.Cursor)
		primaryEnd = primaryStart + len([]byte(newTarget))
		primaryIndex = searchPrimaryIndex(updatedMatches, primaryStart, primaryEnd)
	}
	if primaryIndex < 0 {
		primaryIndex = 0
	}
	if primaryIndex > 0 {
		updatedMatches[0], updatedMatches[primaryIndex] = updatedMatches[primaryIndex], updatedMatches[0]
	}
	st.target = newTarget
	st.matches = updatedMatches
	r.updateMultiEditStatus()
}
