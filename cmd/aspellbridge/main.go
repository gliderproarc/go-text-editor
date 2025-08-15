package main

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "os/exec"
    "strings"
)

// aspellbridge adapts our editor's simple spell protocol to aspell -a.
// Protocol in: one line of whitespace-separated words on stdin.
// Protocol out: one line of whitespace-separated words that are misspelled.
func main() {
    // Allow language override via TEXTEDITOR_SPELL_LANG (e.g., en_US)
    lang := os.Getenv("TEXTEDITOR_SPELL_LANG")
    args := []string{"-a"}
    if lang != "" {
        args = append(args, "--lang="+lang)
    }

    cmd := exec.Command("aspell", args...)
    aspellIn, err := cmd.StdinPipe()
    if err != nil {
        fmt.Fprintln(os.Stderr, "aspell stdin pipe error:", err)
        os.Exit(1)
    }
    aspellOut, err := cmd.StdoutPipe()
    if err != nil {
        fmt.Fprintln(os.Stderr, "aspell stdout pipe error:", err)
        os.Exit(1)
    }
    aspellErr, _ := cmd.StderrPipe()
    if err := cmd.Start(); err != nil {
        fmt.Fprintln(os.Stderr, "failed to start aspell:", err)
        os.Exit(1)
    }
    // Best-effort: drain and discard aspell stderr asynchronously
    go io.Copy(io.Discard, aspellErr)

    outR := bufio.NewReader(aspellOut)
    inW := bufio.NewWriter(aspellIn)

    // aspell -a prints a header line; consume it.
    _, _ = outR.ReadString('\n')

    // Bridge loop: read input lines, check each word via aspell -a
    stdin := bufio.NewScanner(os.Stdin)
    stdout := bufio.NewWriter(os.Stdout)
    defer stdout.Flush()

    for stdin.Scan() {
        line := strings.TrimSpace(stdin.Text())
        if line == "" {
            fmt.Fprintln(stdout)
            continue
        }
        words := strings.Fields(line)
        var bad []string
        for _, w := range words {
            if w == "" {
                continue
            }
            // Send the word to aspell and flush so it processes immediately.
            if _, err := inW.WriteString(w + "\n"); err != nil {
                // On write error, abort the bridge.
                fmt.Fprintln(os.Stderr, "aspell write error:", err)
                return
            }
            if err := inW.Flush(); err != nil {
                fmt.Fprintln(os.Stderr, "aspell flush error:", err)
                return
            }
            // Read a single response line for this word.
            resp, err := outR.ReadString('\n')
            if err != nil {
                fmt.Fprintln(os.Stderr, "aspell read error:", err)
                return
            }
            if len(resp) == 0 {
                continue
            }
            switch resp[0] {
            case '*', '+':
                // correct or root form found; do nothing
            case '&', '#':
                // misspelled or unknown
                bad = append(bad, strings.ToLower(w))
            default:
                // Unexpected; be conservative and skip
            }
        }
        fmt.Fprintln(stdout, strings.Join(bad, " "))
    }
}
