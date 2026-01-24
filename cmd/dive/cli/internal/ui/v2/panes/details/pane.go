// Package details provides the file details pane for displaying information about selected files.
package details

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
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
	focused  bool // Set by parent via FocusStateMsg, not by Focus()/Blur() methods
	width    int
	height   int
	layer    *image.Layer
	viewport viewport.Model
}

// New creates a new details pane
func New() Pane {
	vp := viewport.New(80, 10)
	return Pane{
		width:    80,
		height:   10,
		viewport: vp,
	}
}

// Resize updates the pane dimensions
func (m *Pane) Resize(width, height int) {
	m.width = width
	m.height = height

	// Calculate available height for the viewport content
	// Layout Padding: 2 (Top Border) + 2 (Bottom Border/Title gap) = 4
	viewportWidth := width - 2
	viewportHeight := height - layout.BoxContentPadding
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Width = viewportWidth
	m.viewport.Height = viewportHeight

	// Regenerate content with new width
	m.updateContent()
}

// SetLayer updates the layer to display
func (m *Pane) SetLayer(layer *image.Layer) {
	m.layer = layer
	m.updateContent()
}

// updateContent regenerates the viewport content
func (m *Pane) updateContent() {
	content := m.generateContent()
	m.viewport.SetContent(content)
	m.viewport.GotoTop()
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

	case common.LocalMouseMsg:
		// Handle mouse wheel for scrolling
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.viewport.ScrollUp(1)
			case tea.MouseButtonWheelDown:
				m.viewport.ScrollDown(1)
			}
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the pane
func (m Pane) View() string {
	content := m.viewport.View()
	return styles.RenderBox("Layer Details", m.width, m.height, content, m.focused)
}

// generateContent generates the full details content without truncation
func (m Pane) generateContent() string { //nolint:funlen
	if m.layer == nil {
		return "No details"
	}

	layer := m.layer
	var lines []string

	// Tags
	if len(layer.Names) > 0 {
		tags := strings.Join(layer.Names, ", ")
		if lipgloss.Width(tags) > m.width-8 {
			tags = runewidth.Truncate(tags, m.width-8, "...")
		}
		label := styles.LayerHeaderStyle.Render("Tags:")
		value := styles.LayerValueStyle.Render(tags)
		lines = append(lines, label+" "+value)
	}

	// ID & Size
	idLabel := styles.LayerHeaderStyle.Render("Id:")
	idValue := styles.LayerValueStyle.Render(layer.Id)
	lines = append(lines, idLabel+" "+idValue)

	sizeLabel := styles.LayerHeaderStyle.Render("Size:")
	sizeValue := styles.LayerValueStyle.Render(utils.FormatSize(layer.Size))
	lines = append(lines, sizeLabel+" "+sizeValue)

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
		digestLabel := styles.LayerHeaderStyle.Render("Digest:")
		digestValue := styles.LayerValueStyle.Render(digest)
		lines = append(lines, digestLabel+" "+digestValue)
	}

	// Command
	if layer.Command == "" {
		cmdLabel := styles.LayerHeaderStyle.Render("Command:")
		cmdValue := styles.LayerValueStyle.Render("(unavailable)")
		lines = append(lines, cmdLabel+" "+cmdValue)
	} else {
		// Add header
		cmdLabel := styles.LayerHeaderStyle.Render("Command:")
		lines = append(lines, cmdLabel)

		maxWidth := m.width - 4
		if maxWidth < 10 {
			maxWidth = 10
		}

		// Wrap command to fit width - show ALL lines
		wrappedCmd := lipgloss.NewStyle().Width(maxWidth).Render(layer.Command)
		cmdLines := strings.Split(wrappedCmd, "\n")

		for _, cmdLine := range cmdLines {
			lines = append(lines, styles.LayerValueStyle.Render(cmdLine))
		}
	}

	return strings.Join(lines, "\n")
}

// ShortHelp returns key bindings specific to the details pane.
// Details pane is read-only, so it has no specific keys.
func (m *Pane) ShortHelp() []key.Binding {
	return nil
}
