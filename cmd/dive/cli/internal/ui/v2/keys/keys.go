package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Tab       key.Binding
	Filter    key.Binding
	Quit      key.Binding
	Enter     key.Binding
	Esc       key.Binding
	Space     key.Binding
	ToggleView key.Binding
}

// ShortHelp returns keys that are always visible
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Up, k.Down, k.Enter, k.Filter, k.ToggleView, k.Quit}
}

// FullHelp returns all keys (for extended help)
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},             // Navigation
		{k.Enter, k.Space},         // Actions
		{k.Tab, k.Filter, k.ToggleView, k.Quit}, // System
	}
}

var Keys = KeyMap{
	Up:        key.NewBinding(key.WithKeys("up", "k", "["), key.WithHelp("↑/k/[", "navigate up")),
	Down:      key.NewBinding(key.WithKeys("down", "j", "]"), key.WithHelp("↓/j/]", "navigate down")),
	Left:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "scroll left")),
	Right:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "scroll right")),
	Tab:       key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "switch pane")),
	Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
	Enter:     key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter/spc", "toggle folder")),
	Esc:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Space:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle folder")),
	ToggleView: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "toggle view")),
}
