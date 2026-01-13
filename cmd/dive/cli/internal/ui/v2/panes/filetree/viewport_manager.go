package filetree

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
)

// ViewportManager wraps bubbletea viewport with typed methods
type ViewportManager struct {
	viewport viewport.Model
}

// NewViewportManager creates a new viewport manager with the given dimensions
func NewViewportManager(width, height int) *ViewportManager {
	vp := viewport.New(width, height)
	return &ViewportManager{
		viewport: vp,
	}
}

// SetSize updates the viewport dimensions
func (v *ViewportManager) SetSize(width, height int) {
	v.viewport.Width = width
	v.viewport.Height = height
}

// SetContent updates the viewport content
func (v *ViewportManager) SetContent(content string) {
	v.viewport.SetContent(content)
}

// GetViewport returns the underlying viewport model
func (v *ViewportManager) GetViewport() *viewport.Model {
	return &v.viewport
}

// GotoTop scrolls to the top of the viewport
func (v *ViewportManager) GotoTop() {
	v.viewport.GotoTop()
}

// GotoBottom scrolls to the bottom of the viewport
func (v *ViewportManager) GotoBottom() {
	v.viewport.GotoBottom()
}

// SetYOffset sets the vertical scroll offset
func (v *ViewportManager) SetYOffset(offset int) {
	v.viewport.SetYOffset(offset)
}

// GetYOffset returns the current vertical scroll offset
func (v *ViewportManager) GetYOffset() int {
	return v.viewport.YOffset
}

// GetHeight returns the viewport height
func (v *ViewportManager) GetHeight() int {
	return v.viewport.Height
}

// Update passes a message to the underlying viewport
func (v *ViewportManager) Update(msg tea.Msg) (viewport.Model, tea.Cmd) {
	return v.viewport.Update(msg)
}
