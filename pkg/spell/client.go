package spell

import (
    "bufio"
    "errors"
    "io"
    "os/exec"
    "strings"
    "sync"
)

// Client manages a long-lived external spell checking process communicating
// over stdio. The protocol is a single-line request of whitespace-separated
// words, and a single-line response of whitespace-separated words that need
// checking (e.g., misspellings or unknown words).
type Client struct {
    mu   sync.Mutex
    cmd  *exec.Cmd
    in   io.WriteCloser
    out  *bufio.Reader
    done chan struct{}
}

// Start launches the external process with the given command and arguments.
// The process must read from stdin and write responses to stdout.
func (c *Client) Start(command string, args ...string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.cmd != nil {
        return nil
    }
    cmd := exec.Command(command, args...)
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return err
    }
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        _ = stdin.Close()
        return err
    }
    // In case the process writes to stderr, we ignore it for now.
    if err := cmd.Start(); err != nil {
        _ = stdin.Close()
        return err
    }
    c.cmd = cmd
    c.in = stdin
    c.out = bufio.NewReader(stdout)
    c.done = make(chan struct{})
    go func() {
        _ = cmd.Wait()
        close(c.done)
    }()
    return nil
}

// Stop terminates the process if running.
func (c *Client) Stop() {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.cmd == nil {
        return
    }
    _ = c.in.Close()
    _ = c.cmd.Process.Kill()
    <-c.done
    c.cmd = nil
    c.in = nil
    c.out = nil
    c.done = nil
}

// Check sends the provided words (space-separated) and returns the subset
// that require checking, as returned by the external process. It returns
// an error if the client is not started or if the protocol fails.
func (c *Client) Check(words []string) ([]string, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.cmd == nil || c.in == nil || c.out == nil {
        return nil, errors.New("spell client is not started")
    }
    // Send a single line with whitespace-separated words.
    line := strings.Join(words, " ") + "\n"
    if _, err := io.WriteString(c.in, line); err != nil {
        return nil, err
    }
    // Read a single response line.
    resp, err := c.out.ReadString('\n')
    if err != nil {
        return nil, err
    }
    resp = strings.TrimSpace(resp)
    if resp == "" {
        return nil, nil
    }
    parts := strings.Fields(resp)
    // Normalize to lower-case for stability.
    for i := range parts {
        parts[i] = strings.ToLower(parts[i])
    }
    return parts, nil
}

