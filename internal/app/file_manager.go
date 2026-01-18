package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/texteditor/pkg/buffer"
	"example.com/texteditor/pkg/history"
	"github.com/gdamore/tcell/v2"
)

type View int

const (
	ViewEditor View = iota
	ViewFileManager
)

type fileManagerState struct {
	Dir     string
	Entries []fileEntry
	Return  fileManagerReturnState
}

type fileManagerReturnState struct {
	FilePath    string
	Buf         *buffer.GapBuffer
	Cursor      int
	CursorLine  int
	TopLine     int
	Dirty       bool
	Mode        Mode
	VisualStart int
	VisualLine  bool
	MultiEdit   *multiEditState
	History     *history.History
	KillRing    history.KillRing
	EditSeq     int64
}

type fileEntry struct {
	Name      string
	Path      string
	Info      os.FileInfo
	IsDir     bool
	Editable  bool
	Line      string
	NameStart int
}

type fileRename struct {
	entryIndex int
	oldName    string
	newName    string
	oldLine    string
	newLine    string
	oldPath    string
	newPath    string
}

func (r *Runner) runFileManager() {
	startDir := r.fileManagerStartDir()
	r.enterFileManager(startDir)
}

func (r *Runner) handleFileManagerKey(ev *tcell.EventKey) bool {
	if r.FileManager == nil {
		r.View = ViewEditor
		return false
	}
	if r.matchCommand(ev, "quit") {
		return true
	}
	if r.isCancelKey(ev) {
		if r.Mode == ModeInsert || r.Mode == ModeMultiEdit {
			r.fileManagerExitInsertMode()
			return false
		}
		r.exitFileManager()
		return false
	}
	if r.Mode == ModeInsert || r.Mode == ModeMultiEdit {
		return r.handleFileManagerInsertKey(ev)
	}
	if r.Mode == ModeVisual {
		return r.handleFileManagerVisualKey(ev)
	}
	if ev.Key() == tcell.KeyEnter {
		if entry, ok := r.fileManagerCurrentEntry(); ok {
			r.openFileManagerEntry(entry)
		}
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'i' && ev.Modifiers() == 0 {
		entry, ok := r.fileManagerCurrentEntry()
		if ok && entry.Editable {
			nameStart := r.fileManagerNameStartPos(r.CursorLine)
			r.Cursor = nameStart
			r.Mode = ModeInsert
			r.beginInsertCapture(1, nil)
			r.draw(nil)
		}
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'a' && ev.Modifiers() == 0 {
		entry, ok := r.fileManagerCurrentEntry()
		if ok && entry.Editable {
			end := r.cursorLineEnd(r.CursorLine)
			nameStart := r.fileManagerNameStartPos(r.CursorLine)
			if end < nameStart {
				end = nameStart
			}
			r.Cursor = end
			r.Mode = ModeInsert
			r.beginInsertCapture(1, nil)
			r.draw(nil)
		}
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'v' && ev.Modifiers() == 0 {
		r.Mode = ModeVisual
		r.VisualStart = r.Cursor
		r.VisualLine = false
		r.draw(nil)
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'V' && ev.Modifiers() == 0 {
		start, _ := r.currentLineBounds()
		r.Cursor = start
		r.Mode = ModeVisual
		r.VisualStart = start
		r.VisualLine = true
		r.draw(nil)
		return false
	}
	if r.Mode == ModeVisual && ev.Key() == tcell.KeyRune && ev.Rune() == 'v' && ev.Modifiers() == 0 {
		r.Mode = ModeNormal
		r.VisualStart = -1
		r.VisualLine = false
		r.draw(nil)
		return false
	}
	if r.Mode == ModeVisual && ev.Key() == tcell.KeyRune && ev.Rune() == ' ' && ev.Modifiers() == 0 {
		return r.runMnemonicMenu()
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == ' ' && ev.Modifiers() == 0 {
		return r.runMnemonicMenu()
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0 {
		if r.PendingG {
			r.PendingG = false
			r.Cursor = 0
			r.CursorLine = 0
			r.draw(nil)
		} else {
			r.PendingG = true
		}
		return false
	}
	if r.PendingG && !(ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0) {
		r.PendingG = false
	}
	if r.Mode == ModeNormal && r.PendingG && ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0 {
		r.PendingG = false
		r.Cursor = 0
		r.CursorLine = 0
		r.draw(nil)
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'G' && ev.Modifiers() == 0 {
		if r.Buf != nil && r.Buf.Len() > 0 {
			lines := r.Buf.Lines()
			last := len(lines) - 1
			if last < 0 {
				last = 0
			}
			r.CursorLine = last
			r.Cursor = r.cursorFromLine(last)
			r.draw(nil)
		}
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'l' && ev.Modifiers() == 0 {
		entry, ok := r.fileManagerCurrentEntry()
		if ok && entry.IsDir {
			r.openFileManagerEntry(entry)
		}
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == 0 {
		if r.FileManager != nil {
			parent := filepath.Dir(r.FileManager.Dir)
			_ = r.loadFileManagerDir(parent)
			r.CursorLine = 0
			r.Cursor = 0
			r.TopLine = 0
			r.draw(nil)
		}
		return false
	}
	if r.Mode == ModeNormal || r.Mode == ModeVisual {
		switch {
		case ev.Key() == tcell.KeyUp || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == 0):
			r.moveCursorVertical(-1)
			r.draw(nil)
			return false
		case ev.Key() == tcell.KeyDown || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'j' && ev.Modifiers() == 0):
			r.moveCursorVertical(1)
			r.draw(nil)
			return false
		case ev.Key() == tcell.KeyLeft || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == 0):
			if r.Cursor > 0 {
				r.Cursor--
			}
			r.draw(nil)
			return false
		case ev.Key() == tcell.KeyRight || (!r.isInsertMode() && ev.Key() == tcell.KeyRune && ev.Rune() == 'l' && ev.Modifiers() == 0):
			if r.Buf != nil && r.Cursor < r.Buf.Len() {
				r.Cursor++
			}
			r.draw(nil)
			return false
		}
	}
	return false
}

func (r *Runner) handleFileManagerInsertKey(ev *tcell.EventKey) bool {
	if r.Mode == ModeMultiEdit {
		return r.handleFileManagerMultiEditKey(ev)
	}
	switch {
	case ev.Key() == tcell.KeyEnter:
		r.fileManagerExitInsertMode()
		return false
	case ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2:
		if r.Cursor > 0 {
			start := r.Cursor - 1
			deleted := string(r.Buf.Slice(start, r.Cursor))
			_ = r.deleteRange(start, r.Cursor, deleted)
			if r.CursorLine > 0 && start > 0 && r.Buf.RuneAt(start-1) == '\n' {
				r.CursorLine--
			}
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyDelete:
		if r.Buf != nil && r.Cursor < r.Buf.Len() {
			if r.Buf.RuneAt(r.Cursor) == '\n' {
				return false
			}
			deleted := string(r.Buf.Slice(r.Cursor, r.Cursor+1))
			_ = r.deleteRange(r.Cursor, r.Cursor+1, deleted)
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Modifiers() == 0:
		if ev.Rune() != '\n' {
			r.insertText(string(ev.Rune()))
			r.draw(nil)
		}
		return false
	case ev.Key() == tcell.KeyCtrlA:
		start, _ := r.currentLineBounds()
		r.Cursor = start
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyCtrlE:
		end := r.cursorLineEnd(r.CursorLine)
		r.Cursor = end
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'm' && ev.Modifiers() == tcell.ModAlt:
		return r.runMnemonicMenu()
	}
	return false
}

func (r *Runner) handleFileManagerMultiEditKey(ev *tcell.EventKey) bool {
	switch {
	case ev.Key() == tcell.KeyEnter:
		r.fileManagerExitInsertMode()
		return false
	case r.isCancelKey(ev):
		r.fileManagerExitInsertMode()
		return false
	case ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2:
		if r.Cursor > 0 {
			start := r.Cursor - 1
			deleted := string(r.Buf.Slice(start, r.Cursor))
			_ = r.deleteRange(start, r.Cursor, deleted)
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyDelete:
		if r.Buf != nil && r.Cursor < r.Buf.Len() {
			if r.Buf.RuneAt(r.Cursor) == '\n' {
				return false
			}
			deleted := string(r.Buf.Slice(r.Cursor, r.Cursor+1))
			_ = r.deleteRange(r.Cursor, r.Cursor+1, deleted)
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Modifiers() == 0:
		if ev.Rune() != '\n' {
			r.insertText(string(ev.Rune()))
			r.draw(nil)
		}
		return false
	case ev.Key() == tcell.KeyCtrlA:
		start, _ := r.currentLineBounds()
		r.Cursor = start
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyCtrlE:
		end := r.cursorLineEnd(r.CursorLine)
		r.Cursor = end
		r.draw(nil)
		return false
	}
	return false
}

func (r *Runner) handleFileManagerVisualKey(ev *tcell.EventKey) bool {
	if r.PendingG && !(ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0) {
		r.PendingG = false
	}
	switch {
	case r.isCancelKey(ev) || (ev.Key() == tcell.KeyRune && ev.Rune() == 'v' && ev.Modifiers() == 0):
		r.Mode = ModeNormal
		r.VisualStart = -1
		r.VisualLine = false
		r.PendingG = false
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'g' && ev.Modifiers() == 0:
		if r.PendingG {
			r.PendingG = false
			r.Cursor = 0
			r.CursorLine = 0
			r.draw(nil)
			return false
		}
		r.PendingG = true
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'G' && ev.Modifiers() == 0:
		if r.Buf != nil && r.Buf.Len() > 0 {
			lines := r.Buf.Lines()
			last := len(lines) - 1
			if last < 0 {
				last = 0
			}
			r.CursorLine = last
			r.Cursor = r.cursorFromLine(last)
			r.draw(nil)
		}
		return false
	case ev.Key() == tcell.KeyUp || (ev.Key() == tcell.KeyRune && ev.Rune() == 'k' && ev.Modifiers() == 0):
		r.moveCursorVertical(-1)
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyDown || (ev.Key() == tcell.KeyRune && ev.Rune() == 'j' && ev.Modifiers() == 0):
		r.moveCursorVertical(1)
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyLeft || (ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers() == 0):
		if r.Cursor > 0 {
			r.Cursor--
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRight || (ev.Key() == tcell.KeyRune && ev.Rune() == 'l' && ev.Modifiers() == 0):
		if r.Buf != nil && r.Cursor < r.Buf.Len() {
			r.Cursor++
		}
		r.draw(nil)
		return false
	case ev.Key() == tcell.KeyRune && ev.Rune() == ' ' && ev.Modifiers() == 0:
		return r.runMnemonicMenu()
	}
	return false
}

func (r *Runner) cursorLineEnd(line int) int {
	if r.Buf == nil {
		return 0
	}
	start, end := r.Buf.LineAt(line)
	if end > start && r.Buf.RuneAt(end-1) == '\n' {
		return end - 1
	}
	return end
}

func (r *Runner) fileManagerNameStartPos(line int) int {
	fm := r.FileManager
	if fm == nil {
		return r.cursorFromLine(line)
	}
	if line < 0 || line >= len(fm.Entries) {
		return r.cursorFromLine(line)
	}
	nameStart := fm.Entries[line].NameStart
	return r.cursorFromLine(line) + nameStart
}

func (r *Runner) fileManagerStartDir() string {
	if r.FileManager != nil {
		if r.FileManager.Dir != "" {
			return r.FileManager.Dir
		}
		if r.FileManager.Return.FilePath != "" {
			return filepath.Dir(r.FileManager.Return.FilePath)
		}
	}
	if r.FilePath != "" && !strings.HasPrefix(r.FilePath, "[File Manager]") {
		return filepath.Dir(r.FilePath)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func (r *Runner) enterFileManager(dir string) {
	if r.Screen == nil {
		return
	}
	if r.FileManager == nil {
		r.saveBufferState()
	}
	returnState := fileManagerReturnState{
		FilePath:    r.FilePath,
		Buf:         r.Buf,
		Cursor:      r.Cursor,
		CursorLine:  r.CursorLine,
		TopLine:     r.TopLine,
		Dirty:       r.Dirty,
		Mode:        r.Mode,
		VisualStart: r.VisualStart,
		VisualLine:  r.VisualLine,
		MultiEdit:   r.MultiEdit,
		History:     r.History,
		KillRing:    r.KillRing,
		EditSeq:     r.editSeq,
	}
	fm := r.FileManager
	if fm != nil {
		fm.Return = returnState
	}
	r.FileManager = &fileManagerState{Dir: dir, Return: returnState}
	r.History = history.New()
	r.KillRing = history.KillRing{}
	r.editSeq = 0
	r.View = ViewFileManager
	r.FilePath = "[File Manager] " + dir
	r.Mode = ModeNormal
	r.VisualStart = -1
	r.VisualLine = false
	r.MultiEdit = nil
	r.PendingG = false
	r.PendingD = false
	r.PendingY = false
	r.PendingC = false
	r.PendingTextObject = false
	r.TextObjectAround = false
	r.PendingCount = 0
	r.clearMiniBuffer()
	if err := r.loadFileManagerDir(dir); err != nil {
		r.showDialog("File manager: " + err.Error())
		r.exitFileManager()
		return
	}
	r.draw(nil)
}

func (r *Runner) exitFileManager() {
	if r.FileManager == nil {
		return
	}
	ret := r.FileManager.Return
	r.FileManager = nil
	r.View = ViewEditor
	r.FilePath = ret.FilePath
	r.Buf = ret.Buf
	r.Cursor = ret.Cursor
	r.CursorLine = ret.CursorLine
	r.TopLine = ret.TopLine
	r.Dirty = ret.Dirty
	r.Mode = ret.Mode
	r.VisualStart = ret.VisualStart
	r.VisualLine = ret.VisualLine
	r.MultiEdit = ret.MultiEdit
	r.History = ret.History
	r.KillRing = ret.KillRing
	r.editSeq = ret.EditSeq
	r.clearMiniBuffer()
	r.draw(nil)
}

func (r *Runner) openFileManagerEntry(entry fileEntry) {
	if entry.IsDir {
		_ = r.loadFileManagerDir(entry.Path)
		r.Cursor = 0
		r.CursorLine = 0
		r.TopLine = 0
		r.draw(nil)
		return
	}
	r.exitFileManager()
	if err := r.LoadFile(entry.Path); err != nil {
		r.showDialog("Open failed: " + err.Error())
		return
	}
	r.draw(nil)
}

func (r *Runner) loadFileManagerDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	fm := r.FileManager
	if fm == nil {
		return fmt.Errorf("file manager not active")
	}
	fm.Dir = dir
	fm.Entries = nil
	// Parent entry
	parent := filepath.Dir(dir)
	parentInfo, _ := os.Stat(parent)
	fm.Entries = append(fm.Entries, buildFileEntry("..", parent, parentInfo, true, false))
	// Ensure stable ordering (os.ReadDir returns sorted, but keep explicit)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		fm.Entries = append(fm.Entries, buildFileEntry(name, path, info, true, true))
	}
	lines := make([]string, 0, len(fm.Entries))
	for i, entry := range fm.Entries {
		line, nameStart := formatFileEntryLine(entry)
		fm.Entries[i].Line = line
		fm.Entries[i].NameStart = nameStart
		lines = append(lines, line)
	}
	r.Buf = buffer.NewGapBufferFromString(strings.Join(lines, "\n"))
	r.Cursor = 0
	r.CursorLine = 0
	r.TopLine = 0
	r.Dirty = false
	r.syntaxSrc = ""
	r.syntaxCache = nil
	return nil
}

func buildFileEntry(name, path string, info os.FileInfo, editable bool, allowRename bool) fileEntry {
	entry := fileEntry{
		Name:     name,
		Path:     path,
		Info:     info,
		Editable: editable && allowRename,
	}
	if info != nil {
		entry.IsDir = info.IsDir()
	}
	return entry
}

func formatFileEntryLine(entry fileEntry) (string, int) {
	mode := "----------"
	size := int64(0)
	modTime := time.Now()
	if entry.Info != nil {
		mode = entry.Info.Mode().String()
		if entry.IsDir && !strings.HasPrefix(mode, "d") {
			mode = "d" + mode
		}
		size = entry.Info.Size()
		modTime = entry.Info.ModTime()
	}
	stamp := modTime.Format("Jan 02 15:04")
	prefix := fmt.Sprintf("%s %8d %s ", mode, size, stamp)
	line := prefix + entry.Name
	return line, len([]rune(prefix))
}

func (r *Runner) fileManagerCurrentEntry() (fileEntry, bool) {
	fm := r.FileManager
	if fm == nil {
		return fileEntry{}, false
	}
	idx := r.CursorLine
	if idx < 0 || idx >= len(fm.Entries) {
		return fileEntry{}, false
	}
	return fm.Entries[idx], true
}

func (r *Runner) fileManagerExitInsertMode() {
	changes, err := r.fileManagerRenameChanges()
	if err != nil {
		r.showDialog("Rename error: " + err.Error())
		r.refreshFileManagerListing()
		r.Mode = ModeNormal
		r.MultiEdit = nil
		r.draw(nil)
		return
	}
	if len(changes) == 0 {
		r.refreshFileManagerListing()
		r.Mode = ModeNormal
		r.MultiEdit = nil
		r.draw(nil)
		return
	}
	if !r.confirmFileManagerRenames(changes) {
		r.refreshFileManagerListing()
		r.Mode = ModeNormal
		r.MultiEdit = nil
		r.draw(nil)
		return
	}
	if err := r.applyFileManagerRenames(changes); err != nil {
		r.showDialog("Rename failed: " + err.Error())
	}
	r.refreshFileManagerListing()
	r.Mode = ModeNormal
	r.MultiEdit = nil
	r.draw(nil)
}

func (r *Runner) refreshFileManagerListing() {
	if r.FileManager == nil {
		return
	}
	cursorLine := r.CursorLine
	dir := r.FileManager.Dir
	_ = r.loadFileManagerDir(dir)
	if cursorLine < 0 {
		cursorLine = 0
	}
	if cursorLine >= len(r.FileManager.Entries) {
		cursorLine = len(r.FileManager.Entries) - 1
	}
	if cursorLine < 0 {
		cursorLine = 0
	}
	r.CursorLine = cursorLine
	r.Cursor = r.cursorFromLine(cursorLine)
	r.ensureCursorVisible()
}

func (r *Runner) cursorFromLine(line int) int {
	if r.Buf == nil || line <= 0 {
		return 0
	}
	start, _ := r.Buf.LineAt(line)
	return start
}

func (r *Runner) fileManagerRenameChanges() ([]fileRename, error) {
	fm := r.FileManager
	if fm == nil || r.Buf == nil {
		return nil, nil
	}
	lines := r.Buf.Lines()
	changes := make([]fileRename, 0)
	for i, entry := range fm.Entries {
		if !entry.Editable {
			continue
		}
		if i >= len(lines) {
			continue
		}
		line := lines[i]
		newName := extractFileName(line, entry.NameStart)
		if newName == "" {
			return nil, fmt.Errorf("invalid name for %s", entry.Name)
		}
		if strings.ContainsRune(newName, os.PathSeparator) {
			return nil, fmt.Errorf("invalid name %q", newName)
		}
		if newName == "." || newName == ".." {
			return nil, fmt.Errorf("invalid name %q", newName)
		}
		if newName == entry.Name {
			continue
		}
		updated := entry
		updated.Name = newName
		newLine, _ := formatFileEntryLine(updated)
		changes = append(changes, fileRename{
			entryIndex: i,
			oldName:    entry.Name,
			newName:    newName,
			oldLine:    entry.Line,
			newLine:    newLine,
			oldPath:    entry.Path,
			newPath:    filepath.Join(fm.Dir, newName),
		})
	}
	return changes, nil
}

func extractFileName(line string, nameStart int) string {
	runes := []rune(line)
	if nameStart >= len(runes) {
		return ""
	}
	name := string(runes[nameStart:])
	return strings.TrimRight(name, " ")
}

func (r *Runner) confirmFileManagerRenames(changes []fileRename) bool {
	if r.Screen == nil {
		return false
	}
	lines := r.fileManagerDiffLines(changes)
	prompt := "Really edit this file? (y/n)"
	if len(changes) > 1 {
		prompt = "Really edit these files? (y/n)"
	}
	lines = append(lines, prompt)
	r.setMiniBuffer(lines)
	r.draw(nil)
	for {
		ev := r.waitEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if r.matchCommand(ev, "quit") {
				r.clearMiniBuffer()
				return false
			}
			if r.isCancelKey(ev) || (ev.Key() == tcell.KeyRune && (ev.Rune() == 'n' || ev.Rune() == 'N')) {
				r.clearMiniBuffer()
				return false
			}
			if ev.Key() == tcell.KeyRune && (ev.Rune() == 'y' || ev.Rune() == 'Y') {
				r.clearMiniBuffer()
				return true
			}
		}
	}
}

func (r *Runner) fileManagerDiffLines(changes []fileRename) []string {
	width := 0
	if r.Screen != nil {
		width, _ = r.Screen.Size()
	}
	clip := func(line string) string {
		if width <= 0 {
			return line
		}
		runes := []rune(line)
		if len(runes) > width {
			return string(runes[:width])
		}
		return line
	}
	if len(changes) == 1 {
		before := "- " + changes[0].oldLine
		after := "+ " + changes[0].newLine
		return []string{clip(before), clip(after)}
	}
	maxLines := len(changes)
	if maxLines > 5 {
		maxLines = 5
	}
	lines := make([]string, 0, maxLines*2+2)
	for i := 0; i < maxLines; i++ {
		lines = append(lines, clip("- "+changes[i].oldLine))
	}
	if len(changes) > maxLines {
		lines = append(lines, clip(fmt.Sprintf("... and %d more", len(changes)-maxLines)))
	}
	for i := 0; i < maxLines; i++ {
		lines = append(lines, clip("+ "+changes[i].newLine))
	}
	if len(changes) > maxLines {
		lines = append(lines, clip(fmt.Sprintf("... and %d more", len(changes)-maxLines)))
	}
	return lines
}

func (r *Runner) applyFileManagerRenames(changes []fileRename) error {
	if err := validateFileManagerRenames(changes); err != nil {
		return err
	}
	for _, change := range changes {
		if err := os.Rename(change.oldPath, change.newPath); err != nil {
			return err
		}
	}
	return nil
}

func validateFileManagerRenames(changes []fileRename) error {
	seen := make(map[string]struct{})
	oldNames := make(map[string]struct{})
	for _, change := range changes {
		oldNames[change.oldName] = struct{}{}
	}
	for _, change := range changes {
		if _, ok := seen[change.newName]; ok {
			return fmt.Errorf("duplicate rename target: %s", change.newName)
		}
		seen[change.newName] = struct{}{}
		if change.newName == "" {
			return fmt.Errorf("empty rename target")
		}
		if _, ok := oldNames[change.newName]; ok {
			continue
		}
		if _, err := os.Stat(change.newPath); err == nil {
			return fmt.Errorf("target exists: %s", change.newName)
		}
	}
	return nil
}
