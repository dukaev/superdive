package filetree

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// RenderHeader creates a column header row with FIXED WIDTH columns
func RenderHeader(width int) string {
	// Header style (muted to not distract)
	headerColor := styles.DarkGrayColor

	// Create cell styles with FIXED WIDTH (same as renderer.go!)
	sizeHeaderCell := lipgloss.NewStyle().
		Width(SizeWidth).
		Align(lipgloss.Right).
		Foreground(headerColor)

	uidGidHeaderCell := lipgloss.NewStyle().
		Width(UidGidWidth).
		Align(lipgloss.Right).
		Foreground(headerColor)

	permHeaderCell := lipgloss.NewStyle().
		Width(PermWidth).
		Align(lipgloss.Right).
		Foreground(headerColor)

	// Render each header cell with fixed width
	styledSizeHeader := sizeHeaderCell.Render("Size")
	styledUidGidHeader := uidGidHeaderCell.Render("UID:GID")
	styledPermHeader := permHeaderCell.Render("Permissions")

	// Join cells horizontally with gap (same as renderer.go!)
	metaBlock := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styledSizeHeader,
		lipgloss.NewStyle().Width(len(MetaGap)).Render(MetaGap),
		styledUidGidHeader,
		lipgloss.NewStyle().Width(len(MetaGap)).Render(MetaGap),
		styledPermHeader,
	)

	// CRITICAL: Must match viewport width (width - 2)!
	// Viewport is created with width-2, so header must use the same width
	availableWidth := width - 2
	if availableWidth < 10 {
		availableWidth = 10
	}

	// Left part: "Name" label (with muted color like other headers)
	nameHeaderStyle := lipgloss.NewStyle().Foreground(headerColor)
	styledNameHeader := nameHeaderStyle.Render("Name")

	// Get actual metadata block width
	metaBlockWidth := lipgloss.Width(metaBlock)

	// Calculate padding to push metadata to the right
	padding := availableWidth - runewidth.StringWidth("Name") - metaBlockWidth
	if padding < 1 {
		padding = 1
	}

	// Assemble: Name + padding + right-aligned metadata
	fullText := styledNameHeader + strings.Repeat(" ", padding) + metaBlock

	return fullText
}
