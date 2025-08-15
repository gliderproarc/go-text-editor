package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

// aspellfake simulates a subset of `aspell -a` behavior for tests.
// It prints a header and then for each input word (one per line), writes:
//   *  for correct words
//   & w ... for misspelled words with suggestions
//   # w ... for unknown words without suggestions
//   + root ... when a root form is found
// Behavior can be tweaked via ASPFAKE_MODE:
//   normal: classify some words (*, &, #, +)
//   crash:  print header then exit immediately
func main() {
    mode := os.Getenv("ASPFAKE_MODE")
    // Header similar to aspell
    fmt.Println("@(#) International Ispell Version 3.1.20 (mock Aspell 0.0.1)")
    if mode == "crash" {
        return
    }
    in := bufio.NewScanner(os.Stdin)
    out := bufio.NewWriter(os.Stdout)
    defer out.Flush()
    for in.Scan() {
        w := strings.TrimSpace(in.Text())
        if w == "" { continue }
        lw := strings.ToLower(w)
        switch lw {
        case "good", "hello":
            fmt.Fprintln(out, "*")
        case "rooted":
            fmt.Fprintln(out, "+ root 0 0")
        case "mispelt", "unknown":
            fmt.Fprintf(out, "& %s 1 0: suggestion\n", w)
        default:
            // treat words with digits as unknown
            hasDigit := false
            for _, r := range w {
                if r >= '0' && r <= '9' { hasDigit = true; break }
            }
            if hasDigit {
                fmt.Fprintf(out, "# %s 0\n", w)
            } else {
                fmt.Fprintln(out, "*")
            }
        }
        // flush after each response line to mimic interactive aspell
        _ = out.Flush()
    }
}
