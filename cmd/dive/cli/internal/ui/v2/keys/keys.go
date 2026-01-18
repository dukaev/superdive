// Package keys defines key bindings for the UI
package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Tab        key.Binding
	Filter     key.Binding
	Quit       key.Binding
	Enter      key.Binding
	Esc        key.Binding
	Space      key.Binding
	ToggleView key.Binding

	// Tree Control Keys
	CollapseAll        key.Binding
	ExpandAll          key.Binding
	ToggleAdded        key.Binding
	ToggleRemoved      key.Binding
	ToggleModified     key.Binding
	ToggleUnmodified   key.Binding

	// Layer Navigation Keys
	PrevLayer key.Binding
	NextLayer key.Binding
	LayerNav  key.Binding // Combined layer navigation binding
}

// ShortHelp returns keys that are always visible
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.LayerNav, k.Enter, k.Filter, k.ToggleView}
}

// FullHelp returns all keys (for extended help)
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},               // Navigation
		{k.Enter, k.Space},           // Actions
		{k.CollapseAll, k.ExpandAll}, // Tree folding
		{k.ToggleAdded, k.ToggleRemoved, k.ToggleModified, k.ToggleUnmodified}, // Filters
		{k.Tab, k.Filter, k.ToggleView, k.Quit},                                // System
	}
}

// Keys holds all key bindings for the UI
var Keys = KeyMap{
	Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	Tab:        key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "pane")),
	Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
	Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Space:      key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
	ToggleView: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "view")),

	// Tree control bindings
	CollapseAll:      key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "collapse")),
	ExpandAll:        key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "expand")),
	ToggleAdded:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "added")),
	ToggleRemoved:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "removed")),
	ToggleModified:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "modified")),
	ToggleUnmodified: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unmodified")),

	// Layer navigation bindings
	PrevLayer: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev")),
	NextLayer: key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next")),
	LayerNav:  key.NewBinding(key.WithKeys("[", "]"), key.WithHelp("[,]", "layers")),
}
