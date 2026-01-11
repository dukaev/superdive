package image

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/image"
)

// Pane displays image-level statistics and inefficiencies
type Pane struct {
	focused  bool
	width    int
	height   int
	analysis *image.Analysis
	viewport viewport.Model
}

// New creates a new image pane
func New(analysis *image.Analysis) Pane {
	vp := viewport.New(80, 20)
	p := Pane{
		analysis: analysis,
		viewport: vp,
		width:    80,
		height:   20,
	}
	// IMPORTANT: Generate content immediately so viewport is not empty on startup
	p.updateContent()
	return p
}

// SetSize updates the pane dimensions
func (m *Pane) SetSize(width, height int) {
	m.width = width
	m.height = height

	viewportWidth := width - 2
	viewportHeight := height - layout.BoxContentPadding
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Width = viewportWidth
	m.viewport.Height = viewportHeight

	// CRITICAL: Regenerate content with new width for proper truncation
	m.updateContent()
}

// SetAnalysis updates the analysis data
func (m *Pane) SetAnalysis(analysis *image.Analysis) {
	m.analysis = analysis
	m.updateContent()
}

// Focus sets the pane as active
func (m *Pane) Focus() {
	m.focused = true
}

// Blur sets the pane as inactive
func (m *Pane) Blur() {
	m.focused = false
}

// IsFocused returns true if the pane is focused
func (m *Pane) IsFocused() bool {
	return m.focused
}

// Init initializes the pane
func (m Pane) Init() tea.Cmd {
	m.updateContent()
	return nil
}

// Update handles messages
func (m Pane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		// Handle scrolling
		switch msg.String() {
		case "up", "k":
			m.viewport.LineUp(1)
		case "down", "j":
			m.viewport.LineDown(1)
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			if msg.Button == tea.MouseButtonWheelUp {
				m.viewport.LineUp(1)
			} else if msg.Button == tea.MouseButtonWheelDown {
				m.viewport.LineDown(1)
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the pane
func (m Pane) View() string {
	content := m.viewport.View()
	return styles.RenderBox("Image Details", m.width, m.height, content, m.focused)
}

// updateContent regenerates the viewport content
func (m *Pane) updateContent() {
	if m.analysis == nil {
		m.viewport.SetContent("No image data")
		return
	}

	content := m.generateContent()
	m.viewport.SetContent(content)
}

// generateContent creates the image statistics content
func (m *Pane) generateContent() string {
	width := m.width - 2 // Subtract borders

	// Header with statistics
	headerText := fmt.Sprintf(
		"Image name: %s\nTotal Image size: %s\nPotential wasted space: %s\nImage efficiency score: %.0f%%",
		m.analysis.Image,
		utils.FormatSize(m.analysis.SizeBytes),
		utils.FormatSize(m.analysis.WastedBytes),
		m.analysis.Efficiency*100,
	)

	// Table header
	tableHeader := fmt.Sprintf("\n%-5s %-12s %s", "Count", "Total Space", "Path")

	// Build full content with all rows
	var fullContent strings.Builder
	fullContent.WriteString(headerText)
	fullContent.WriteString("\n")
	fullContent.WriteString(styles.LayerHeaderStyle.Render(tableHeader))
	fullContent.WriteString("\n")

	if len(m.analysis.Inefficiencies) > 0 {
		for _, file := range m.analysis.Inefficiencies {
			row := fmt.Sprintf("%-5d %-12s %s", len(file.Nodes), utils.FormatSize(uint64(file.CumulativeSize)), file.Path)
			if lipgloss.Width(row) > width {
				row = runewidth.Truncate(row, width, "...")
			}
			fullContent.WriteString(styles.FileTreeModifiedStyle.Render(row))
			fullContent.WriteString("\n")
		}
	} else {
		fullContent.WriteString("No inefficiencies detected - great job!")
	}

	return fullContent.String()
}
