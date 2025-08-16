package main

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"
)

// slowchecker delays its response by SLOW_MS milliseconds, then echoes back
// the subset of words considered misspelled: mispelt, oops, unknown.
// Used to simulate a slow or stalled checker to test timeouts.
func main() {
    in := bufio.NewScanner(os.Stdin)
    out := bufio.NewWriter(os.Stdout)
    defer out.Flush()
    delay := 2000 * time.Millisecond
    if v := os.Getenv("SLOW_MS"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n >= 0 {
            delay = time.Duration(n) * time.Millisecond
        }
    }
    for in.Scan() {
        line := strings.TrimSpace(in.Text())
        time.Sleep(delay)
        if line == "" { fmt.Fprintln(out); _ = out.Flush(); continue }
        words := strings.Fields(line)
        var bad []string
        for _, w := range words {
            lw := strings.ToLower(w)
            if lw == "mispelt" || lw == "oops" || lw == "unknown" {
                bad = append(bad, lw)
            }
        }
        fmt.Fprintln(out, strings.Join(bad, " "))
        _ = out.Flush()
    }
}

