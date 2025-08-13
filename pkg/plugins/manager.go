package plugins

// Plugin represents a generic extension component.
type Plugin interface {
    Name() string
}

// SyntaxNode is the minimal node API our editor needs.
type SyntaxNode interface {
    Type() string
}

// SyntaxTree is the minimal tree API our editor needs.
type SyntaxTree interface {
    RootNode() SyntaxNode
}

// SyntaxPlugin provides parsing capabilities without tying to a concrete lib.
type SyntaxPlugin interface {
    Plugin
    Parse(src []byte) SyntaxTree
}

// Manager keeps track of registered plug-ins.
type Manager struct {
    registry map[string]Plugin
}

// NewManager creates an empty plug-in registry.
func NewManager() *Manager {
    return &Manager{registry: make(map[string]Plugin)}
}

// Register adds a plug-in to the registry.
func (m *Manager) Register(p Plugin) {
    m.registry[p.Name()] = p
}

// Get retrieves a plug-in by name.
func (m *Manager) Get(name string) (Plugin, bool) {
    p, ok := m.registry[name]
    return p, ok
}

// List returns all registered plug-ins.
func (m *Manager) List() []Plugin {
    out := make([]Plugin, 0, len(m.registry))
    for _, p := range m.registry {
        out = append(out, p)
    }
    return out
}
