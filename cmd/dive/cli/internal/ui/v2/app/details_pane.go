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
		// Truncate to fit pane width (leave space for "Digest: " prefix)
		maxDigestWidth := m.width - 10
		if maxDigestWidth < 20 {
			maxDigestWidth = 20
		}
		if lipgloss.Width(digest) > maxDigestWidth {
			digest = runewidth.Truncate(digest, maxDigestWidth, "...")
		}
		if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Digest: %s", digest))) {
			goto finish
		}
	}

	// Command - Maximum 2 lines!
	if !addLine(v2styles.LayerHeaderStyle.Render("Command:")) {
		goto finish
	}

	if layer.Command == "" {
		addLine(v2styles.LayerValueStyle.Render("(unavailable)"))
	} else {
		maxWidth := m.width - 4
		if maxWidth < 10 {
			maxWidth = 10
		}

		// Wrap command to fit width
		wrappedCmd := lipgloss.NewStyle().Width(maxWidth).Render(layer.Command)
		cmdLines := strings.Split(wrappedCmd, "\n")

		// Show max 2 lines: first line + last line (with "..." prefix if long)
		if len(cmdLines) == 1 {
			// Short command - fits in 1 line
			addLine(v2styles.LayerValueStyle.Render(cmdLines[0]))
		} else if len(cmdLines) == 2 {
			// Exactly 2 lines - show both
			addLine(v2styles.LayerValueStyle.Render(cmdLines[0]))
			addLine(v2styles.LayerValueStyle.Render(cmdLines[1]))
		} else {
			// Long command (>2 lines) - show first and last
			addLine(v2styles.LayerValueStyle.Render(cmdLines[0]))

			// Last line with "..." prefix
			lastLine := cmdLines[len(cmdLines)-1]
			secondLine := "..." + lastLine

			// Truncate if still too long
			if lipgloss.Width(secondLine) > maxWidth {
				secondLine = runewidth.Truncate(secondLine, maxWidth, "...")
			}

			addLine(v2styles.LayerValueStyle.Render(secondLine))
		}
	}

finish:
	return strings.Join(lines, "\n")
}
