package buffer

import "unicode"

// IsWordRune reports whether r is considered part of a word.
// Words consist of letters, digits, or underscore characters.
func IsWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// WordStart returns the index of the beginning of the word that ends at or before pos.
// It behaves similar to Vim's 'b' motion.
func WordStart(g *GapBuffer, pos int) int {
	if g == nil || g.Len() == 0 {
		return 0
	}
	if pos > g.Len() {
		pos = g.Len()
	}
	if pos > 0 {
		pos--
	}
	for pos > 0 && !IsWordRune(g.RuneAt(pos)) {
		pos--
	}
	for pos > 0 && IsWordRune(g.RuneAt(pos-1)) {
		pos--
	}
	return pos
}

// WordEnd returns the index of the end of the word that begins at or after pos.
// It behaves similar to Vim's 'e' motion.
func WordEnd(g *GapBuffer, pos int) int {
	if g == nil || g.Len() == 0 {
		return 0
	}
	if pos >= g.Len() {
		return g.Len() - 1
	}
	if IsWordRune(g.RuneAt(pos)) && (pos == g.Len()-1 || !IsWordRune(g.RuneAt(pos+1))) {
		pos++
	}
	for pos < g.Len() && !IsWordRune(g.RuneAt(pos)) {
		pos++
	}
	for pos < g.Len() && IsWordRune(g.RuneAt(pos)) {
		pos++
	}
	if pos > 0 {
		pos--
	}
	return pos
}

// NextWordStart returns the index of the start of the next word after pos.
// It behaves similar to Vim's 'w' motion.
func NextWordStart(g *GapBuffer, pos int) int {
	if g == nil || g.Len() == 0 {
		return 0
	}
	if pos >= g.Len() {
		return g.Len()
	}
	for pos < g.Len() && IsWordRune(g.RuneAt(pos)) {
		pos++
	}
	for pos < g.Len() && !IsWordRune(g.RuneAt(pos)) {
		pos++
	}
	return pos
}
