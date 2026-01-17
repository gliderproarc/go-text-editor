package app

func isTextObjectDelimiter(delim rune) bool {
	switch delim {
	case '"', '\'', '`', '(', ')', '[', ']', '{', '}':
		return true
	default:
		return false
	}
}

func textObjectPair(delim rune) (open, close rune, ok bool) {
	switch delim {
	case '"', '\'', '`':
		return delim, delim, true
	case '(', ')':
		return '(', ')', true
	case '[', ']':
		return '[', ']', true
	case '{', '}':
		return '{', '}', true
	default:
		return 0, 0, false
	}
}

func (r *Runner) textObjectBounds(delim rune, around bool) (start, end int, ok bool) {
	if r.Buf == nil || r.Buf.Len() == 0 {
		return 0, 0, false
	}
	open, close, ok := textObjectPair(delim)
	if !ok {
		return 0, 0, false
	}
	cursor := r.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= r.Buf.Len() {
		cursor = r.Buf.Len() - 1
	}
	leftSearch := cursor
	rightSearch := cursor
	if open == close {
		if r.Buf.RuneAt(cursor) == open {
			leftSearch = cursor - 1
			rightSearch = cursor
		}
	} else {
		if r.Buf.RuneAt(cursor) == open {
			rightSearch = cursor + 1
		} else if r.Buf.RuneAt(cursor) == close {
			leftSearch = cursor - 1
		}
	}
	left := -1
	for i := leftSearch; i >= 0; i-- {
		if r.Buf.RuneAt(i) == open {
			left = i
			break
		}
	}
	right := -1
	for i := rightSearch; i < r.Buf.Len(); i++ {
		if r.Buf.RuneAt(i) == close {
			right = i
			break
		}
	}
	if left == -1 || right == -1 || left >= right {
		return 0, 0, false
	}
	if around {
		start = left
		end = right + 1
	} else {
		start = left + 1
		end = right
	}
	if start < 0 || end > r.Buf.Len() || start >= end {
		return 0, 0, false
	}
	return start, end, true
}
