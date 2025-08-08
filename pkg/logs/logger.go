package logs

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// Logger writes JSON lines with a timestamp and event fields.
type Logger struct {
    mu      sync.Mutex
    w       *bufio.Writer
    f       *os.File
    enabled bool
}

// NewFromEnv returns a logger if TEXTEDITOR_LOG is set to a truthy value
// or if TEXTEDITOR_LOG_FILE is provided. Otherwise it returns a disabled logger.
// When enabled and no file is specified, it writes to ./texteditor.log.
func NewFromEnv() *Logger {
    lf := os.Getenv("TEXTEDITOR_LOG_FILE")
    enabled := false
    if v := os.Getenv("TEXTEDITOR_LOG"); v != "" && v != "0" && v != "false" {
        enabled = true
    }
    if lf != "" {
        enabled = true
    }
    if !enabled {
        return &Logger{enabled: false}
    }
    if lf == "" {
        lf = filepath.Join(".", "texteditor.log")
    }
    f, err := os.OpenFile(lf, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        // If we cannot open the requested file, disable logging silently.
        return &Logger{enabled: false}
    }
    return &Logger{w: bufio.NewWriter(f), f: f, enabled: true}
}

// Close flushes and closes the underlying file if enabled.
func (l *Logger) Close() {
    if !l.enabled {
        return
    }
    l.mu.Lock()
    defer l.mu.Unlock()
    _ = l.w.Flush()
    _ = l.f.Close()
}

// Event writes a JSON line with the event name and fields.
// Common fields: key, rune, modifiers, action, cursor, buffer_len, file.
func (l *Logger) Event(event string, fields map[string]any) {
    if !l.enabled {
        return
    }
    rec := map[string]any{
        "time":  time.Now().Format(time.RFC3339Nano),
        "event": event,
    }
    for k, v := range fields {
        rec[k] = v
    }
    l.mu.Lock()
    defer l.mu.Unlock()
    enc := json.NewEncoder(l.w)
    _ = enc.Encode(rec)
    _ = l.w.Flush()
}

