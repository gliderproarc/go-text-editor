package history

import (
    "testing"

    "example.com/texteditor/pkg/buffer"
)

func TestHistory_UndoRedo_InsertDelete(t *testing.T) {
    b := buffer.NewGapBufferFromString("abc")
    h := New()

    // Insert 'X' at position 1: aXbc
    if err := b.Insert(1, []rune("X")); err != nil {
        t.Fatalf("insert failed: %v", err)
    }
    h.RecordInsert(1, "X")
    cursor := 2
    if b.String() != "aXbc" {
        t.Fatalf("expected aXbc, got %q", b.String())
    }

    // Delete 'b' at pos 2 (positions after insert): aXc
    // slice rune at 2..3
    del := string(b.Slice(2, 3))
    if err := b.Delete(2, 3); err != nil {
        t.Fatalf("delete failed: %v", err)
    }
    h.RecordDelete(2, del)
    if b.String() != "aXc" {
        t.Fatalf("expected aXc, got %q", b.String())
    }

    // Undo delete -> aXbc
    if err := h.Undo(b, &cursor); err != nil {
        t.Fatalf("undo failed: %v", err)
    }
    if b.String() != "aXbc" {
        t.Fatalf("expected aXbc after undo, got %q", b.String())
    }

    // Undo insert -> abc
    if err := h.Undo(b, &cursor); err != nil {
        t.Fatalf("undo failed: %v", err)
    }
    if b.String() != "abc" {
        t.Fatalf("expected abc after undo, got %q", b.String())
    }

    // Redo insert -> aXbc
    if err := h.Redo(b, &cursor); err != nil {
        t.Fatalf("redo failed: %v", err)
    }
    if b.String() != "aXbc" {
        t.Fatalf("expected aXbc after redo, got %q", b.String())
    }

    // Redo delete -> aXc
    if err := h.Redo(b, &cursor); err != nil {
        t.Fatalf("redo failed: %v", err)
    }
    if b.String() != "aXc" {
        t.Fatalf("expected aXc after redo, got %q", b.String())
    }
}

func TestKillRing_Basic(t *testing.T) {
    var k KillRing
    if k.HasData() {
        t.Fatalf("expected empty kill ring")
    }
    k.Set("line1\n")
    if !k.HasData() {
        t.Fatalf("expected kill ring to have data")
    }
    if k.Get() != "line1\n" {
        t.Fatalf("unexpected kill ring content: %q", k.Get())
    }
}

