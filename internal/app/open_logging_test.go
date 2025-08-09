package app

import (
    "bufio"
    "encoding/json"
    "os"
    "testing"

    "example.com/texteditor/pkg/buffer"
    "example.com/texteditor/pkg/history"
    "example.com/texteditor/pkg/logs"
)

// eventRec matches the structure written by pkg/logs.Logger
type eventRec struct {
    Time  string                 `json:"time"`
    Event string                 `json:"event"`
    Fields map[string]any        `json:"-"`
}

func TestLoadFile_EmitsLoggingEvents(t *testing.T) {
    // Prepare a temp log file and direct logger to it
    logf, err := os.CreateTemp("", "texteditor_log_*.jsonl")
    if err != nil { t.Fatalf("CreateTemp log: %v", err) }
    logPath := logf.Name()
    _ = logf.Close()
    defer os.Remove(logPath)

    // Ensure env points to our temp file for NewFromEnv
    t.Setenv("TEXTEDITOR_LOG_FILE", logPath)
    t.Setenv("TEXTEDITOR_LOG", "1")

    r := &Runner{Buf: buffer.NewGapBuffer(0), History: history.New()}
    r.Logger = logs.NewFromEnv()
    if r.Logger == nil {
        t.Fatalf("expected logger from env")
    }

    // 1) Non-existent path -> expect open.attempt and open.error
    bad := "/this/does/not/exist.txt"
    _ = r.LoadFile(bad)

    // 2) Valid temp file -> expect open.attempt and open.success
    tf, err := os.CreateTemp("", "texteditor_file_*.txt")
    if err != nil { t.Fatalf("CreateTemp file: %v", err) }
    path := tf.Name()
    _, _ = tf.WriteString("hello\r\nworld\r\n")
    _ = tf.Close()
    defer os.Remove(path)

    if err := r.LoadFile(path); err != nil {
        t.Fatalf("LoadFile on temp file: %v", err)
    }
    if got := r.Buf.String(); got != "hello\nworld\n" {
        t.Fatalf("expected normalized content, got %q", got)
    }

    // Read back the log file and assert events exist
    f, err := os.Open(logPath)
    if err != nil { t.Fatalf("open log: %v", err) }
    defer f.Close()

    haveAttemptBad := false
    haveErrorBad := false
    haveAttemptGood := false
    haveSuccessGood := false

    s := bufio.NewScanner(f)
    for s.Scan() {
        line := s.Bytes()
        var rec map[string]any
        if err := json.Unmarshal(line, &rec); err != nil {
            t.Fatalf("unmarshal log line: %v", err)
        }
        ev, _ := rec["event"].(string)
        file, _ := rec["file"].(string)
        switch ev {
        case "open.attempt":
            if file == bad { haveAttemptBad = true }
            if file == path { haveAttemptGood = true }
        case "open.error":
            if file == bad { haveErrorBad = true }
        case "open.success":
            if file == path { haveSuccessGood = true }
        }
    }
    if err := s.Err(); err != nil {
        t.Fatalf("scan log: %v", err)
    }

    if !haveAttemptBad || !haveErrorBad {
        t.Fatalf("expected attempt+error for bad path; got attempt=%v error=%v", haveAttemptBad, haveErrorBad)
    }
    if !haveAttemptGood || !haveSuccessGood {
        t.Fatalf("expected attempt+success for good path; got attempt=%v success=%v", haveAttemptGood, haveSuccessGood)
    }
}

