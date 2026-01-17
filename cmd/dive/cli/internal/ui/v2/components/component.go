package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Component is a base interface for all UI components
type Component interface {
	// Update handles messages and returns the updated component
	Update(msg tea.Msg) (Component, tea.Cmd)

	// View renders the component
	View() string
}

// BaseComponent provides common functionality for all components
type BaseComponent struct {
	width   int
	height  int
	focused bool
}

// NewBaseComponent creates a new base component
func NewBaseComponent() BaseComponent {
	return BaseComponent{
		width:   0,
		height:  0,
		focused: false,
	}
}

// SetSize sets the component size
func (c *BaseComponent) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Size returns the component size
func (c *BaseComponent) Size() (int, int) {
	return c.width, c.height
}

// Focus focuses the component
func (c *BaseComponent) Focus() {
	c.focused = true
}

// Blur removes focus from the component
func (c *BaseComponent) Blur() {
	c.focused = false
}

// IsFocused returns whether the component is focused
func (c *BaseComponent) IsFocused() bool {
	return c.focused
}
