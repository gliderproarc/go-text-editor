package app

import (
    "regexp"
    "sort"
    "strings"
    "sync/atomic"

    "example.com/texteditor/pkg/search"
    "example.com/texteditor/pkg/spell"
)

// Spell checking integration: lightweight viewport-based word scanning
// and background IPC to an external spell checker.

// wordRE matches simple words (letters, digits, apostrophes within words).
var wordRE = regexp.MustCompile(`(?i)[A-Za-z][A-Za-z0-9']*`)

// SpellState is the runner's spell-check subsystem state.
type SpellState struct {
    Enabled     bool
    Client      *spell.Client
    ranges      []search.Range
    lastTopLine int
    lastMaxLines int
    running     atomic.Bool
}

// EnableSpellCheck starts the external checker process. If already started,
// it simply marks spell checking enabled.
func (r *Runner) EnableSpellCheck(command string, args ...string) error {
    if r.Spell == nil {
        r.Spell = &SpellState{}
    }
    if r.Spell.Client == nil {
        r.Spell.Client = &spell.Client{}
        if err := r.Spell.Client.Start(command, args...); err != nil {
            r.Spell.Client = nil
            return err
        }
    }
    r.Spell.Enabled = true
    // Trigger initial check on next draw
    r.updateSpellAsync()
    return nil
}

// DisableSpellCheck stops highlighting; the client remains running for now.
func (r *Runner) DisableSpellCheck() {
    if r.Spell == nil {
        return
    }
    r.Spell.Enabled = false
    r.Spell.ranges = nil
    r.draw(nil)
}

// spellHighlights returns current cached ranges for rendering.
func (r *Runner) spellHighlights() []search.Range {
    if r.Spell == nil || !r.Spell.Enabled {
        return nil
    }
    if len(r.Spell.ranges) == 0 {
        return nil
    }
    out := make([]search.Range, len(r.Spell.ranges))
    copy(out, r.Spell.ranges)
    return out
}

// updateSpellAsync scans visible lines and sends unique words to the checker.
// When a response arrives, it computes background highlight ranges and triggers
// a redraw. Calls are coalesced while a previous run is in flight.
func (r *Runner) updateSpellAsync() {
    if r.Screen == nil || r.Buf == nil || r.Spell == nil || !r.Spell.Enabled || r.Spell.Client == nil {
        return
    }
    if r.Spell.running.Load() {
        return
    }
    width, height := r.Screen.Size()
    _ = width
    mbHeight := len(r.MiniBuf)
    maxLines := height - 1 - mbHeight
    if maxLines < 0 {
        maxLines = 0
    }
    // Avoid re-scanning if viewport unchanged.
    if r.Spell.lastTopLine == r.TopLine && r.Spell.lastMaxLines == maxLines && len(r.Spell.ranges) > 0 {
        return
    }
    lines := r.Buf.Lines()
    startLine := r.TopLine
    endLine := r.TopLine + maxLines
    if endLine > len(lines) {
        endLine = len(lines)
    }
    // compute byte offset of startLine
    byteStart := 0
    for i := 0; i < startLine && i < len(lines); i++ {
        byteStart += len([]byte(lines[i])) + 1 // include newline
    }
    // Collect unique lowercase words and track positions of occurrences per word.
    wordSet := make(map[string]struct{})
    type occ struct{ s, e int }
    occs := make(map[string][]occ)
    off := byteStart
    for i := startLine; i < endLine; i++ {
        line := lines[i]
        for _, loc := range wordRE.FindAllStringIndex(line, -1) {
            w := strings.ToLower(line[loc[0]:loc[1]])
            wordSet[w] = struct{}{}
            occs[w] = append(occs[w], occ{s: off + loc[0], e: off + loc[1]})
        }
        off += len([]byte(line)) + 1
    }
    if len(wordSet) == 0 {
        r.Spell.ranges = nil
        r.draw(nil)
        return
    }
    r.Spell.running.Store(true)
    r.Spell.lastTopLine = r.TopLine
    r.Spell.lastMaxLines = maxLines
    // Prepare ordered word list for stable behavior.
    words := make([]string, 0, len(wordSet))
    for w := range wordSet {
        words = append(words, w)
    }
    sort.Strings(words)
    go func(words []string, occs map[string][]occ) {
        bad, err := r.Spell.Client.Check(words)
        // Always clear running flag when done
        r.Spell.running.Store(false)
        if err != nil {
            // On error, clear spell ranges but do not spam UI.
            r.Spell.ranges = nil
            r.draw(nil)
            return
        }
        if len(bad) == 0 {
            r.Spell.ranges = nil
            r.draw(nil)
            return
        }
        // Build highlight ranges for the bad words in the viewport.
        var rs []search.Range
        for _, w := range bad {
            if locs, ok := occs[w]; ok {
                for _, p := range locs {
                    rs = append(rs, search.Range{Start: p.s, End: p.e, Group: "bg.spell"})
                }
            }
        }
        r.Spell.ranges = rs
        r.draw(nil)
    }(words, occs)
}

