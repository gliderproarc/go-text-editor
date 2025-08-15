package spell_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "example.com/texteditor/internal/testhelpers"
    "example.com/texteditor/pkg/spell"
)

func TestClientCheckBasic(t *testing.T) {
    bin := testhelpers.BuildBin(t, "simplechecker", "./internal/testhelpers/simplechecker")
    var c spell.Client
    if err := c.Start(bin); err != nil {
        t.Fatalf("start: %v", err)
    }
    defer c.Stop()
    words := []string{"Hello", "mispelt", "OK", "OOPS", "unknown"}
    bad, err := c.Check(words)
    if err != nil { t.Fatalf("check: %v", err) }
    got := strings.Join(bad, ",")
    // expect lowercased unique list; order by server echo (simplechecker preserves input order)
    wantContain := []string{"mispelt", "unknown"}
    for _, w := range wantContain {
        if !strings.Contains(got, w) {
            t.Fatalf("expected %q in %q", w, got)
        }
    }
}

func TestClientHandlesCrash(t *testing.T) {
    // Build a one-shot checker: returns an empty line then exits immediately.
    code := `package main
import ("bufio";"fmt";"os";"strings")
func main(){in:=bufio.NewScanner(os.Stdin);out:=bufio.NewWriter(os.Stdout);defer out.Flush();if in.Scan(){line:=strings.TrimSpace(in.Text());_ = line;fmt.Fprintln(out, "")};return}
`
    // Write code to repo tmp and build
    cwd, _ := os.Getwd()
    tmpFile := filepath.Join(cwd, "crash_helper.go")
    if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil { t.Fatal(err) }
    t.Cleanup(func(){ _ = os.Remove(tmpFile) })
    outPath := testhelpers.BuildBin(t, "one_shot", tmpFile)
    var c3 spell.Client
    if err := c3.Start(outPath); err != nil { t.Fatalf("start crashy: %v", err) }
    // First check may succeed (empty), but subsequent should error because process exits.
    if _, err := c3.Check([]string{"hello"}); err != nil {
        // ok: errored already
        return
    }
    if _, err := c3.Check([]string{"again"}); err == nil {
        t.Fatalf("expected error after crash, got nil")
    }
}
