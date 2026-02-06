package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseKeybinding(t *testing.T) {
	kb, err := ParseKeybinding("Ctrl+X")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	ev := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModCtrl)
	if !kb.Matches(ev) {
		t.Fatalf("expected match for Ctrl+X")
	}
}

func TestParseKeybinding_AltRune(t *testing.T) {
	kb, err := ParseKeybinding("Alt+X")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	ev := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt)
	if !kb.Matches(ev) {
		t.Fatalf("expected match for Alt+X")
	}
}

func TestParseKeybinding_FunctionKey(t *testing.T) {
	kb, err := ParseKeybinding("F2")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	ev := tcell.NewEventKey(tcell.KeyF2, 0, 0)
	if !kb.Matches(ev) {
		t.Fatalf("expected match for F2")
	}
}

func TestParseKeybinding_Invalid(t *testing.T) {
	if _, err := ParseKeybinding("Ctrl+"); err == nil {
		t.Fatalf("expected error for invalid keybinding")
	}
}

func TestLoadConfigRemap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := []byte("keymap:\n  quit: Ctrl+X\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ev := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModCtrl)
	if !cfg.Keymap["quit"].Matches(ev) {
		t.Fatalf("expected remapped quit to Ctrl+X")
	}
}

func TestLoadConfigRemap_MenuF2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := []byte("keymap:\n  menu: F2\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ev := tcell.NewEventKey(tcell.KeyF2, 0, 0)
	if !cfg.Keymap["menu"].Matches(ev) {
		t.Fatalf("expected remapped menu to F2")
	}
}
