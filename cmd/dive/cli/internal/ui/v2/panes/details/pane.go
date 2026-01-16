package details

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/image"
)

// FocusStateMsg is sent by parent to tell the pane whether it's focused or not
type FocusStateMsg struct {
	Focused bool
}

// Pane displays information about a single layer
type Pane struct {
	focused bool // Set by parent via FocusStateMsg, not by Focus()/Blur() methods
	width   int
	height  int
	layer   *image.Layer
}

// New creates a new details pane
func New() Pane {
	return Pane{
		width:  80,
		height: 10,
	}
}

// Resize updates the pane dimensions
func (m *Pane) Resize(width, height int) {
	m.width = width
	m.height = height
}

// SetLayer updates the layer to display
func (m *Pane) SetLayer(layer *image.Layer) {
	m.layer = layer
}

// Init initializes the pane
func (m *Pane) Init() tea.Cmd {
	return nil
}

// SetFocused sets the focus state of the pane
func (m *Pane) SetFocused(focused bool) {
	m.focused = focused
}

// Update handles messages
func (m *Pane) Update(msg tea.Msg) (common.Pane, tea.Cmd) {
	switch msg := msg.(type) {
	case common.LayoutMsg:
		// Parent sends layout info instead of calling Resize()
		// Extract what we need from the message
		m.Resize(msg.LeftWidth, msg.DetailsHeight)
		return m, nil

	case common.LayerSelectedMsg:
		// Parent sends layer selection via message instead of calling SetLayer()
		m.SetLayer(msg.Layer)
		return m, nil

	case FocusStateMsg:
		// Parent controls focus state - use SetFocused method
		m.SetFocused(msg.Focused)
		return m, nil
	}
	// Details pane doesn't handle any other messages - it's read-only
	return m, nil
}

// View renders the pane
func (m Pane) View() string {
	content := m.renderContent()
	return styles.RenderBox("Layer Details", m.width, m.height, content, m.focused)
}

// renderContent generates the details content
func (m Pane) renderContent() string {
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
		if !addLine(styles.LayerHeaderStyle.Render(fmt.Sprintf("Tags: %s", tags))) {
			goto finish
		}
	}

	// ID & Size
	if !addLine(styles.LayerValueStyle.Render(fmt.Sprintf("Id: %s", layer.Id))) {
		goto finish
	}
	if !addLine(styles.LayerValueStyle.Render(fmt.Sprintf("Size: %s", utils.FormatSize(layer.Size)))) {
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
		if !addLine(styles.LayerValueStyle.Render(fmt.Sprintf("Digest: %s", digest))) {
			goto finish
		}
	}

	// Command - Maximum 2 lines!
	if !addLine(styles.LayerHeaderStyle.Render("Command:")) {
		goto finish
	}

	if layer.Command == "" {
		addLine(styles.LayerValueStyle.Render("(unavailable)"))
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
			addLine(styles.LayerValueStyle.Render(cmdLines[0]))
		} else if len(cmdLines) == 2 {
			// Exactly 2 lines - show both
			addLine(styles.LayerValueStyle.Render(cmdLines[0]))
			addLine(styles.LayerValueStyle.Render(cmdLines[1]))
		} else {
			// Long command (>2 lines) - show first and last
			addLine(styles.LayerValueStyle.Render(cmdLines[0]))

			// Last line with "..." prefix
			lastLine := cmdLines[len(cmdLines)-1]
			secondLine := "..." + lastLine

			// Truncate if still too long
			if lipgloss.Width(secondLine) > maxWidth {
				secondLine = runewidth.Truncate(secondLine, maxWidth, "...")
			}

			addLine(styles.LayerValueStyle.Render(secondLine))
		}
	}

finish:
	return strings.Join(lines, "\n")
}

// ShortHelp returns key bindings specific to the details pane.
// Details pane is read-only, so it has no specific keys.
func (p *Pane) ShortHelp() []key.Binding {
	return nil
}
