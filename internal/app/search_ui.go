package app

import (
	"fmt"

	"example.com/texteditor/pkg/search"
)

func buildSearchHighlights(raw []search.Range, current int) []search.Range {
	ranges := make([]search.Range, 0, len(raw))
	for i, rge := range raw {
		if i == current {
			ranges = append(ranges, search.Range{Start: rge.Start, End: rge.End, Group: "bg.search.current"})
		} else {
			ranges = append(ranges, search.Range{Start: rge.Start, End: rge.End, Group: "bg.search"})
		}
	}
	return ranges
}

func buildSearchPromptLine(text string, match search.Range, selected bool) string {
	prefix := text[:match.Start]
	ln := 1
	for j := 0; j < len(prefix); j++ {
		if prefix[j] == '\n' {
			ln++
		}
	}
	prefixStr := "  "
	if selected {
		prefixStr = "> "
	}
	return fmt.Sprintf("%s%d: %q", prefixStr, ln, text[match.Start:match.End])
}
