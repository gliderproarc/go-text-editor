<TextEditorTest> — A Go CLI Text Editor

A tiny nano-like editor that can grow toward Emacs

Goal: Start with a reliable, minimalist, cross-platform terminal editor (nano clone), then expand toward Emacs-style power via a contextual command menu and a plug-in system.

⸻

0) Quick Start (for the coding agent)
	•	Language: Go 1.22+
	•	Terminal UI: Prefer tcell (github.com/gdamore/tcell/v2) for cross-platform input/rendering.
	•	Build:

go build ./cmd/<project>


	•	Run:

./<project> [path/to/file]


	•	Test:

go test ./...



⸻

1) Project Phases & Milestones

We’ll ship in small, testable increments. Each milestone has acceptance criteria.

M0 — Scaffold & Event Loop
	•	Create module, folders (see Structure).
	•	Initialize terminal (raw mode), main loop, clean shutdown on Ctrl+Q.
	•	Render a static screen and status bar; show current file name or [No File].
Done when: App starts, draws a status bar, exits cleanly on Ctrl+Q; unit test covers event loop tick handler.

M1 — Open/Display & Basic Navigation (nano parity, read-only)
	•	Load small/medium files into a text buffer.
	•	Display with scrolling; handle cursor movement: arrows, PageUp/Down, Home/End.
	•	Status line shows: filename, cursor line:col, modification flag.
Done when: Can open a file and navigate end-to-end without editing; tests cover file load and viewport math.

M2 — Editing Core (insert, delete, newlines)
	•	Implement TextStorage interface and start with a gap buffer (simple and fast enough).
	•	Insert characters, Backspace/Delete, Enter (LF newlines).
	•	Mark buffer “dirty”; enable Ctrl+S to save.
Done when: Can edit text, save to disk, reopen and see changes; tests cover insert/delete correctness.

M3 — Search (incremental) & Go-to
	•	Ctrl+W: incremental search UI with highlights; Enter to jump; Esc to cancel.
	•	Ctrl+_ (or Alt+G): go-to line.
Done when: Search works across file; tests cover search index and wrap-around.

M4 — Clipboard-like actions (line kill/yank) & Undo/Redo v1
	•	Ctrl+K: cut line to an internal kill ring (single slot is fine for v1).
	•	Ctrl+U: paste line.
	•	Ctrl+Z/Ctrl+Y: undo/redo using a simple history stack.
Done when: Basic cut/paste and undo/redo function with tests on edit history.

M5 — Config & Keymaps
	•	Read ~/.<project>/config.(yaml|toml) for settings: theme, keybindings.
	•	Keymap system maps sequences to commands.
Done when: Users can remap a key (e.g., save); tests on key resolution.

M6 — Multiple Buffers & Simple Windowing
	•	Open multiple files in memory; switch buffers; optional horizontal/vertical split (one level).
Done when: Open two files and flip focus; tests assert buffer focus and rendering state.

M7 — Contextual Command Menu (foundation for “M-x” feel)
	•	Key: F2 (or Alt+X) opens a popup “command palette”.
	•	Filter by fuzzy match; show context-valid commands (see Context Model).
Done when: Core commands appear, are filterable, and execute; tests cover menu filtering.

M8 — Plug-in API v0 (external process, JSON-RPC over stdio)
	•	Define minimal RPC: Register, ListCommands, Execute.
	•	Provide editor capabilities via Context: buffer text slice, selection, cursor, file path, filetype.
	•	Ship one reference plug-in (e.g., word count) in Go or Python.
Done when: Plugin registers commands visible in the contextual menu and can mutate the buffer via RPC; end-to-end test with a sample plug-in.

M9 — Quality Pass & Large Files
	•	Swap gapbuffer behind TextStorage with a piece table implementation (flag-gated).
	•	Add basic performance tests with a ~5–20MB file.
Done when: Large files remain responsive within reasonable limits (document thresholds in README).

⸻

2) Non-Goals (early)
	•	Full Emacs parity (modes, elisp) — out of scope early.
	•	Mouse support — optional later.
	•	Binary diff/merge tools — later.
	•	Windows console “legacy mode” quirks — best effort; keep tcell default paths.

⸻

3) Architecture Overview

Core Packages (proposed)

cmd/<project>/        # main entry
internal/app/         # bootstrap, DI wiring (internal vis)
pkg/editor/           # Editor orchestrator, lifecycle
pkg/buffer/           # TextStorage interface + gapbuffer, piecetable
pkg/view/             # Viewport, status bar, popups
pkg/input/            # Key handling, keymap, chords
pkg/render/           # Drawing via tcell
pkg/files/            # I/O, encodings, line endings
pkg/search/           # Incremental search
pkg/history/          # Undo/Redo
pkg/config/           # Config load/validate
pkg/commands/         # Built-in commands
pkg/menu/             # Contextual command menu
pkg/plugins/          # Plugin manager, JSON-RPC transport, capability guards

Key Data Structures
	•	Editor: holds Buffers, focused Window, Keymap, Config, PluginManager.
	•	Buffer: wraps TextStorage, Cursor, Selection, Dirty flag, FilePath, FileType.
	•	TextStorage (interface):

type TextStorage interface {
    Insert(pos int, s []rune) error
    Delete(start, end int) error
    Slice(start, end int) []rune
    Len() int
    LineAt(idx int) (start, end int)
}


	•	Context (passed to commands/plugins):

type Context struct {
    BufferID   string
    FilePath   string
    FileType   string
    Cursor     Position
    Selection  *Range // nil if none
    ReadOnly   bool
    // Editor services (capability-limited in plugins):
    GetText(start, end int) (string, error)
    Replace(start, end int, with string) error
    Insert(pos int, s string) error
    Message(msg string)
}



Command Model
	•	Commands are functions with signature:
func(ctx *Context, args map[string]any) error
	•	Registered in a catalog with:
	•	Name (id), Title (for UI), When predicate (context applicability), Category, DefaultKeybinds.

⸻

4) Contextual Command Menu (v1)
	•	Trigger: F2 (or Alt+X).
	•	Shows a list of applicable commands from Core + Plugins.
	•	Input supports fuzzy filtering; Enter executes selected command.
	•	Context passed includes FileType (to enable language-aware commands), Selection, Cursor.
	•	UI: popup panel centered; help line shows keybinding for selected item.

Extension later: categories/tabs, preview pane, and per-mode menus.

⸻

5) Plug-in System (Path to Power)

Design Choice

Use out-of-process plugins communicating via JSON-RPC 2.0 over stdio (or pipes).
Why:
	•	Portable across platforms and architectures.
	•	Safer: crash/isolate; no Go plugin ABI issues.
	•	Language-agnostic; authors can use Go/Python/Rust/…

Transport
	•	Editor spawns plugin executable listed in config.
	•	Handshake:
	•	Editor → Plugin: Register (editor version, capabilities).
	•	Plugin → Editor: declares commands[] with metadata.
	•	Runtime:
	•	Editor → Plugin: Execute with Context and args.
	•	Plugin → Editor: ApplyEdits / PostMessage via RPC callbacks.

Minimal RPC (v0)

// Editor -> Plugin
{ "jsonrpc":"2.0", "method":"Register", "params":{"editorVersion":"0.1.0"}, "id":1 }
{ "jsonrpc":"2.0", "method":"ListCommands", "id":2 }
{ "jsonrpc":"2.0", "method":"Execute",
  "params":{"name":"word.count","context":{...},"args":{}}, "id":3 }

// Plugin -> Editor (callback)
{ "jsonrpc":"2.0", "method":"ApplyEdits",
  "params":{"bufferId":"...","edits":[{"start":10,"end":20,"text":"X"}]} }
{ "jsonrpc":"2.0", "method":"PostMessage", "params":{"text":"Done."} }

Security & Sandboxing (initial stance)
	•	Default deny: plugins cannot read files or environment via the editor.
	•	Editor exposes only text operations and messages by default.
	•	Opt-in capabilities later (e.g., filesystem, spawn processes) via config flags.

Reference Plugin (for milestone M8)
	•	wordcount (read-only), then toggle-comment (mutating) for Go/JSON.

⸻

6) Keybindings (nano-like defaults)
	•	Ctrl+Q: Quit (prompt if dirty)
	•	Ctrl+S: Save
	•	Ctrl+O: Open file (prompt)
	•	Ctrl+W: Search (incremental)
	•	Ctrl+K: Cut line (kill)
	•	Ctrl+U: Paste (yank)
	•	Ctrl+Z / Ctrl+Y: Undo / Redo
	•	F2: Contextual Command Menu
(Remappable in config at M5.)

⸻

7) Config

File: ~/.<project>/config.yaml (or local .editorconfig.yaml in project root)

theme: "default"
keymap:
  "Ctrl+Q": "editor.quit"
  "Ctrl+S": "file.save"
  "F2":     "menu.context"
plugins:
  - path: "/usr/local/bin/<project>-wordcount"
    enabled: true
    capabilities: ["read"]


⸻

8) File Handling
	•	Default newline: \n (preserve existing when possible).
	•	UTF-8 only v1; detect BOM and drop on write unless present on read.
	•	Large files: start with in-memory; document practical limits; add piece table in M9.

⸻

9) Building Blocks & Decisions
	•	Terminal lib: tcell for input/render; fall back plan: termbox if needed.
	•	Text storage: Start gap buffer behind TextStorage; plan swap to piece table.
	•	Search: naive forward scan with highlights; optimize later with indexes.
	•	Undo/Redo: command log with coalescing for typed runs; cap history size.

⸻

10) Directory Structure (initial)

.
├── cmd/<project>/main.go
├── pkg/
│   ├── editor/
│   ├── buffer/
│   ├── view/
│   ├── input/
│   ├── render/
│   ├── files/
│   ├── search/
│   ├── history/
│   ├── config/
│   ├── commands/
│   └── plugins/
├── internal/app/
├── examples/plugins/wordcount/   # reference plugin
├── testdata/
└── README.md


⸻

11) Testing Strategy
	•	Unit tests for buffer ops, search, history, keymap resolution.
	•	Golden tests for rendering (string snapshots of screen buffer).
	•	Integration tests:
	•	Open/edit/save flow.
	•	Plugin handshake and command execution (spawn child process).
	•	Performance tests (M9): load & scroll large file; basic timing.

Run all:

go test ./... -run . -v


⸻

12) Developer Workflow (for agents)
	1.	Keep PRs small; each should close or advance a milestone task.
	2.	Write tests first for buffer and commands when practical.
	3.	Document public types and interfaces.
	4.	Log via a minimal logger behind a --debug flag; no noisy stdout in normal runs.
	5.	Add a note to CHANGELOG.md for user-visible changes.

⸻

13) Tasks Backlog
	•	M0: Init module, main.go, tcell setup, clean exit.
	•	M1: File open, viewport, cursor move, status bar.
	•	M2: Gap buffer TextStorage, edit ops, save.
	•	M3: Incremental search & highlight.
	•	M4: Kill/yank; undo/redo v1.
	•	M5: Config loader (YAML), keymap remap.
	•	M6: Multiple buffers; buffer switch.
	•	M7: Contextual command menu UI + registry.
	•	M8: Plugin manager (JSON-RPC stdio) + sample plugin.
	•	M9: Piece table option; perf tests.

⸻

14) Naming & Branding

Use <PROJECT_NAME> for now. Suggestions: germ (growing editor), gnano, minos. Rename cmd/<project> accordingly.

⸻

15) License

MIT (proposed).

⸻

16) Future Ideas
	•	Syntax highlighting (plugin-provided, e.g., tree-sitter via external process).
	•	LSP integration as a plugin (stdio), surfacing actions through the contextual menu.
	•	Macro recording/replay.
	•	Session persistence & MRU lists.
	•	Theming and status line customization.

⸻

Appendix A — Command Catalog (starting set)

Command ID	Title	Default Keys	When (predicate)
editor.quit	Quit	Ctrl+Q	always
file.open	Open File	Ctrl+O	always
file.save	Save	Ctrl+S	buffer.isDirty
edit.insertRune	Insert Character	(typed)	buffer.isEditable
edit.backspace	Backspace	Backspace	cursor.pos>0
edit.delete	Delete	Delete	not at EOF
edit.newline	Newline	Enter	buffer.isEditable
nav.left/right/up/down	Move	arrows	always
search.incremental	Search	Ctrl+W	always
edit.killLine	Cut Line	Ctrl+K	buffer.isEditable
edit.yank	Paste	Ctrl+U	killring.hasData
history.undo	Undo	Ctrl+Z	history.canUndo
history.redo	Redo	Ctrl+Y	history.canRedo
menu.context	Command Menu	F2	always


⸻

How an Agent Should Start
	1.	Implement M0 with tcell: create internal/app/runner.go to own lifecycle.
	2.	Create pkg/buffer/gapbuffer.go and pkg/buffer/storage.go with TextStorage.
	3.	Build minimal pkg/commands with editor.quit.
	4.	Add CI (GitHub Actions) to run go test ./....
	5.	Keep README updated as milestones complete.

⸻

End of README.
