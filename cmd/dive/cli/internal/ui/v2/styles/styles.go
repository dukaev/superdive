package styles

import (
	"strings"

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
	Foreground(MutedTextColor) // Changed from LightGray to adaptive MutedTextColor

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

// RenderBox creates a bordered box with title embedded in the border
// IMPORTANT: Title is rendered directly on the top border, not inside content area
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

	// 2. Truncate title to prevent breaking the border
	maxTitleWidth := width - 4
	if maxTitleWidth < 0 {
		maxTitleWidth = 0
	}
	truncatedTitle := runewidth.Truncate(title, maxTitleWidth, "…")

	// 3. Create base box style and render content
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2)

	if content == "" {
		content = " "
	}

	// 4. If no title, render normally
	if truncatedTitle == "" {
		return boxStyle.Render(content)
	}

	// 5. Render box with title on top border
	// First render standard box
	rendered := boxStyle.Render(content)

	// Split into lines
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	// Replace top border line with title version
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)
	if isSelected {
		titleStyle = titleStyle.Foreground(PrimaryColor)
	}

	titleWidth := runewidth.StringWidth(truncatedTitle)
	borderFillWidth := width - titleWidth - 4
	if borderFillWidth < 0 {
		borderFillWidth = 0
	}

	// Build new top border: ┌─title───┐
	// Render each part separately to preserve colors
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	leftPart := borderStyle.Render("┌─")
	titlePart := titleStyle.Render(truncatedTitle)
	rightPart := borderStyle.Render("─" + strings.Repeat("─", borderFillWidth) + "┐")

	topBorder := leftPart + titlePart + rightPart

	// Replace first line
	lines[0] = topBorder

	return strings.Join(lines, "\n")
}

// --- Utility Functions ---

// TruncateString truncates a string by visual width
func TruncateString(s string, maxLen int) string {
	return runewidth.Truncate(s, maxLen, "...")
}

// --- 4. FILE TREE VISUAL STYLES ---

// TreeGuideStyle for tree guide lines (│ ├ └)
var TreeGuideStyle = lipgloss.NewStyle().
	Foreground(BorderColor) // Changed from DarkGray to adaptive BorderColor

// MetaDataStyle for permissions, UID, and size (muted, less prominent)
var MetaDataStyle = lipgloss.NewStyle().
	Foreground(MutedTextColor)

// HelpStyle for help/instruction bar at the bottom (muted)
var HelpStyle = lipgloss.NewStyle().
	Foreground(MutedTextColor) // Changed from Gray to adaptive MutedTextColor

// --- 5. SEARCH BAR STYLES ---

// SearchPrefixStyle for the "Filter:" prefix in search bar
var SearchPrefixStyle = lipgloss.NewStyle().
	Foreground(PrimaryColor). // Blue text (accent color)
	Bold(true)

// SearchInputStyle for the search input text (yellow for visibility)
var SearchInputStyle = lipgloss.NewStyle().
	Foreground(HighlightColor) // Yellow text (highlight color)

// SearchErrorStyle for invalid regex indication
var SearchErrorStyle = lipgloss.NewStyle().
	Foreground(ErrorColor) // Red text on error
