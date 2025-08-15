package testhelpers

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

// BuildBin builds a repo-local package or file at pkgPath into a temp binary
// under the repository root and returns the resulting path. It configures
// build caches and temp dirs under the repo to avoid sandbox/network issues.
func BuildBin(t *testing.T, outName, pkgPath string) string {
    t.Helper()
    cwd, err := os.Getwd()
    if err != nil { t.Fatal(err) }
    // find repo root (directory containing go.mod)
    dir := cwd
    for {
        if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil { break }
        parent := filepath.Dir(dir)
        if parent == dir { break }
        dir = parent
    }
    tmpDir, err := os.MkdirTemp(dir, "testbin-")
    if err != nil { t.Fatal(err) }
    outPath := filepath.Join(tmpDir, outName)
    cmd := exec.Command("go", "build", "-o", outPath, pkgPath)
    cmd.Dir = dir
    // keep caches and tmp under repo so tests run offline
    gocache := filepath.Join(dir, ".gocache")
    gomod := filepath.Join(dir, ".gomodcache")
    gotmp := filepath.Join(dir, ".gotmp")
    _ = os.MkdirAll(gocache, 0o755)
    _ = os.MkdirAll(gomod, 0o755)
    _ = os.MkdirAll(gotmp, 0o755)
    cmd.Env = append(os.Environ(),
        "GOCACHE="+gocache,
        "GOMODCACHE="+gomod,
        "GOTMPDIR="+gotmp,
    )
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("build %s failed: %v\n%s", pkgPath, err, string(out))
    }
    return outPath
}

