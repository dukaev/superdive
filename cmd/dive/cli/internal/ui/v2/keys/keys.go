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
	CollapseAll      key.Binding
	ExpandAll        key.Binding
	ToggleAdded      key.Binding
	ToggleRemoved    key.Binding
	ToggleModified   key.Binding
	ToggleUnmodified key.Binding
}

// ShortHelp returns keys that are always visible
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Filter, k.ToggleView}
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
	Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "scroll left")),
	Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "scroll right")),
	Tab:        key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "switch pane")),
	Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
	Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Space:      key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle folder")),
	ToggleView: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "toggle view")),

	// Tree control bindings
	CollapseAll:    key.NewBinding(key.WithKeys("C"), key.WithHelp("shift+c", "collapse all")),
	ExpandAll:      key.NewBinding(key.WithKeys("O"), key.WithHelp("shift+o", "expand all")),
	ToggleAdded:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "added")),
	ToggleRemoved:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "removed")),
	ToggleModified: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "modified")),
}
