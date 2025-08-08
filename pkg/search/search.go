package search

import "strings"

// Range represents a byte-offset half-open interval [Start, End)
// (for simplicity this package uses byte offsets; TextStorage may convert as needed).
type Range struct {
	Start int
	End   int
}

// SearchAll returns all non-overlapping occurrences of query in text as byte ranges.
// It performs a simple forward scan; empty query returns nil.
func SearchAll(text, query string) []Range {
	if query == "" {
		return nil
	}
	var res []Range
	off := 0
	for {
		idx := strings.Index(text[off:], query)
		if idx < 0 {
			break
		}
		start := off + idx
		end := start + len(query)
		res = append(res, Range{Start: start, End: end})
		off = end
	}
	return res
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
