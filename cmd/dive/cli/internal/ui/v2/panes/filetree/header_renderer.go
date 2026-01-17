package filetree

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// RenderHeader creates a column header row with DYNAMIC columns
func RenderHeader(width int) string {
	// 1. Determine which columns to show
	showSize, showUid, showPerm := getColumnVisibility(width)

	// Header style (muted to not distract)
	headerColor := styles.BorderColor

	// Create cell styles
	sizeHeaderCell := lipgloss.NewStyle().Width(SizeWidth).Align(lipgloss.Right).Foreground(headerColor)
	uidGidHeaderCell := lipgloss.NewStyle().Width(UidGidWidth).Align(lipgloss.Right).Foreground(headerColor)
	permHeaderCell := lipgloss.NewStyle().Width(PermWidth).Align(lipgloss.Right).Foreground(headerColor)
	gapStyle := lipgloss.NewStyle().Width(len(MetaGap))

	// 2. Build metadata cells dynamically
	var metaCells []string

	if showSize {
		metaCells = append(metaCells, sizeHeaderCell.Render("Size"))
	}
	if showUid {
		// Add gap before UID if Size is also shown
		if len(metaCells) > 0 {
			metaCells = append(metaCells, gapStyle.Render(MetaGap))
		}
		metaCells = append(metaCells, uidGidHeaderCell.Render("UID:GID"))
	}
	if showPerm {
		// Add gap before Permissions
		if len(metaCells) > 0 {
			metaCells = append(metaCells, gapStyle.Render(MetaGap))
		}
		metaCells = append(metaCells, permHeaderCell.Render("Permissions"))
	}

	// If all hidden, just render "Name" header
	if len(metaCells) == 0 {
		return lipgloss.NewStyle().Foreground(headerColor).Render("Name")
	}

	// Join cells horizontally
	metaBlock := lipgloss.JoinHorizontal(lipgloss.Top, metaCells...)

	// CRITICAL: Must match viewport width (width - 2)!
	availableWidth := width - 2
	if availableWidth < 10 {
		availableWidth = 10
	}

	// Left part: "Name" label
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
