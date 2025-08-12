package buffer

import (
	"fmt"
	"strings"
)

// GapBuffer is a simple gap-buffer implementation for runes.
// The underlying slice stores runes with a gap between gapStart and gapEnd.
type GapBuffer struct {
	buf      []rune
	gapStart int
	gapEnd   int

	cacheString string
	cacheLines  []string
	cacheValid  bool
}

// NewGapBuffer creates an empty GapBuffer with an initial capacity.
func NewGapBuffer(cap int) *GapBuffer {
	if cap < 1 {
		cap = 128
	}
	b := make([]rune, cap)
	return &GapBuffer{buf: b, gapStart: 0, gapEnd: cap}
}

// NewGapBufferFromString initializes a GapBuffer with the provided text.
func NewGapBufferFromString(s string) *GapBuffer {
	runes := []rune(s)
	cap := len(runes) + 128
	b := NewGapBuffer(cap)
	// place the runes before the gap
	copy(b.buf, runes)
	b.gapStart = len(runes)
	b.gapEnd = cap
	return b
}

func (g *GapBuffer) ensureGap(n int) {
	gap := g.gapEnd - g.gapStart
	if gap >= n {
		return
	}
	// grow buffer: double size or add n
	needed := n - gap
	newCap := len(g.buf)*2 + needed
	newBuf := make([]rune, newCap)
	// copy prefix
	copy(newBuf, g.buf[:g.gapStart])
	// copy suffix after gap to end
	suffixLen := len(g.buf) - g.gapEnd
	copy(newBuf[newCap-suffixLen:], g.buf[g.gapEnd:])
	g.gapEnd = newCap - suffixLen
	g.buf = newBuf
}

// moveGap moves the gap so that gapStart == pos
func (g *GapBuffer) moveGap(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > g.Len() {
		pos = g.Len()
	}
	if pos == g.gapStart {
		return
	}
	if pos < g.gapStart {
		// move prefix left into gap
		d := g.gapStart - pos
		for i := 0; i < d; i++ {
			g.buf[g.gapEnd-1-i] = g.buf[g.gapStart-1-i]
			// optional: zero out moved slot
			// g.buf[g.gapStart-1-i] = 0
		}
		g.gapEnd -= d
		g.gapStart = pos
		return
	}
	// pos > gapStart: move suffix into gap
	if pos > g.gapStart {
		d := pos - g.gapStart
		for i := 0; i < d; i++ {
			g.buf[g.gapStart+i] = g.buf[g.gapEnd+i]
			// g.buf[g.gapEnd+i] = 0
		}
		g.gapStart += d
		g.gapEnd += d
		return
	}
}

// Insert inserts runes at position pos (0..Len()).
func (g *GapBuffer) Insert(pos int, s []rune) error {
	if pos < 0 || pos > g.Len() {
		return fmt.Errorf("position out of range")
	}
	g.moveGap(pos)
	g.ensureGap(len(s))
	// copy into gap
	for i, r := range s {
		g.buf[g.gapStart+i] = r
	}
	g.gapStart += len(s)
	g.cacheValid = false
	return nil
}

// Delete removes runes in [start,end).
func (g *GapBuffer) Delete(start, end int) error {
	if start < 0 || end < start || end > g.Len() {
		return fmt.Errorf("invalid range")
	}
	g.moveGap(start)
	// expand gap by (end-start) from the right
	d := end - start
	g.gapEnd += d
	if g.gapEnd > len(g.buf) {
		g.gapEnd = len(g.buf)
	}
	g.cacheValid = false
	return nil
}

// Slice returns a slice of runes in [start,end)
func (g *GapBuffer) Slice(start, end int) []rune {
	if start < 0 {
		start = 0
	}
	if end > g.Len() {
		end = g.Len()
	}
	if start >= end {
		return []rune{}
	}
	out := make([]rune, end-start)
	// iterate through and copy
	idx := 0
	for i := 0; i < g.Len() && idx < (end-start); i++ {
		r := g.runeAt(i)
		if i >= start && i < end {
			out[idx] = r
			idx++
		}
	}
	return out
}

// Len returns the logical length (excluding gap)
func (g *GapBuffer) Len() int {
	return len(g.buf) - (g.gapEnd - g.gapStart)
}

// RuneAt returns the rune at index i. If i is out of bounds, it returns 0.
func (g *GapBuffer) RuneAt(i int) rune {
	if i < 0 || i >= g.Len() {
		return 0
	}
	return g.runeAt(i)
}

func (g *GapBuffer) runeAt(i int) rune {
	if i < g.gapStart {
		return g.buf[i]
	}
	return g.buf[g.gapEnd+(i-g.gapStart)]
}

// LineAt returns the rune start and end indices for the given line number
// (0-based). If the line index is past the end, it returns the last line's
// bounds. The end index is one past the last rune of the line (i.e., it will
// include the terminating '\n' when present).
func (g *GapBuffer) LineAt(idx int) (start, end int) {
	if idx < 0 {
		idx = 0
	}
	if g.Len() == 0 {
		return 0, 0
	}
	line := 0
	start = 0
	for i := 0; i < g.Len(); i++ {
		if line == idx {
			// find end
			for j := i; j < g.Len(); j++ {
				if g.runeAt(j) == '\n' {
					// return j+1 so the newline is included
					return i, j + 1
				}
			}
			return i, g.Len()
		}
		if g.runeAt(i) == '\n' {
			line++
			start = i + 1
		}
	}
	// if idx beyond last, return last line
	return start, g.Len()
}

// String returns the buffer as a string (for debugging)
func (g *GapBuffer) String() string {
	if g.cacheValid {
		return g.cacheString
	}
	out := make([]rune, 0, g.Len())
	for i := 0; i < g.Len(); i++ {
		out = append(out, g.runeAt(i))
	}
	g.cacheString = string(out)
	g.cacheLines = strings.Split(g.cacheString, "\n")
	g.cacheValid = true
	return g.cacheString
}

// Lines returns the buffer split into lines. The result is cached until the
// buffer is modified.
func (g *GapBuffer) Lines() []string {
	if !g.cacheValid {
		_ = g.String()
	}
	return g.cacheLines
}
