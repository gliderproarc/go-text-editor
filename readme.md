<TextEditorTest> — A Go CLI Text Editor

A tiny nano-like editor that can grow toward Emacs

Goal: Start with a reliable, minimalist, cross-platform terminal editor (nano clone), then expand toward Emacs-style power via a contextual command menu and a plug-in system.

Parallelization Plan
--------------------
To keep the editor responsive as features grow, we will split work between a
single sequential core and multiple background goroutines.

Sequential core (single goroutine)
- Maintains the edit log and undo/redo history in order.
- Owns cursor position and selection state.
- Applies all edits and command dispatch so changes occur deterministically.

Concurrent subsystems (worker goroutines)
- Rendering, file I/O, search/indexing, plug‑in host, and other background
  analysis.
- Workers communicate with the core through typed channels and operate on
  snapshots of editor state. They never mutate core state directly.

Integration guidelines
- Offload long‑running tasks to workers and report results back as commands.
- Keep data structures copy‑on‑write or immutable when shared across threads.
- Log every edit through the sequential core to preserve ordering for future
  replay or collaboration features.

This approach lets most of the application run in parallel while guaranteeing
that the edit history and cursor/selection remain consistent.

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

Example: Build & basic usage

Build the editor binary from the repository root:

    go build ./cmd/texteditor

This produces a binary named `texteditor` in the working directory. Run it with an optional file path:

    ./texteditor [path/to/file]

Interactive example (typical session):

- Open a file:

    ./texteditor README.md

 - Move the cursor with the arrow keys (or Ctrl+B/F/P/N), PageUp/PageDown, Home/End.
- Search (incremental): press Ctrl+W, type a query — matches are highlighted in the viewport as you type; press Enter to jump to the current match, Esc to cancel.
- Go to line: press Alt+G, enter a 1-based line number, press Enter to jump.
- Mnemonic menu: press Space in normal mode or Alt+M in insert mode to open a mnemonic key menu; press Space within this menu to switch to the everything menu.
- Save changes: press Ctrl+S.
- Quit: press Ctrl+Q (the editor will prompt if the buffer is dirty in future milestones).

Notes:
- Search highlights and go-to behavior are implemented in M3. Highlight styles use a high-contrast background for visibility; theming and color configuration come later (M5).

Language Support
----------------
Syntax highlighting is configured via a simple JSON file and supports both tree-sitter-backed and lightweight heuristic highlighters.

- Config file: `config/languages.json`
- Defaults: Go (tree-sitter) and Markdown (basic heuristics)
- Build tags:
  - With tree-sitter providers: `go build -tags tree_sitter ./cmd/texteditor`
  - Without tree-sitter: Markdown highlighting still works; other languages fall back to none.
- Tests with tag: `go test -tags tree_sitter ./...`
- CI/sandbox tip: force local Go caches if needed:

    GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache GOTMPDIR=$(pwd)/.gotmp go test -tags tree_sitter ./...

Config format

    {
      "languages": [
        { "id": "go", "name": "Go", "extensions": [".go"], "highlighter": "tree-sitter-go" },
        { "id": "markdown", "name": "Markdown", "extensions": [".md", ".markdown"], "highlighter": "markdown-basic" }
      ]
    }

How it works
- The editor detects the language by the file’s extension using `config/languages.json` (falls back to built-in defaults if missing/invalid).
- It instantiates the configured highlighter for that language.
- Syntax highlighting runs asynchronously in a background worker on the latest buffer snapshot; results are applied on the next frame if still current (coalesced by edit sequence).
- Highlight groups map to the theme in `internal/app/runner_draw.go`: `keyword`, `string`, `comment`, `number`, `function`, `type`.

Current coverage
- Go: keywords, strings, comments, numbers, and function/type names (via tree-sitter when built with `-tags tree_sitter`).
- Markdown: headings, code fences and inline code, blockquotes, list markers, links, and emphasis (heuristic, always available).

Add a new language
1) Tree-sitter-backed
- Add the corresponding Go binding (e.g., `github.com/smacker/go-tree-sitter/rust`).
- Implement a highlighter plugin similar to `pkg/plugins/treesitter.go` for that grammar, or adapt it for the new language.
- Register it in `pkg/plugins/language_provider_ts.go` by returning the new plugin for a unique `highlighter` id.
- Add an entry in `config/languages.json` with the file extensions and that `highlighter` id.
- Build with `-tags tree_sitter` to enable it.

2) Heuristic/regex-backed
- Create `pkg/plugins/<lang>.go` implementing:
  - `type <Name>Highlighter struct{}`
  - `func (h *<Name>Highlighter) Name() string`
  - `func (h *<Name>Highlighter) Highlight(src []byte) []search.Range`
- Add a case to `pkg/plugins/language_provider_notts.go` (and to the tree-sitter provider if you want it available in both builds) to return the new highlighter for a unique `highlighter` id.
- Add the language entry to `config/languages.json` with its extensions and `highlighter` id.

Notes
- The config is read from `config/languages.json` relative to the working directory. If the file is missing or malformed, built-in defaults are used.
- Markdown highlighter aligns with existing groups so theme colors apply automatically.
- Since syntax computation is async, extremely large files may show base UI first and then highlights pop in shortly after.

Timeouts
- `TEXTEDITOR_SYNTAX_TIMEOUT_MS` is reserved for future cancellation support. Currently syntax computation is async but not cancelable; slow results are safely dropped if the buffer changes before they finish.
- You can prototype a new language by only adding a heuristic highlighter; move to tree-sitter later for accuracy.

Spell Checking (IPC Prototype)
------------------------------
- The editor can pipe whitespace-separated words to an external spell checker process and highlight words that need checking in the viewport.
- Protocol: one request line contains space-separated words; one response line contains the subset of words to highlight (also space-separated). Words are treated case-insensitively.
- Toggle via command menu: open the command palette (Space in normal mode or Alt+M in insert), search for "spell: toggle".
  - Default: uses `./aspellbridge` which wraps `aspell -a`. If `aspellbridge` fails, it falls back to `./spellmock`.
  - Override with environment: set `TEXTEDITOR_SPELL=/path/to/custom-checker` to force a specific command.
- Recheck the viewport at any time with the command "spell: recheck"; otherwise, checks trigger when the viewport changes.
- Highlight color is configurable in the theme as `highlight.spell.bg` and `highlight.spell.fg`.

Note on async behavior
- The "spell: check word" (word-at-point) command runs a one-shot check with a short timeout so it won’t block the UI if the external checker stalls. Configure the timeout via `TEXTEDITOR_SPELL_TIMEOUT_MS` (default: 500ms).
- Viewport spell checking (background highlighting) now also uses a timeout-aware request to the persistent spell client. If the checker stalls beyond the timeout, the client is terminated and will be restarted on demand; the UI remains responsive and simply clears spell highlights for that frame.

aspell bridge

- Build: `go build -o aspellbridge ./cmd/aspellbridge`
- Requires `aspell` to be installed and on `PATH`.
- Language can be set with `TEXTEDITOR_SPELL_LANG` (e.g., `en_US`); defaults to aspell's default language if unset.

Mock checker for local testing
- Build the mock: `go build -o spellmock ./cmd/spellmock`
- It flags words containing digits, ALL-CAPS (>=3 letters), or very long words (>14 letters) to demonstrate the integration.

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
		•	Ctrl+K: cut to end of line to an internal kill ring (single slot is fine for v1).
		•	Ctrl+U/Ctrl+Y: paste line.
		•	p: paste from the kill ring in normal mode.
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
		•	Ctrl+K: Cut to end of line
		•	Ctrl+U/Ctrl+Y: Paste (yank)
		•	Ctrl+Z / Ctrl+Y: Undo / Redo
		•	Ctrl+A/Ctrl+E: Line start/end (insert)
		•	F2: Contextual Command Menu
(Remappable in config at M5.)

⸻

7) Config

File: ~/.<project>/config.yaml (or local .editorconfig.yaml in project root)

theme:
  preset: "light" # or "dark" or "terminal"; omit to use defaults (terminal)
  # Optional overrides (any of these keys)
  # ui.background: black
  # ui.foreground: white
  # status.bg: white
  # status.fg: black
  # mini.bg: white
  # mini.fg: black
  # cursor.text: black
  # cursor.insert.bg: blue
  # cursor.normal.bg: green
  # text.default: white
  # highlight.search.bg: yellow
  # highlight.search.fg: black
  # highlight.search.current.bg: blue
  # highlight.search.current.fg: white
  # syntax.keyword: red
  # syntax.string: green
  # syntax.comment: gray
  # syntax.number: yellow
  # syntax.type: blue
  # syntax.function: blue

Using Base16 or Alacritty themes
Terminal theme (follow terminal palette)
- Use the built-in terminal-compliant theme to piggy-back on your terminal's colors. It avoids hard-coded RGB values and relies on the terminal's default fg/bg and standard ANSI palette for UI and syntax.
  theme:
    preset: "terminal"

Defaults
- If you omit the theme block entirely, the editor loads with the terminal-compliant theme by default.

- Base16: point to any Base16 YAML (keys base00..base0F) and optionally override:
  theme:
    file: "/absolute/path/to/base16-scheme.yaml"
    # overrides here are applied after import

- Alacritty: reference your Alacritty colors file (YAML keys supported):
  theme:
    file: "/absolute/path/to/alacritty/colors.yml"

Bundled examples
- Base16 presets included under `config/themes/base16/`:
  - `base16-default-dark.yaml`
  - `base16-default-light.yaml`
  - `base16-solarized-dark.yaml`
  - `base16-gruvbox-dark.yaml`

Relative `theme.file` paths are resolved against the directory of your config file.
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
	•	M10: Smooth scrolling & viewport engine.
	•	M11: Syntax highlighting via tree-sitter plugin.

⸻

14) Naming & Branding

Use <PROJECT_NAME> for now. Suggestions: germ (growing editor), gnano, minos. Rename cmd/<project> accordingly.

⸻

15) License

MIT (proposed).

⸻

16) Future Ideas
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
edit.killLine   Cut to End of Line  Ctrl+K  buffer.isEditable
edit.yank       Paste   Ctrl+U/Ctrl+Y  killring.hasData
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

Status & Next Milestones (updated)

This project has moved beyond the early scaffolding and through the first interactive features. Here’s a quick snapshot of what’s done and what’s next.

Completed (high level)
- M0: Scaffold and event loop: terminal init, clean shutdown, basic UI.
- M1: Open/display and navigation: file load, cursor movement, status UI.
- M2: Editing core: gap buffer, insert/delete/newline, save + Save As.
- M3: Search and go-to: incremental highlights, Enter to jump; Alt+G go-to.
- M4: Kill/yank and undo/redo v1: Ctrl+K, Ctrl+U/Ctrl+Y, Ctrl+Z/Ctrl+Y; single-slot kill ring.
- Extras: Help screen (F1/Ctrl+H), dirty-quit confirmation, basic normal/insert/visual modes, logging hooks.
- M5: Config and keymaps: load ~/.texteditor/config.yaml to remap quit/save/search.
- M6: Multiple buffers with open-in-new-buffer and buffer switching.
- M7: Contextual command menu with fuzzy filtering (F2) and command execution.

Next Milestones (proposal)
- M8 — Plugin API v0
  - Tasks: JSON-RPC stdio handshake, command registration, example wordcount plugin.
  - Files: pkg/plugins/*, internal/app/runner.go (plugin host), tests for handshake and command execution.
  - Acceptance: editor discovers and executes external plugin commands.

Upcoming (after M8)
- M9 — Syntax Highlighting: integrate tree-sitter via plugin; render highlighted tokens.
- M10 — Quality & Performance: piece table option behind TextStorage, large-file tests and thresholds, profiling.
- M11 — Smooth Scrolling & Viewport Engine: adjust viewport offset when cursor moves and add tests for edge cases.

Notes
- The existing UI already includes open/save prompts, search/go-to, and quit confirmation; the plan above focuses on configurability and multi-buffer ergonomics next.
- Consider consolidating duplicated helpers (drawUI/drawHelp) found in cmd/texteditor and internal/app/runner.go during M5 refactors.
