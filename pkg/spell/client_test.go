package spell_test

import (
	"os"
	"strings"
	"testing"

	"example.com/texteditor/internal/testhelpers"
	"example.com/texteditor/pkg/spell"
)

func buildEOFChecker(t *testing.T) string {
	t.Helper()
	code := `package main
import (
    "bufio"
    "fmt"
    "os"
    "strings"
)
func main(){in:=bufio.NewScanner(os.Stdin);out:=bufio.NewWriter(os.Stdout);defer out.Flush();if in.Scan(){line:=strings.ToLower(strings.TrimSpace(in.Text()));if strings.Contains(line,"mispelt"){fmt.Fprint(out,"mispelt")}}}
`
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp, err := os.CreateTemp(cwd, "eof_checker-*.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(code); err != nil {
		_ = tmp.Close()
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })
	return testhelpers.BuildBin(t, "eof_checker", tmp.Name())
}

func TestClientCheckBasic(t *testing.T) {
	bin := testhelpers.BuildBin(t, "simplechecker", "./internal/testhelpers/simplechecker")
	var c spell.Client
	if err := c.Start(bin); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Stop()
	words := []string{"Hello", "mispelt", "OK", "OOPS", "unknown"}
	bad, err := c.Check(words)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	got := strings.Join(bad, ",")
	// expect lowercased unique list; order by server echo (simplechecker preserves input order)
	wantContain := []string{"mispelt", "unknown"}
	for _, w := range wantContain {
		if !strings.Contains(got, w) {
			t.Fatalf("expected %q in %q", w, got)
		}
	}
}

func TestClientCheckEOFResponse(t *testing.T) {
	bin := buildEOFChecker(t)
	var c spell.Client
	if err := c.Start(bin); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Stop()
	bad, err := c.Check([]string{"mispelt"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(bad) != 1 || bad[0] != "mispelt" {
		t.Fatalf("expected mispelt, got %v", bad)
	}
}

func buildCrashChecker(t *testing.T) string {
	t.Helper()
	// Build a one-shot checker: exits immediately without responding.
	code := `package main
func main(){return}
`
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp, err := os.CreateTemp(cwd, "crash_helper-*.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(code); err != nil {
		_ = tmp.Close()
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })
	return testhelpers.BuildBin(t, "one_shot", tmp.Name())
}

func TestClientHandlesCrash(t *testing.T) {
	outPath := buildCrashChecker(t)
	var c3 spell.Client
	if err := c3.Start(outPath); err != nil {
		t.Fatalf("start crashy: %v", err)
	}
	// First check should be treated as an error because the process exits mid-response.
	if _, err := c3.Check([]string{"hello"}); err == nil {
		t.Fatalf("expected error after crash, got nil")
	}

	// Second check must also fail because the client is no longer running.
	if _, err := c3.Check([]string{"again"}); err == nil {
		t.Fatalf("expected error on subsequent check, got nil")
	}
}
