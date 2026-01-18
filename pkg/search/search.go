package search

import "unicode"

// Range represents a byte-offset half-open interval [Start, End)
// (for simplicity this package uses byte offsets; TextStorage may convert as needed).
type Range struct {
	Start int
	End   int
	// Group is an optional category for styling (e.g. "keyword", "string").
	// Empty means generic highlight (used for search/selection background).
	Group string
}

// SearchAll returns all non-overlapping occurrences of query in text as byte ranges.
// It performs a simple forward scan; empty query returns nil.
func SearchAll(text, query string) []Range {
	return SearchAllCase(text, query, true)
}

// SearchAllCase returns all non-overlapping occurrences of query in text as byte ranges.
// When caseSensitive is false, the search is performed case-insensitively.
func SearchAllCase(text, query string, caseSensitive bool) []Range {
	if query == "" {
		return nil
	}
	queryRunes := []rune(query)
	var textRunes []rune
	if caseSensitive {
		textRunes = []rune(text)
	} else {
		textRunes = foldRunes([]rune(text))
		queryRunes = foldRunes(queryRunes)
	}
	queryLen := len(queryRunes)
	if queryLen == 0 {
		return nil
	}

	prefixBytes := make([]int, len(textRunes)+1)
	byteOffset := 0
	for i, r := range textRunes {
		prefixBytes[i] = byteOffset
		byteOffset += len(string(r))
	}
	prefixBytes[len(textRunes)] = byteOffset

	var res []Range
	for i := 0; i+queryLen <= len(textRunes); {
		match := true
		for j := 0; j < queryLen; j++ {
			if textRunes[i+j] != queryRunes[j] {
				match = false
				break
			}
		}
		if match {
			start := prefixBytes[i]
			end := prefixBytes[i+queryLen]
			res = append(res, Range{Start: start, End: end})
			i += queryLen
			continue
		}
		i++
	}
	return res
}

func foldRunes(in []rune) []rune {
	out := make([]rune, len(in))
	for i, r := range in {
		out[i] = unicode.ToLower(r)
	}
	return out
}

// SearchNext returns the index in ranges of the next match at or after pos.
// If pos is past all matches, it wraps and returns 0. Returns -1 if no ranges.
func SearchNext(ranges []Range, pos int) int {
	if len(ranges) == 0 {
		return -1
	}
	for i, r := range ranges {
		if pos < r.End {
			return i
		}
	}
	// wrap
	return 0
}

// HighlightRanges is an identity helper for now; kept for API completeness.
func HighlightRanges(ranges []Range) []Range {
	return ranges
}
