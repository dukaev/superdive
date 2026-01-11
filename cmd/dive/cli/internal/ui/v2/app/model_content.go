package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// generateLayersContent creates the full layers content string (for SetContent)
func (m *Model) generateLayersContent() string {
	if m.layerVM == nil || len(m.layerVM.Layers) == 0 {
		return "No layer data"
	}

	width := m.layout.LeftWidth - 2 // Subtract borders

	const (
		idWidth   = 12 // ID image (truncate to 12 chars)
		sizeWidth = 9  // Size (e.g., "999.9 MB")
		spaces    = 4  // Spaces between columns (2 after ID, 2 after Size)
	)

	var fullContent strings.Builder

	for i, layer := range m.layerVM.Layers {
		// 1. Selection marker
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.layerVM.LayerIndex {
			prefix = "● "
			style = v2styles.SelectedLayerStyle
		}

		// 2. Prepare ID (truncate if too long)
		id := layer.Id
		if len(id) > idWidth {
			id = id[:idWidth]
		}

		// 3. Prepare Size
		size := formatSize(layer.Size)

		// 4. Prepare Command (IMPORTANT: remove newlines!)
		rawCmd := strings.ReplaceAll(layer.Command, "\n", " ")
		rawCmd = strings.TrimSpace(rawCmd)

		// 5. Calculate available width for command
		availableCmdWidth := width - 2 - idWidth - spaces - sizeWidth

		// Minimum width protection for narrow terminals
		if availableCmdWidth < 5 {
			availableCmdWidth = 0
		}

		// Truncate command to fit available width
		cmd := ""
		if availableCmdWidth > 0 && rawCmd != "" {
			cmd = runewidth.Truncate(rawCmd, availableCmdWidth, "...")
		}

		// 6. Format string with fixed column widths
		text := fmt.Sprintf("%s%-*s  %*s  %s", prefix, idWidth, id, sizeWidth, size, cmd)

		// Protect against overflow
		maxLineWidth := width
		if runewidth.StringWidth(text) > maxLineWidth {
			text = runewidth.Truncate(text, maxLineWidth, "")
		}

		fullContent.WriteString(style.Render(text))
		fullContent.WriteString("\n")
	}

	return fullContent.String()
}

// generateImageContent creates the full image details content string (for SetContent)
func (m *Model) generateImageContent() string {
	width := m.layout.LeftWidth - 2 // Subtract borders

	// Header with statistics
	headerText := fmt.Sprintf(
		"Image name: %s\nTotal Image size: %s\nPotential wasted space: %s\nImage efficiency score: %.0f%%",
		m.analysis.Image,
		formatSize(m.analysis.SizeBytes),
		formatSize(m.analysis.WastedBytes),
		m.analysis.Efficiency*100,
	)

	// Table header
	tableHeader := fmt.Sprintf("\n%-5s %-12s %s", "Count", "Total Space", "Path")

	// Build full content with all rows
	var fullContent strings.Builder
	fullContent.WriteString(headerText)
	fullContent.WriteString("\n")
	fullContent.WriteString(v2styles.LayerHeaderStyle.Render(tableHeader))
	fullContent.WriteString("\n")

	if len(m.analysis.Inefficiencies) > 0 {
		for _, file := range m.analysis.Inefficiencies {
			row := fmt.Sprintf("%-5d %-12s %s", len(file.Nodes), formatSize(uint64(file.CumulativeSize)), file.Path)
			if lipgloss.Width(row) > width {
				row = runewidth.Truncate(row, width, "...")
			}
			fullContent.WriteString(v2styles.FileTreeModifiedStyle.Render(row))
			fullContent.WriteString("\n")
		}
	} else {
		fullContent.WriteString("No inefficiencies detected - great job!")
	}

	return fullContent.String()
}

// formatSize formats bytes into human-readable size
func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
