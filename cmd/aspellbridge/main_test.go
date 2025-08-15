package main_test

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "example.com/texteditor/internal/testhelpers"
)

func TestAspellBridgeClassifiesShapes(t *testing.T) {
    bridge := testhelpers.BuildBin(t, "aspellbridge", "./cmd/aspellbridge")
    // Build fake as 'aspell' so bridge finds it on PATH
    fake := testhelpers.BuildBin(t, "aspell", "./internal/testhelpers/aspellfake")
    path := filepath.Dir(fake) + string(os.PathListSeparator) + os.Getenv("PATH")
    cmd := exec.Command(bridge)
    cmd.Env = append(os.Environ(), "PATH="+path)
    // Provide one line of input and let stdin close so the bridge exits
    cmd.Stdin = strings.NewReader("good mispelt unknown rooted x123\n")
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("bridge run failed: %v\n%s", err, string(out))
    }
    // Only one output line is expected
    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    if len(lines) == 0 { t.Fatalf("no output from bridge") }
    joined := strings.TrimSpace(lines[len(lines)-1])
    // Expect misspellings only: mispelt unknown x123
    for _, w := range []string{"mispelt", "unknown", "x123"} {
        if !strings.Contains(joined, w) {
            t.Fatalf("expected %q in output %q", w, joined)
        }
    }
    // And ensure known are not present
    for _, w := range []string{"good", "rooted"} {
        if strings.Contains(joined, w) {
            t.Fatalf("did not expect %q in output %q", w, joined)
        }
    }
}

func TestAspellBridgeAspellMissing(t *testing.T) {
    bridge := testhelpers.BuildBin(t, "aspellbridge", "./cmd/aspellbridge")
    cmd := exec.Command(bridge)
    // Ensure aspell is not found
    cmd.Env = append(os.Environ(), "PATH=/nonexistent")
    if err := cmd.Run(); err == nil {
        t.Fatalf("expected error when aspell missing, got nil")
    }
}

func TestAspellBridgeAspellCrash(t *testing.T) {
    bridge := testhelpers.BuildBin(t, "aspellbridge", "./cmd/aspellbridge")
    fake := testhelpers.BuildBin(t, "aspell", "./internal/testhelpers/aspellfake")
    path := filepath.Dir(fake) + string(os.PathListSeparator) + os.Getenv("PATH")
    cmd := exec.Command(bridge)
    cmd.Env = append(os.Environ(), "PATH="+path, "ASPFAKE_MODE=crash")
    // Provide one line; fake exits after header, bridge should not hang
    cmd.Stdin = strings.NewReader("hello world\n")
    // We accept either error or success; the key is it returns promptly
    done := make(chan struct{})
    var runErr error
    go func() { _, runErr = cmd.CombinedOutput(); close(done) }()
    select {
    case <-done:
        _ = runErr // ignore value; behavior may vary
    case <-time.After(3 * time.Second):
        t.Fatalf("bridge hung when aspell crashed")
    }
}
