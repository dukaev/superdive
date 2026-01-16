package filetree

import (
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
	row := RenderRow(node, prefix, isSelected, width)
	sb.WriteString(row)
	sb.WriteString("\n")
}

// RenderRow renders a single tree node row using lipgloss.JoinHorizontal for clean layout
func RenderRow(node *filetree.FileNode, prefix string, isSelected bool, width int) string {
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
	name := node.Name
	if name == "" {
		name = "/"
	}
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// 4. Common background for selected state
	bg := lipgloss.Color("")
	if isSelected {
		bg = lipgloss.Color("#1C1C1E")
	}

	// 5. Build styled components
	// Tree guides (prefix)
	prefixStyle := lipgloss.NewStyle().Foreground(styles.DarkGrayColor).Background(bg)
	styledPrefix := prefixStyle.Render(prefix)

	// Icon
	iconStyle := lipgloss.NewStyle().Background(bg)
	styledIcon := iconStyle.Render(icon)

	// Filename with diff color
	nameStyle := lipgloss.NewStyle().Foreground(color).Background(bg)
	if isSelected {
		nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
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

	// Truncate name if needed
	displayName := name
	if runewidth.StringWidth(name) > availableForName {
		displayName = runewidth.Truncate(name, availableForName, "…")
	}
	styledName := nameStyle.Render(displayName)

	// 8. Calculate flexible padding to push metadata to right edge
	contentWidth := fixedPartWidth + lipgloss.Width(styledName) + metaBlockWidth
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
		styledName,
		padding,
		metaBlock,
	)
}

// RenderNodeLine renders a single node line for viewport.
// This is a convenience wrapper around RenderRow.
func RenderNodeLine(node *filetree.FileNode, prefix string, isSelected bool, width int) string {
	return RenderRow(node, prefix, isSelected, width)
}
