package buffer

// TextStorage defines the minimal storage operations used by the editor.
// Positions and lengths are expressed in runes (not bytes).
type TextStorage interface {
	Insert(pos int, s []rune) error
	Delete(start, end int) error
	Slice(start, end int) []rune
	Len() int
	LineAt(idx int) (start, end int)
}
