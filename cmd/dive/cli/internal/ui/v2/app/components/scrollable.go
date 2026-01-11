package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
)

// ScrollablePane wraps a viewport with common functionality
type ScrollablePane struct {
	viewport viewport.Model
	title    string
	content  string
}

// NewScrollablePane creates a new scrollable pane
func NewScrollablePane(title string, width, height int) ScrollablePane {
	return ScrollablePane{
		viewport: viewport.New(width, height),
		title:    title,
	}
}

// SetSize updates the pane dimensions
func (s *ScrollablePane) SetSize(width, height int) {
	s.viewport.Width = width
	s.viewport.Height = height
}

// SetContent updates the pane content and resets scroll to top
func (s *ScrollablePane) SetContent(content string) {
	s.content = content
	s.viewport.SetContent(content)
	s.viewport.GotoTop()
}

// Update processes messages for the viewport
func (s *ScrollablePane) Update(msg tea.Msg) (ScrollablePane, tea.Cmd) {
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return *s, cmd
}

// View returns the rendered viewport content
func (s *ScrollablePane) View() string {
	return s.viewport.View()
}

// LineUp scrolls up by n lines
func (s *ScrollablePane) LineUp(n int) {
	s.viewport.LineUp(n)
}

// LineDown scrolls down by n lines
func (s *ScrollablePane) LineDown(n int) {
	s.viewport.LineDown(n)
}

// GotoTop scrolls to the top
func (s *ScrollablePane) GotoTop() {
	s.viewport.GotoTop()
}

// GetViewport returns the underlying viewport model for direct access
func (s *ScrollablePane) GetViewport() *viewport.Model {
	return &s.viewport
}

// GetContent returns the current content
func (s *ScrollablePane) GetContent() string {
	return s.content
}
