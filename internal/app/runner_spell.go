package app

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"example.com/texteditor/pkg/search"
	"example.com/texteditor/pkg/spell"
)

// Spell checking integration: lightweight viewport-based word scanning
// and background IPC to an external spell checker.

// wordRE matches simple words (letters, digits, apostrophes within words).
var wordRE = regexp.MustCompile(`(?i)[A-Za-z][A-Za-z0-9']*`)

// SpellState is the runner's spell-check subsystem state.
type SpellState struct {
	Enabled      bool
	Client       *spell.Client
	ranges       []search.Range
	lastTopLine  int
	lastMaxLines int
	running      atomic.Bool
	// lastEditSeq is the Runner.editSeq value the last time we scanned.
	lastEditSeq int64
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
	// Coalesce when viewport unchanged AND content unchanged.
	if r.Spell.lastTopLine == r.TopLine && r.Spell.lastMaxLines == maxLines && r.Spell.lastEditSeq == r.editSeq && len(r.Spell.ranges) > 0 {
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
	r.Spell.lastEditSeq = r.editSeq
	// Prepare ordered word list for stable behavior.
	words := make([]string, 0, len(wordSet))
	for w := range wordSet {
		words = append(words, w)
	}
	sort.Strings(words)
	go func(words []string, occs map[string][]occ) {
		// Use a timeout to avoid hanging the background worker on a stuck checker.
		bad, err := r.Spell.Client.CheckWithTimeout(words, spellTimeout())
		// Always clear running flag when done
		r.Spell.running.Store(false)
		if err != nil {
			// On error, clear spell ranges but do not spam UI.
			// If the client timed out or failed, reset it so the next call can restart.
			if r.Spell != nil && r.Spell.Client != nil {
				r.Spell.Client.Stop()
				r.Spell.Client = nil
			}
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

// ensureSpellClient ensures the spell client process is started without toggling
// the background highlighting feature. It honors TEXTEDITOR_SPELL env, then
// falls back to local helpers.
func (r *Runner) ensureSpellClient() error {
	if r.Spell == nil {
		r.Spell = &SpellState{}
	}
	if r.Spell.Client != nil {
		return nil
	}
	cmd := os.Getenv("TEXTEDITOR_SPELL")
	var err error
	r.Spell.Client = &spell.Client{}
	if cmd != "" {
		err = r.Spell.Client.Start(cmd)
	} else {
		// Try aspell bridge first; fall back to mock
		if err = r.Spell.Client.Start("./aspellbridge"); err != nil {
			err = r.Spell.Client.Start("./spellmock")
		}
	}
	if err != nil {
		r.Spell.Client = nil
	}
	return err
}

// CheckWordAtCursor checks the word under the cursor using the spell client
// and displays a transient message in the mini-buffer. If the cursor is not on
// a word, it shows "No word found".
func (r *Runner) CheckWordAtCursor() string {
	if r.Buf == nil || r.Buf.Len() == 0 {
		return ""
	}
	text := r.Buf.String()
	// Limit scan to current line for efficiency
	lineStartRune, lineEndRune := r.currentLineBounds()
	startByte := runeIndexToByteOffset(text, lineStartRune)
	endByte := runeIndexToByteOffset(text, lineEndRune)
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(text) {
		endByte = len(text)
	}
	line := text[startByte:endByte]
	curByte := runeIndexToByteOffset(text, r.Cursor)
	rel := curByte - startByte
	if rel < 0 {
		rel = 0
	}
	if rel > len(line) {
		rel = len(line)
	}

	// Find the word span on this line that contains the cursor
	var word string
	locs := wordRE.FindAllStringIndex(line, -1)
	for _, loc := range locs {
		if rel >= loc[0] && rel < loc[1] { // cursor strictly inside the word span
			word = strings.ToLower(line[loc[0]:loc[1]])
			break
		}
	}
	if word == "" {
		r.setMiniBuffer([]string{"No word found"})
		r.draw(nil)
		r.clearMiniBuffer()
		return "No word found"
	}
	// Perform a single-word check with a timeout to avoid freezing the UI.
	timeout := spellTimeout()
	bad, err := r.checkWordsOnceWithTimeout([]string{word}, timeout)
	if err != nil {
		msg := "Spell check failed: " + err.Error()
		if errorsIsDeadline(err) {
			msg = "Spell check timed out"
		}
		r.setMiniBuffer([]string{msg})
		r.draw(nil)
		r.clearMiniBuffer()
		return msg
	}
	// Determine result
	miss := false
	for _, b := range bad {
		if b == word {
			miss = true
			break
		}
	}
	msg := "OK: " + word
	if miss {
		msg = "Misspelled: " + word
	}
	r.setMiniBuffer([]string{msg})
	r.draw(nil)
	r.clearMiniBuffer()
	return msg
}

// errorsIsDeadline reports whether the error indicates a context deadline.
func errorsIsDeadline(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// Fallback substring check to be safe across exec errors
	return strings.Contains(strings.ToLower(err.Error()), "deadline exceeded")
}

// spellTimeout reads a timeout from TEXTEDITOR_SPELL_TIMEOUT_MS, defaulting to 500ms.
func spellTimeout() time.Duration {
	const def = 500 * time.Millisecond
	v := os.Getenv("TEXTEDITOR_SPELL_TIMEOUT_MS")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Millisecond
}

// checkWordsOnceWithTimeout runs a one-shot spell check command with a timeout.
// It uses TEXTEDITOR_SPELL if set, otherwise falls back to local helpers.
func (r *Runner) checkWordsOnceWithTimeout(words []string, timeout time.Duration) ([]string, error) {
	cmdPath := os.Getenv("TEXTEDITOR_SPELL")
	if cmdPath == "" {
		// Prefer aspell bridge; fall back to mock
		if _, err := os.Stat("./aspellbridge"); err == nil {
			cmdPath = "./aspellbridge"
		} else {
			cmdPath = "./spellmock"
		}
	}
	cmd := exec.Command(cmdPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}

	// Write request line
	if _, err := io.WriteString(stdin, strings.Join(words, " ")+"\n"); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return nil, err
	}
	_ = stdin.Close()

	type result struct {
		bad []string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		br := bufio.NewReader(stdout)
		line, err := br.ReadString('\n')
		_ = cmd.Wait()
		if err != nil && !errors.Is(err, io.EOF) {
			resCh <- result{nil, err}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			if errors.Is(err, io.EOF) {
				resCh <- result{nil, err}
				return
			}
			resCh <- result{nil, nil}
			return
		}
		parts := strings.Fields(line)
		for i := range parts {
			parts[i] = strings.ToLower(parts[i])
		}
		resCh <- result{parts, nil}
	}()

	select {
	case r := <-resCh:
		return r.bad, r.err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		// Ensure process is reaped
		go cmd.Wait()
		return nil, context.DeadlineExceeded
	}
}
