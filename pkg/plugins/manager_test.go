package plugins

import "testing"

type dummyPlugin struct{ name string }

func (d dummyPlugin) Name() string { return d.name }

func TestManagerRegisterAndGet(t *testing.T) {
    m := NewManager()
    p := dummyPlugin{name: "dummy"}
    m.Register(p)
    got, ok := m.Get("dummy")
    if !ok {
        t.Fatalf("expected plugin to be registered")
    }
    if got.Name() != "dummy" {
        t.Fatalf("unexpected plugin name: %s", got.Name())
    }
}

