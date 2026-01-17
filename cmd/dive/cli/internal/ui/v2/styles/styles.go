package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// --- 1. BASE STYLES ---

var (
	// TitleStyle for main titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Background(PanelBgColor).
			Padding(0, 1)

	// StatusStyle for status bar
	StatusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(MainTextColor).
			Background(SecondaryColor).
			Padding(0, 1)

	// FilterStyle for filter input
	FilterStyle = lipgloss.NewStyle().
			Foreground(MainTextColor).
			Background(InputBgColor).
			Padding(0, 1)
)

// --- 2. COMPONENT STYLES ---

// SelectedLayerStyle highlights the currently selected layer
var SelectedLayerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		Background(PanelBgColor)

// LayerHeaderStyle for layer field headers
var LayerHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(SecondaryColor)

// LayerValueStyle for layer field values
var LayerValueStyle = lipgloss.NewStyle().
		Foreground(LightGray)

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

// --- 4. FILE TREE VISUAL STYLES ---

// TreeGuideStyle for tree guide lines (│ ├ └)
var TreeGuideStyle = lipgloss.NewStyle().
	Foreground(DarkGray)

// MetaDataStyle for permissions, UID, and size (muted, less prominent)
var MetaDataStyle = lipgloss.NewStyle().
	Foreground(MutedTextColor)

// HelpStyle for help/instruction bar at the bottom (gray, muted)
var HelpStyle = lipgloss.NewStyle().
	Foreground(Gray)

// --- 5. SEARCH BAR STYLES ---

// SearchPrefixStyle for the "Filter:" prefix in search bar
var SearchPrefixStyle = lipgloss.NewStyle().
	Foreground(PanelBgColor).  // Dark text
	Background(PrimaryColor).  // Accent background (blue)
	Bold(true).
	Padding(0, 1)

// SearchInputStyle for the search input text (blue like the prefix)
var SearchInputStyle = lipgloss.NewStyle().
	Foreground(PrimaryColor).   // Blue text (accent color)
	Background(StatusBarBgColor). // Dark gray status bar background
	Padding(0, 1)

// SearchErrorStyle for invalid regex indication
var SearchErrorStyle = lipgloss.NewStyle().
	Foreground(ErrorColor).    // Red text on error
	Background(StatusBarBgColor).
	Padding(0, 1)
