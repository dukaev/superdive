package common

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Pane defines the interface that all UI panes must implement.
// This interface enables polymorphic interaction between the main app model
// and child panes, eliminating type assertions and tight coupling.
type Pane interface {
	// Init initializes the pane component
	Init() tea.Cmd

	// Update handles incoming messages and returns the updated pane.
	// Returns Pane (not tea.Model) to enable polymorphic updates without type assertions.
	Update(msg tea.Msg) (Pane, tea.Cmd)

	// View renders the pane to a string
	View() string

	// Resize updates the pane dimensions. Called when the terminal is resized
	// or when the layout engine recalculates pane sizes.
	Resize(width, height int)

	// SetFocused sets the focus state of the pane.
	// When focused, the pane handles keyboard input.
	// When unfocused, the pane ignores keyboard input.
	SetFocused(focused bool)

	// ShortHelp returns key bindings specific to this pane.
	// These are displayed in the status bar when the pane is focused.
	// Returns nil or empty slice if the pane has no specific keys.
	ShortHelp() []key.Binding
}
