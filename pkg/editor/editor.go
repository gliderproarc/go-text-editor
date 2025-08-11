package editor

import (
	"os"
	"strings"

	"example.com/texteditor/pkg/buffer"
)

// BufferState holds the state of a single editor buffer.
type BufferState struct {
	FilePath string
	Buf      *buffer.GapBuffer
	Cursor   int
	Dirty    bool
}

// Editor manages multiple buffers and the focused buffer index.
type Editor struct {
	Buffers []BufferState
	Current int
}

// New creates an empty Editor.
func New() *Editor {
	return &Editor{}
}

// AddBuffer appends a buffer and makes it the current one.
func (e *Editor) AddBuffer(bs BufferState) {
	e.Buffers = append(e.Buffers, bs)
	e.Current = len(e.Buffers) - 1
}

// UpdateCurrent stores the provided state into the current buffer.
func (e *Editor) UpdateCurrent(bs BufferState) {
	if e.Current >= 0 && e.Current < len(e.Buffers) {
		e.Buffers[e.Current] = bs
	}
}

// CurrentBuffer returns the active buffer state.
func (e *Editor) CurrentBuffer() BufferState {
	if e.Current >= 0 && e.Current < len(e.Buffers) {
		return e.Buffers[e.Current]
	}
	return BufferState{}
}

// Next advances focus to the next buffer and returns it.
func (e *Editor) Next() BufferState {
	if len(e.Buffers) == 0 {
		return BufferState{}
	}
	e.Current = (e.Current + 1) % len(e.Buffers)
	return e.Buffers[e.Current]
}

// Prev moves focus to the previous buffer and returns it.
func (e *Editor) Prev() BufferState {
	if len(e.Buffers) == 0 {
		return BufferState{}
	}
	e.Current = (e.Current - 1 + len(e.Buffers)) % len(e.Buffers)
	return e.Buffers[e.Current]
}

// LoadFile reads a file and adds it as a new buffer.
func (e *Editor) LoadFile(path string) (BufferState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BufferState{}, err
	}
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	gb := buffer.NewGapBufferFromString(normalized)
	bs := BufferState{FilePath: path, Buf: gb, Cursor: gb.Len(), Dirty: false}
	e.AddBuffer(bs)
	return bs, nil
}
