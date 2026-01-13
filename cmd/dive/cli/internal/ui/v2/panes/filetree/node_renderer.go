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

	// 1. Icon and base color
	icon := styles.IconFile
	diffIcon := "" // 1 space (compact, like nvim-tree)
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

	// 2. Diff status (color only, no icons)
	switch node.Data.DiffType {
	case filetree.Added:
		color = styles.DiffAddedColor
	case filetree.Removed:
		color = styles.DiffRemovedColor
	case filetree.Modified:
		color = styles.DiffModifiedColor
	}

	// 3. Format metadata (right-aligned)
	perm := FormatPermissions(node.Data.FileInfo.Mode)

	// Show UID:GID only if not the default root:root (0:0)
	var uidGid string
	if node.Data.FileInfo.Uid != 0 || node.Data.FileInfo.Gid != 0 {
		uidGid = FormatUidGid(node.Data.FileInfo.Uid, node.Data.FileInfo.Gid)
	} else {
		uidGid = "-"
	}

	// Size (empty for folders)
	var sizeStr string
	if !node.Data.FileInfo.IsDir() {
		sizeStr = utils.FormatSize(uint64(node.Data.FileInfo.Size))
	}

	// Format name
	name := node.Name
	if name == "" {
		name = "/"
	}
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// 4. Build line with new order: cursor | tree-guides diff icon icon filename [metadata...]

	// Cursor mark (same as Layers)
	cursorMark := ""
	if isSelected {
		cursorMark = ""
	}

	// Render tree guides (lines should be gray)
	// If selected, apply background to tree guides too
	styledPrefix := styles.TreeGuideStyle.Render(prefix)
	if isSelected {
		styledPrefix = lipgloss.NewStyle().
			Foreground(styles.DarkGrayColor).
			Background(lipgloss.Color("#1C1C1E")).
			Render(prefix)
	}

	// Render filename with diff color
	nameStyle := lipgloss.NewStyle().Foreground(color)
	if isSelected {
		// Use SelectedLayerStyle (with background and primary color)
		nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor).Background(lipgloss.Color("#1C1C1E"))
	}

	// Render metadata with FIXED WIDTH columns for strict grid layout
	// Each column gets exact width to ensure headers align with data

	// Base metadata color
	metaColor := lipgloss.Color("#6e6e73")

	// If selected, use background color for metadata too
	metaBg := lipgloss.Color("")
	if isSelected {
		metaBg = lipgloss.Color("#1C1C1E")
	}

	// Create cell styles with fixed width and right alignment
	sizeCell := lipgloss.NewStyle().
		Width(SizeWidth).
		Align(lipgloss.Right).
		Foreground(metaColor).
		Background(metaBg)

	uidGidCell := lipgloss.NewStyle().
		Width(UidGidWidth).
		Align(lipgloss.Right).
		Foreground(metaColor).
		Background(metaBg)

	permCell := lipgloss.NewStyle().
		Width(PermWidth).
		Align(lipgloss.Right).
		Foreground(metaColor).
		Background(metaBg)

	// Render each cell with fixed width
	styledSize := sizeCell.Render(sizeStr)
	styledUidGid := uidGidCell.Render(uidGid)
	styledPerm := permCell.Render(perm)

	// Gap style (must have background if selected)
	gapStyle := lipgloss.NewStyle().Width(len(MetaGap))
	if isSelected {
		gapStyle = gapStyle.Background(lipgloss.Color("#1C1C1E"))
	}
	styledGap := gapStyle.Render(MetaGap)

	// Join cells horizontally with gap
	// This creates a rigid block where each column has exact width
	metaBlock := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styledSize,
		styledGap,
		styledUidGid,
		styledGap,
		styledPerm,
	)

	// Calculate widths for truncation
	// Fixed part: cursor + prefix + diffIcon + icon
	fixedPartWidth := runewidth.StringWidth(cursorMark) +
		runewidth.StringWidth(prefix) +
		runewidth.StringWidth(diffIcon) +
		runewidth.StringWidth(icon)

	// Get actual metadata block width (should be: sizeWidth + gap + uidGidWidth + gap + permWidth)
	metaBlockWidth := lipgloss.Width(metaBlock)

	// Available width for filename (between file and right-aligned metadata)
	availableForName := width - fixedPartWidth - metaBlockWidth - 2 // -2 for gaps
	if availableForName < 5 {
		availableForName = 5
	}

	// Truncate name if needed
	displayName := name
	if runewidth.StringWidth(name) > availableForName {
		displayName = runewidth.Truncate(name, availableForName, "…")
	}

	styledName := nameStyle.Render(displayName)

	// Apply background to diffIcon and icon if selected
	if isSelected {
		bg := lipgloss.Color("#1C1C1E")
		if diffIcon != "" {
			diffIcon = lipgloss.NewStyle().Background(bg).Render(diffIcon)
		}
		icon = lipgloss.NewStyle().Background(bg).Render(icon)
	}

	// Calculate EXACT padding to push metadata to the right edge
	// Current content width (without spacer)
	currentContentWidth := fixedPartWidth + runewidth.StringWidth(displayName) + metaBlockWidth

	// How many spaces needed to fill to width?
	paddingNeeded := width - currentContentWidth
	if paddingNeeded < 1 {
		paddingNeeded = 1 // At least 1 space gap
	}

	// If selected, padding should also have background
	paddingStyle := lipgloss.NewStyle()
	if isSelected {
		paddingStyle = paddingStyle.Background(lipgloss.Color("#1C1C1E"))
	}
	padding := paddingStyle.Render(strings.Repeat(" ", paddingNeeded))

	// Assemble final line: tree-guides cursor diff icon filename [SPACER] metadata
	sb.WriteString(styledPrefix)
	sb.WriteString(cursorMark)
	sb.WriteString(diffIcon)
	sb.WriteString(icon)
	sb.WriteString(styledName)
	sb.WriteString(padding) // <--- THIS PUSHES METADATA TO THE RIGHT EDGE
	sb.WriteString(metaBlock)
	sb.WriteString("\n")

	// Note: Filename comes first, metadata is right-aligned at the end
	// Order: filename → size → uid:gid → permissions
}
