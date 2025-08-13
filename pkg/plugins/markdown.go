package plugins

import (
    "bytes"
    "regexp"

    "example.com/texteditor/pkg/search"
)

// MarkdownHighlighter provides very basic, fast heuristics-based highlighting for Markdown.
// It highlights:
// - ATX headings starting with '#'
// - Code fences and their contents
// - Inline code `code`
// - Blockquotes starting with '>'
// - List markers (-, *, +, or numbered)
// - Links [text](url)
// Groups map onto existing theme keys: function (headings), string (code),
// comment (blockquote), keyword (markers/links), type (link text).
type MarkdownHighlighter struct{}

func NewMarkdownHighlighter() *MarkdownHighlighter { return &MarkdownHighlighter{} }

func (m *MarkdownHighlighter) Name() string { return "markdown-basic" }

func (m *MarkdownHighlighter) Highlight(src []byte) []search.Range {
    if len(src) == 0 {
        return nil
    }
    var ranges []search.Range
    // Regexes used per line
    // Inline code: `code`
    inlineCode := regexp.MustCompile("`[^`\n]+`")
    // Links: [text](url)
    link := regexp.MustCompile(`\[[^\]]+\]\([^\)]+\)`)
    // Emphasis/bold: **text** or *text* or _text_
    emphasis := regexp.MustCompile(`(\*\*[^\*\n]+\*\*|\*[^\*\n]+\*|_[^_\n]+_)`)

    // Track fenced code blocks
    inFence := false
    fenceLang := ""
    off := 0
    for len(src) > 0 {
        // split line including no trailing newline
        idx := bytes.IndexByte(src, '\n')
        var line []byte
        if idx == -1 {
            line = src
        } else {
            line = src[:idx]
        }

        trim := bytes.TrimLeft(line, " \t")
        // Code fence toggles
        if bytes.HasPrefix(trim, []byte("```")) || bytes.HasPrefix(trim, []byte("~~~")) {
            // mark the fence line as string
            ranges = append(ranges, search.Range{Start: off, End: off + len(line), Group: "string"})
            // toggle fence state
            if inFence {
                inFence = false
                fenceLang = ""
            } else {
                inFence = true
                // capture optional language id after fence markers
                if len(trim) > 3 {
                    fenceLang = string(bytes.TrimSpace(trim[3:]))
                }
            }
        } else if inFence {
            // Entire line inside fence is code (string)
            ranges = append(ranges, search.Range{Start: off, End: off + len(line), Group: "string"})
        } else {
            // Headings: starts with # after optional spaces
            if bytes.HasPrefix(trim, []byte{'#'}) {
                // highlight full line as a heading (use function for bold/blue)
                ranges = append(ranges, search.Range{Start: off, End: off + len(line), Group: "function"})
            }
            // Blockquote lines starting with '>'
            if bytes.HasPrefix(trim, []byte{'>'}) {
                ranges = append(ranges, search.Range{Start: off, End: off + len(line), Group: "comment"})
            }
            // List markers at start of trimmed line
            if bytes.HasPrefix(trim, []byte("- ")) || bytes.HasPrefix(trim, []byte("* ")) || bytes.HasPrefix(trim, []byte("+ ")) {
                // highlight just the marker (2 bytes)
                markerStart := off + (len(line) - len(trim))
                ranges = append(ranges, search.Range{Start: markerStart, End: markerStart + 2, Group: "keyword"})
            } else {
                // Numbered list: 1. or 1)
                // Scan digits at start of trim
                di := 0
                for di < len(trim) && trim[di] >= '0' && trim[di] <= '9' {
                    di++
                }
                if di > 0 && di < len(trim) && (trim[di] == '.' || trim[di] == ')') {
                    markerStart := off + (len(line) - len(trim))
                    ranges = append(ranges, search.Range{Start: markerStart, End: markerStart + di + 1, Group: "keyword"})
                }
            }

            // Inline patterns within the line
            // Inline code as string
            for _, loc := range inlineCode.FindAllIndex(line, -1) {
                ranges = append(ranges, search.Range{Start: off + loc[0], End: off + loc[1], Group: "string"})
            }
            // Links: highlight whole link as keyword and the [text] as type
            for _, loc := range link.FindAllIndex(line, -1) {
                start := off + loc[0]
                end := off + loc[1]
                ranges = append(ranges, search.Range{Start: start, End: end, Group: "keyword"})
                // text part inside []
                l := line[loc[0]:loc[1]]
                if lb := bytes.IndexByte(l, '['); lb != -1 {
                    if rb := bytes.IndexByte(l, ']'); rb != -1 && rb > lb {
                        ranges = append(ranges, search.Range{Start: start + lb, End: start + rb + 1, Group: "type"})
                    }
                }
            }
            // Emphasis as type
            for _, loc := range emphasis.FindAllIndex(line, -1) {
                ranges = append(ranges, search.Range{Start: off + loc[0], End: off + loc[1], Group: "type"})
            }
        }

        // advance
        if idx == -1 {
            break
        }
        // include the newline in offset
        off += idx + 1
        src = src[idx+1:]
    }
    _ = fenceLang
    return ranges
}

