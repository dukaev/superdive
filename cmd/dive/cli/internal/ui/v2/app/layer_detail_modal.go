package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/image"
)

// LayerDetailMsg is sent to show layer details
type LayerDetailMsg struct {
	Layer *image.Layer
}

// LayerDetailModal manages the layer detail modal
type LayerDetailModal struct {
	visible bool
	layer   *image.Layer
	width   int
	height  int
}

// NewLayerDetailModal creates a new layer detail modal
func NewLayerDetailModal() LayerDetailModal {
	return LayerDetailModal{
		visible: false,
		layer:   nil,
	}
}

// Show makes the modal visible with the given layer
func (m *LayerDetailModal) Show(layer *image.Layer) {
	m.visible = true
	m.layer = layer
}

// Hide hides the modal
func (m *LayerDetailModal) Hide() {
	m.visible = false
	m.layer = nil
}

// IsVisible returns whether the modal is visible
func (m *LayerDetailModal) IsVisible() bool {
	return m.visible
}

// Update handles messages for the modal
func (m LayerDetailModal) Update(msg tea.Msg) (LayerDetailModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case LayerDetailMsg:
		m.Show(msg.Layer)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", " ", "enter":
			m.Hide()
			return m, nil
		}
	}

	return m, nil
}

// View renders the modal
func (m LayerDetailModal) View(screenWidth, screenHeight int) string {
	if !m.visible || m.layer == nil {
		return ""
	}

	// Calculate modal dimensions
	modalWidth := min(screenWidth-10, 80)
	modalHeight := min(screenHeight-10, 25)

	// Create modal style
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(v2styles.PrimaryColor).
		Background(lipgloss.Color("#1C1C1E")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	// Build content
	title := v2styles.LayerHeaderStyle.Render("📦 Layer Details")
	content := m.buildContent(modalWidth - 4) // -4 for padding
	help := v2styles.LayerValueStyle.Render("ESC/SPACE/ENTER: close")

	fullContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		content,
		"",
		help,
	)

	// Render modal
	modal := modalStyle.Render(fullContent)

	// Place modal in center
	return lipgloss.Place(screenWidth, screenHeight, lipgloss.Center, lipgloss.Center, modal)
}

// buildContent creates the modal content
func (m LayerDetailModal) buildContent(width int) string {
	if m.layer == nil {
		return "No layer data available"
	}

	var lines []string

	// Layer ID
	lines = append(lines, m.renderField("ID", m.layer.Id, width))

	// Digest
	lines = append(lines, m.renderField("Digest", m.layer.Digest, width))

	// Index
	lines = append(lines, m.renderField("Index", fmt.Sprintf("%d", m.layer.Index), width))

	// Size
	size := utils.FormatSize(m.layer.Size)
	lines = append(lines, m.renderField("Size", fmt.Sprintf("%s (%d bytes)", size, m.layer.Size), width))

	// Human-readable size
	humanSize := humanize.Bytes(m.layer.Size)
	lines = append(lines, m.renderField("Human Size", humanSize, width))

	// Names
	if len(m.layer.Names) > 0 {
		names := strings.Join(m.layer.Names, ", ")
		lines = append(lines, m.renderField("Names", names, width))
	}

	// Command (multiline)
	lines = append(lines, "")
	lines = append(lines, v2styles.LayerHeaderStyle.Render("Command:"))
	cmdLines := m.wrapText(m.layer.Command, width)
	for _, line := range cmdLines {
		lines = append(lines, v2styles.LayerValueStyle.Render(line))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderField renders a key-value pair
func (m LayerDetailModal) renderField(key, value string, width int) string {
	label := v2styles.LayerHeaderStyle.Render(key + ":")
	return lipgloss.JoinHorizontal(lipgloss.Top, label, " ", v2styles.LayerValueStyle.Render(value))
}

// wrapText wraps text to fit width
func (m LayerDetailModal) wrapText(text string, width int) []string {
	if text == "" {
		return []string{"(none)"}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{"(none)"}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine == "" {
			testLine = word
		} else {
			testLine += " " + word
		}

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
