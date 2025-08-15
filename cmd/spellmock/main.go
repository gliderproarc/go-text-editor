package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "unicode"
)

// spellmock is a tiny mock spell checker:
// - Reads lines from stdin; each line contains whitespace-separated words.
// - Writes back a single line with words deemed "misspelled".
// Heuristic: flag words that contain any digits, or are ALL CAPS (>=3 chars),
// or are longer than 14 letters.
func main() {
    in := bufio.NewScanner(os.Stdin)
    out := bufio.NewWriter(os.Stdout)
    defer out.Flush()
    for in.Scan() {
        line := in.Text()
        words := strings.Fields(line)
        var bad []string
        for _, w := range words {
            if needsCheck(w) {
                bad = append(bad, strings.ToLower(w))
            }
        }
        fmt.Fprintln(out, strings.Join(bad, " "))
    }
}

func needsCheck(w string) bool {
    if len(w) == 0 {
        return false
    }
    // contains digits
    for _, r := range w {
        if unicode.IsDigit(r) {
            return true
        }
    }
    // ALL CAPS and length >= 3 (likely acronym; flag for demo)
    if len([]rune(w)) >= 3 {
        allUpper := true
        anyLetter := false
        for _, r := range w {
            if unicode.IsLetter(r) {
                anyLetter = true
                if !unicode.IsUpper(r) {
                    allUpper = false
                    break
                }
            }
        }
        if anyLetter && allUpper {
            return true
        }
    }
    // overly long words
    letters := 0
    for _, r := range w {
        if unicode.IsLetter(r) {
            letters++
        }
    }
    if letters > 14 {
        return true
    }
    return false
}

