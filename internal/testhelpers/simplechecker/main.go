package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

// simplechecker implements the editor protocol for tests: it reads
// whitespace-separated words per line and echoes back known "bad" words.
func main() {
    in := bufio.NewScanner(os.Stdin)
    out := bufio.NewWriter(os.Stdout)
    defer out.Flush()
    for in.Scan() {
        line := strings.TrimSpace(in.Text())
        if line == "" { fmt.Fprintln(out); continue }
        words := strings.Fields(line)
        var bad []string
        for _, w := range words {
            lw := strings.ToLower(w)
            if lw == "mispelt" || lw == "oops" || lw == "unknown" {
                bad = append(bad, lw)
            }
        }
        fmt.Fprintln(out, strings.Join(bad, " "))
        // flush after each line so interactive clients receive responses
        _ = out.Flush()
    }
}
