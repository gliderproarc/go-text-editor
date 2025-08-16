package app

import (
    "path/filepath"
    "sync/atomic"
    "time"

    "example.com/texteditor/pkg/plugins"
    "example.com/texteditor/pkg/search"
)

// SyntaxState holds async syntax highlighting state.
type SyntaxState struct {
    ranges      []search.Range
    running     atomic.Bool
    lastEditSeq int64
    lastLang    string // highlighter name used for last compute
}

// syntaxHighlightsCached returns the last computed syntax highlight ranges.
func (r *Runner) syntaxHighlightsCached() []search.Range {
    if r == nil || r.SyntaxAsync == nil {
        return nil
    }
    rs := r.SyntaxAsync.ranges
    if len(rs) == 0 {
        return nil
    }
    out := make([]search.Range, len(rs))
    copy(out, rs)
    return out
}

// syntaxTimeout currently unused for cancellation (compute is not cancelable),
// but kept for symmetry with spell and potential future use.
func syntaxTimeout() time.Duration { return spellTimeout() }

// updateSyntaxAsync schedules a background highlight computation for the
// current buffer snapshot if needed. Results are applied only if still current
// by edit sequence and language when the computation finishes.
func (r *Runner) updateSyntaxAsync() {
    if r.Buf == nil {
        return
    }
    if r.SyntaxAsync == nil {
        r.SyntaxAsync = &SyntaxState{}
    }
    // Detect language by file extension using configured languages
    var lang *plugins.LanguageSpec
    if r.FilePath != "" {
        cfg := plugins.LoadLanguageConfig(filepath.Join("config", "languages.json"))
        lang = plugins.DetectLanguageByPath(cfg, r.FilePath)
    }
    if lang == nil {
        // No highlighter for this file; clear any existing ranges.
        if len(r.SyntaxAsync.ranges) != 0 {
            r.SyntaxAsync.ranges = nil
            r.draw(nil)
        }
        return
    }
    // Coalesce if we have up-to-date ranges and no edits since.
    if r.SyntaxAsync.lastEditSeq == r.editSeq && r.SyntaxAsync.lastLang == lang.Highlighter && len(r.SyntaxAsync.ranges) > 0 {
        return
    }
    // Avoid launching if one is currently running and content/lang unchanged.
    if r.SyntaxAsync.running.Load() && r.SyntaxAsync.lastEditSeq == r.editSeq && r.SyntaxAsync.lastLang == lang.Highlighter {
        return
    }
    // Snapshot inputs for the worker.
    src := r.Buf.String()
    seq := r.editSeq
    langName := lang.Highlighter
    r.SyntaxAsync.lastEditSeq = seq
    r.SyntaxAsync.lastLang = langName
    r.SyntaxAsync.running.Store(true)

    go func(src string, seq int64, lang *plugins.LanguageSpec) {
        // Create a fresh highlighter instance for this run.
        h := plugins.HighlighterFor(lang)
        var ranges []search.Range
        if h != nil {
            ranges = h.Highlight([]byte(src))
        }
        // Apply if still current; discard if stale.
        if r.SyntaxAsync != nil && r.SyntaxAsync.lastEditSeq == seq && r.SyntaxAsync.lastLang == lang.Highlighter {
            r.SyntaxAsync.ranges = ranges
            r.SyntaxAsync.running.Store(false)
            r.draw(nil)
            return
        }
        // Stale result; just clear running flag.
        r.SyntaxAsync.running.Store(false)
    }(src, seq, lang)
}

