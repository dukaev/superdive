package filetree

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/filetree"
)

// RenderNodeWithCursor renders a node with tree guides and improved visual design
func RenderNodeWithCursor(sb *strings.Builder, node *filetree.FileNode, prefix string, isSelected bool, width int) {
	if node == nil {
		return
	}
	row := RenderRow(node, prefix, "", isSelected, width, nil)
	sb.WriteString(row)
	sb.WriteString("\n")
}

// RenderRow renders a single tree node row using lipgloss.JoinHorizontal for clean layout
// displayName is optional - if empty, node.Name will be used
// filterRegex is optional - if provided, matching text will be highlighted
func RenderRow(node *filetree.FileNode, prefix string, displayName string, isSelected bool, width int, filterRegex *regexp.Regexp) string {
	// 1. Icon and color
	icon := styles.IconFile
	color := styles.DiffNormalColor

	if node.Data.FileInfo.IsDir() {
		if node.Data.ViewInfo.Collapsed {
			icon = styles.IconDirClosed
		} else {
			icon = styles.IconDirOpen
		}
	} else if node.Data.FileInfo.TypeFlag == 16 { // Symlink
		icon = styles.IconSymlink
	}

	// 2. Diff status color
	switch node.Data.DiffType {
	case filetree.Added:
		color = styles.DiffAddedColor
	case filetree.Removed:
		color = styles.DiffRemovedColor
	case filetree.Modified:
		color = styles.DiffModifiedColor
	}

	// 3. Format metadata (fixed width, right-aligned)
	perm := FormatPermissions(node.Data.FileInfo.Mode)

	var uidGid string
	if node.Data.FileInfo.Uid != 0 || node.Data.FileInfo.Gid != 0 {
		uidGid = FormatUidGid(node.Data.FileInfo.Uid, node.Data.FileInfo.Gid)
	} else {
		uidGid = "-"
	}

	var sizeStr string
	if !node.Data.FileInfo.IsDir() {
		sizeStr = utils.FormatSize(uint64(node.Data.FileInfo.Size))
	}

	// Format name with symlink target
	name := displayName
	if name == "" {
		name = node.Name
	}
	if name == "" {
		name = "/"
	}
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// 4. Common background for selected state
	// IMPROVED: Better contrast with lighter background
	bg := lipgloss.Color("")
	if isSelected {
		bg = lipgloss.Color("#48484A") // Lighter gray for better contrast
	}

	// 5. Build styled components
	// Tree guides (prefix)
	prefixStyle := lipgloss.NewStyle().Foreground(styles.DarkGrayColor).Background(bg)
	styledPrefix := prefixStyle.Render(prefix)

	// Icon
	iconStyle := lipgloss.NewStyle().Background(bg)
	styledIcon := iconStyle.Render(icon)

	// Filename with diff color
	// Apply match highlighting if filter is provided
	var finalStyledName string
	if filterRegex != nil && filterRegex.MatchString(name) {
		// Highlight matching portions
		finalStyledName = highlightMatches(name, filterRegex, color, isSelected, bg)
	} else {
		// No highlighting, use normal style
		nameStyle := lipgloss.NewStyle().Foreground(color).Background(bg)
		if isSelected {
			nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
		}
		finalStyledName = nameStyle.Render(name)
	}

	// 6. Render metadata cells (fixed width)
	metaColor := lipgloss.Color("#6e6e73")

	sizeCell := lipgloss.NewStyle().
		Width(SizeWidth).
		Align(lipgloss.Right).
		Foreground(metaColor).
		Background(bg).
		Render(sizeStr)

	uidGidCell := lipgloss.NewStyle().
		Width(UidGidWidth).
		Align(lipgloss.Right).
		Foreground(metaColor).
		Background(bg).
		Render(uidGid)

	permCell := lipgloss.NewStyle().
		Width(PermWidth).
		Align(lipgloss.Right).
		Foreground(metaColor).
		Background(bg).
		Render(perm)

	gap := lipgloss.NewStyle().Width(len(MetaGap)).Background(bg).Render(MetaGap)

	// Metadata block (right-aligned columns)
	metaBlock := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sizeCell, gap, uidGidCell, gap, permCell,
	)

	// 7. Calculate available width for filename
	fixedPartWidth := lipgloss.Width(styledPrefix) + lipgloss.Width(styledIcon)
	metaBlockWidth := lipgloss.Width(metaBlock)

	availableForName := width - fixedPartWidth - metaBlockWidth - 2 // -2 for spacing
	if availableForName < 5 {
		availableForName = 5
	}

	// Truncate name if needed (check visual width, not character count)
	truncatedName := name
	if runewidth.StringWidth(finalStyledName) > availableForName {
		// If highlighting makes it too long, truncate without highlighting
		if runewidth.StringWidth(name) > availableForName {
			truncatedName = runewidth.Truncate(name, availableForName, "…")
		}
		// Re-apply highlighting to truncated name
		if filterRegex != nil && filterRegex.MatchString(truncatedName) {
			finalStyledName = highlightMatches(truncatedName, filterRegex, color, isSelected, bg)
		} else {
			nameStyle := lipgloss.NewStyle().Foreground(color).Background(bg)
			if isSelected {
				nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
			}
			finalStyledName = nameStyle.Render(truncatedName)
		}
	}

	// 8. Calculate flexible padding to push metadata to right edge
	contentWidth := fixedPartWidth + lipgloss.Width(finalStyledName) + metaBlockWidth
	paddingNeeded := width - contentWidth
	if paddingNeeded < 1 {
		paddingNeeded = 1
	}

	padding := lipgloss.NewStyle().Width(paddingNeeded).Background(bg).Render(strings.Repeat(" ", paddingNeeded))

	// 9. Join all components horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		styledPrefix,
		styledIcon,
		finalStyledName,
		padding,
		metaBlock,
	)
}

// highlightMatches applies regex highlighting to matching portions of the text
func highlightMatches(text string, filter *regexp.Regexp, baseColor lipgloss.Color, isSelected bool, bg lipgloss.Color) string {
	// Find all matches
	matches := filter.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		// Should not happen since we check MatchString before calling
		nameStyle := lipgloss.NewStyle().Foreground(baseColor).Background(bg)
		if isSelected {
			nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
		}
		return nameStyle.Render(text)
	}

	// Build highlighted string
	var result strings.Builder
	lastEnd := 0

	// Highlight color: bright yellow for visibility
	highlightColor := lipgloss.Color("#FFFF00") // Bright yellow
	if isSelected {
		highlightColor = lipgloss.Color("#FFD700") // Gold for selected state
	}

	normalStyle := lipgloss.NewStyle().Foreground(baseColor).Background(bg)
	if isSelected {
		normalStyle = normalStyle.Bold(true).Foreground(styles.PrimaryColor)
	}

	highlightStyle := lipgloss.NewStyle().Foreground(highlightColor).Background(bg)
	if isSelected {
		highlightStyle = highlightStyle.Bold(true)
	}

	for _, match := range matches {
		// Add non-matching text before this match
		if match[0] > lastEnd {
			result.WriteString(normalStyle.Render(text[lastEnd:match[0]]))
		}

		// Add matching text with highlight
		result.WriteString(highlightStyle.Render(text[match[0]:match[1]]))

		lastEnd = match[1]
	}

	// Add remaining text after last match
	if lastEnd < len(text) {
		result.WriteString(normalStyle.Render(text[lastEnd:]))
	}

	return result.String()
}

// RenderNodeLine renders a single node line for viewport.
// This is a convenience wrapper around RenderRow.
func RenderNodeLine(node *filetree.FileNode, prefix string, isSelected bool, width int, filterRegex *regexp.Regexp) string {
	return RenderRow(node, prefix, "", isSelected, width, filterRegex)
}

// RenderNodeLineWithDisplayName renders a single node line with a custom display name.
// This is used for flat view where the full path is shown instead of just the name.
func RenderNodeLineWithDisplayName(node *filetree.FileNode, prefix string, displayName string, isSelected bool, width int, filterRegex *regexp.Regexp) string {
	return RenderRow(node, prefix, displayName, isSelected, width, filterRegex)
}
