package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// FilterAppliedMsg is sent when the user applies a filter
type FilterAppliedMsg struct {
	Pattern string
}

// FilterModel manages the filter input modal
type FilterModel struct {
	textinput.Model
	visible bool
}

// NewFilterModel creates a new filter input model
func NewFilterModel() FilterModel {
	ti := textinput.New()
	ti.Placeholder = "Filter files..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return FilterModel{
		Model:   ti,
		visible: false,
	}
}

// Show makes the filter modal visible
func (m *FilterModel) Show() {
	m.visible = true
	m.Focus()
}

// Hide hides the filter modal
func (m *FilterModel) Hide() {
	m.visible = false
	m.Blur()
	m.SetValue("")
}

// IsVisible returns whether the filter modal is visible
func (m *FilterModel) IsVisible() bool {
	return m.visible
}

// Update handles messages for the filter model
func (m FilterModel) Update(msg tea.Msg) (FilterModel, tea.Cmd) {
	var cmd tea.Cmd

	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Apply filter and hide
			pattern := m.Value()
			m.Hide()
			return m, func() tea.Msg {
				return FilterAppliedMsg{Pattern: pattern}
			}
		case "esc":
			m.Hide()
			return m, nil
		}
	}

	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

// View renders the filter modal
func (m FilterModel) View(width, height int) string {
	if !m.visible {
		return ""
	}

	// Calculate modal dimensions
	modalWidth := min(width-20, 80)
	modalHeight := 7

	// Create modal style with shadow effect
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(v2styles.PrimaryColor).
		Background(lipgloss.Color("#1C1C1E")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Bold(true)

	// Create content
	title := v2styles.LayerHeaderStyle.Render("🔍 Filter Files")
	input := m.Model.View()
	help := v2styles.LayerValueStyle.Render("ESC: close | ↑↓: navigate | ENTER: apply")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", input, "", help)

	// Render modal
	modal := modalStyle.Render(content)

	// Place modal in center with base content underneath
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
