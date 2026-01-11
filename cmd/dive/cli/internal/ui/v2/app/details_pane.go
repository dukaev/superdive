package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/dive/image"
	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// DetailsPane displays information about a single layer
type DetailsPane struct {
	focused bool
	width   int
	height  int
	layer   *image.Layer
}

// NewDetailsPane creates a new details pane
func NewDetailsPane() DetailsPane {
	return DetailsPane{
		width:  80,
		height: 10,
	}
}

// SetSize updates the pane dimensions
func (m *DetailsPane) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetLayer updates the layer to display
func (m *DetailsPane) SetLayer(layer *image.Layer) {
	m.layer = layer
}

// Focus sets the pane as active
func (m *DetailsPane) Focus() {
	m.focused = true
}

// Blur sets the pane as inactive
func (m *DetailsPane) Blur() {
	m.focused = false
}

// IsFocused returns true if the pane is focused
func (m *DetailsPane) IsFocused() bool {
	return m.focused
}

// Init initializes the pane
func (m DetailsPane) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m DetailsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Details pane doesn't handle any messages - it's read-only
	return m, nil
}

// View renders the pane
func (m DetailsPane) View() string {
	content := m.renderContent()
	return v2styles.RenderBox("Layer Details", m.width, m.height, content, m.focused)
}

// renderContent generates the details content
func (m DetailsPane) renderContent() string {
	// Calculate available space: Height - Borders(2) - Header(2)
	maxLines := m.height - 4
	if maxLines < 0 {
		maxLines = 0
	}

	if m.layer == nil {
		return "No details"
	}

	layer := m.layer
	var lines []string

	// Helper: add line only if space available
	addLine := func(s string) bool {
		if len(lines) < maxLines {
			lines = append(lines, s)
			return true
		}
		return false
	}

	// Tags
	if len(layer.Names) > 0 {
		tags := strings.Join(layer.Names, ", ")
		if lipgloss.Width(tags) > m.width-8 {
			tags = runewidth.Truncate(tags, m.width-8, "...")
		}
		if !addLine(v2styles.LayerHeaderStyle.Render(fmt.Sprintf("Tags: %s", tags))) {
			goto finish
		}
	}

	// ID & Size
	if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Id: %s", layer.Id))) {
		goto finish
	}
	if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Size: %s", formatSize(layer.Size)))) {
		goto finish
	}

	// Digest
	if layer.Digest != "" {
		digest := layer.Digest
		if len(digest) > 71 {
			digest = digest[:71] + "..."
		}
		if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Digest: %s", digest))) {
			goto finish
		}
	}

	// Spacer
	if !addLine("") {
		goto finish
	}

	// Command
	if layer.Command != "" {
		if !addLine(v2styles.LayerHeaderStyle.Render("Command:")) {
			goto finish
		}

		maxWidth := m.width - 4
		if maxWidth < 20 {
			maxWidth = 20
		}

		// Wrap command to fit width
		wrappedCmd := lipgloss.NewStyle().Width(maxWidth).Render(layer.Command)
		wrappedLines := strings.Split(wrappedCmd, "\n")

		// Add command lines while space remains
		remainingLines := maxLines - len(lines)
		if remainingLines > 0 {
			for i, line := range wrappedLines {
				if i >= remainingLines {
					// No space left, replace last line with "..."
					lines[len(lines)-1] = v2styles.LayerValueStyle.Render("...")
					break
				}
				addLine(v2styles.LayerValueStyle.Render(line))
			}
		}
	} else {
		addLine(v2styles.LayerHeaderStyle.Render("Command:"))
		addLine(v2styles.LayerValueStyle.Render("(unavailable)"))
	}

finish:
	return strings.Join(lines, "\n")
}
