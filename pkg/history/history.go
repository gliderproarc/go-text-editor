package history

import (
    "fmt"

    "example.com/texteditor/pkg/buffer"
)

// OpType represents the type of an edit operation.
type OpType int

const (
    InsertOp OpType = iota
    DeleteOp
)

// Operation captures a single edit for undo/redo.
// Pos is a rune index; Text is the inserted/deleted text.
type Operation struct {
    Type OpType
    Pos  int
    Text string
}

// History keeps stacks of past/future operations for undo/redo.
type History struct {
    past   []Operation
    future []Operation
}

// New creates an empty History.
func New() *History { return &History{} }

// RecordInsert records an insertion at pos.
func (h *History) RecordInsert(pos int, text string) {
    if text == "" {
        return
    }
    h.past = append(h.past, Operation{Type: InsertOp, Pos: pos, Text: text})
    h.future = nil
}

// RecordDelete records a deletion at pos of the given text.
func (h *History) RecordDelete(pos int, text string) {
    if text == "" {
        return
    }
    h.past = append(h.past, Operation{Type: DeleteOp, Pos: pos, Text: text})
    h.future = nil
}

// CanUndo reports whether there is an operation to undo.
func (h *History) CanUndo() bool { return len(h.past) > 0 }

// CanRedo reports whether there is an operation to redo.
func (h *History) CanRedo() bool { return len(h.future) > 0 }

// Undo applies the inverse of the last operation to buf and updates the cursor.
func (h *History) Undo(buf *buffer.GapBuffer, cursor *int) error {
    if !h.CanUndo() {
        return fmt.Errorf("nothing to undo")
    }
    op := h.past[len(h.past)-1]
    h.past = h.past[:len(h.past)-1]
    // Apply inverse
    switch op.Type {
    case InsertOp:
        // inverse of insert is delete the inserted text
        start := op.Pos
        end := op.Pos + len([]rune(op.Text))
        if err := buf.Delete(start, end); err != nil {
            return err
        }
        if cursor != nil {
            if *cursor > end {
                *cursor -= len([]rune(op.Text))
            } else {
                // move cursor to start of deletion if it was inside
                if *cursor > start {
                    *cursor = start
                }
            }
        }
        // push onto future as original op for redo
        h.future = append(h.future, op)
        return nil
    case DeleteOp:
        // inverse of delete is insert the deleted text
        if err := buf.Insert(op.Pos, []rune(op.Text)); err != nil {
            return err
        }
        if cursor != nil {
            if *cursor >= op.Pos {
                *cursor += len([]rune(op.Text))
            }
        }
        h.future = append(h.future, op)
        return nil
    default:
        return fmt.Errorf("unknown op type")
    }
}

// Redo reapplies the next operation to buf and updates the cursor.
func (h *History) Redo(buf *buffer.GapBuffer, cursor *int) error {
    if !h.CanRedo() {
        return fmt.Errorf("nothing to redo")
    }
    op := h.future[len(h.future)-1]
    h.future = h.future[:len(h.future)-1]
    switch op.Type {
    case InsertOp:
        if err := buf.Insert(op.Pos, []rune(op.Text)); err != nil {
            return err
        }
        if cursor != nil {
            if *cursor >= op.Pos {
                *cursor += len([]rune(op.Text))
            }
        }
        h.past = append(h.past, op)
        return nil
    case DeleteOp:
        start := op.Pos
        end := op.Pos + len([]rune(op.Text))
        if err := buf.Delete(start, end); err != nil {
            return err
        }
        if cursor != nil {
            if *cursor > end {
                *cursor -= len([]rune(op.Text))
            } else {
                if *cursor > start {
                    *cursor = start
                }
            }
        }
        h.past = append(h.past, op)
        return nil
    default:
        return fmt.Errorf("unknown op type")
    }
}

