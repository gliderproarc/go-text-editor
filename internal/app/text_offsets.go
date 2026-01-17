package app

// runeIndexToByteOffset converts a rune index (count of runes from start) to a byte offset in s.
func runeIndexToByteOffset(s string, runeIndex int) int {
	runes := []rune(s)
	if runeIndex <= 0 {
		return 0
	}
	if runeIndex >= len(runes) {
		return len(s)
	}
	return len(string(runes[:runeIndex]))
}

// byteOffsetToRuneIndex converts a byte offset into s to the corresponding rune index.
func byteOffsetToRuneIndex(s string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(s) {
		return len([]rune(s))
	}
	return len([]rune(s[:byteOffset]))
}

func lineNumberForByte(text string, pos int) int {
	if pos <= 0 {
		return 1
	}
	if pos > len(text) {
		pos = len(text)
	}
	line := 1
	for i := 0; i < pos; i++ {
		if text[i] == '\n' {
			line++
		}
	}
	return line
}
