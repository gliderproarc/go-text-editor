package plugins

import "testing"

func TestMarkdownHighlight_Basics(t *testing.T) {
    md := NewMarkdownHighlighter()
    src := []byte("# Title\n\nSome `code` and a [link](https://example.com).\n\n- item\n> quote\n")
    hs := md.Highlight(src)
    if len(hs) == 0 {
        t.Fatalf("expected some highlights for markdown")
    }
    // Expect at least one heading and one inline code and one link
    var gotHeading, gotCode, gotLink bool
    for _, h := range hs {
        if h.Group == "function" { // heading
            gotHeading = true
        }
        if h.Group == "string" { // inline code or fence
            gotCode = true
        }
        if h.Group == "keyword" { // link token
            gotLink = true
        }
    }
    if !gotHeading {
        t.Fatalf("expected heading highlight")
    }
    if !gotCode {
        t.Fatalf("expected inline code highlight")
    }
    if !gotLink {
        t.Fatalf("expected link highlight")
    }
}

