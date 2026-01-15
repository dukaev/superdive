package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// --- Base Styles ---

var (
	// TitleStyle for main titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Background(lipgloss.Color("#1C1C1E")).
			Padding(0, 1)

	// StatusStyle for status bar
	StatusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(SecondaryColor).
			Padding(0, 1)

	// FilterStyle for filter input
	FilterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(DarkGrayColor).
			Padding(0, 1)
)

// --- Component Styles ---

// SelectedLayerStyle highlights the currently selected layer
var SelectedLayerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		Background(lipgloss.Color("#1C1C1E"))

// LayerHeaderStyle for layer field headers
var LayerHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(SecondaryColor)

// LayerValueStyle for layer field values
var LayerValueStyle = lipgloss.NewStyle().
		Foreground(LightGrayColor)

// FileTreeDirStyle for directories in file tree
var FileTreeDirStyle = lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

// FileTreeModifiedStyle for modified files in file tree
var FileTreeModifiedStyle = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true)

// --- File Stats Styles ---

// FileStatsAddedStyle for added files count
var FileStatsAddedStyle = lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

// FileStatsModifiedStyle for modified files count
var FileStatsModifiedStyle = lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true)

// FileStatsRemovedStyle for removed files count
var FileStatsRemovedStyle = lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true)

// --- Rendering Functions ---

// RenderBox creates a bordered box with title and content
// IMPORTANT: Truncates title to guarantee single line height
func RenderBox(title string, width, height int, content string, isSelected bool) string {
	// 1. Protect minimum sizes
	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}

	borderColor := BorderColor
	if isSelected {
		borderColor = PrimaryColor
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2)

	if title == "" {
		if content == "" {
			content = " "
		}
		return boxStyle.Render(content)
	}

	// 2. Truncate title to prevent wrapping to 2 lines
	// Title width: Window width - 2 (borders) - 2 (margin)
	maxTitleWidth := width - 4
	if maxTitleWidth < 0 {
		maxTitleWidth = 0
	}

	truncatedTitle := runewidth.Truncate(title, maxTitleWidth, "…")

	// 3. Render title
	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	titleLine := titleStyle.Render(truncatedTitle)

	// 4. Assemble content: Title + Space + Data
	// Using " " (space) to guarantee 1 line height for padding
	innerContent := lipgloss.JoinVertical(lipgloss.Left, titleLine, " ", content)

	return boxStyle.Render(innerContent)
}

// --- Utility Functions ---

// TruncateString truncates a string by visual width
func TruncateString(s string, maxLen int) string {
	return runewidth.Truncate(s, maxLen, "...")
}

// --- File Tree Visual Styles ---

// TreeGuideStyle for tree guide lines (│ ├ └)
var TreeGuideStyle = lipgloss.NewStyle().
	Foreground(DarkGrayColor)

// MetaDataStyle for permissions, UID, and size (muted, less prominent)
var MetaDataStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#6e6e73"))

// HelpStyle for help/instruction bar at the bottom (gray, muted)
var HelpStyle = lipgloss.NewStyle().
	Foreground(GrayColor)
