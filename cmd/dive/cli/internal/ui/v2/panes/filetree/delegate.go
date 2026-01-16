package filetree

import (
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/filetree"
)

// TreeDelegate handles rendering of a single row in the file tree list
type TreeDelegate struct {
	// Can store shared styles here to avoid recreating them
}

func NewTreeDelegate() TreeDelegate {
	return TreeDelegate{}
}

func (d TreeDelegate) Height() int {
	return 1
}

func (d TreeDelegate) Spacing() int {
	return 0
}

func (d TreeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

// Render renders a single row of the file tree
func (d TreeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(TreeItem)
	if !ok {
		return
	}

	node := item.node
	isSelected := index == m.Index()

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

	// Color for Diff status
	switch node.Data.DiffType {
	case filetree.Added:
		color = styles.DiffAddedColor
	case filetree.Removed:
		color = styles.DiffRemovedColor
	case filetree.Modified:
		color = styles.DiffModifiedColor
	}

	// 2. Metadata (size, permissions)
	perm := FormatPermissions(node.Data.FileInfo.Mode)
	uidGid := "-"
	if node.Data.FileInfo.Uid != 0 || node.Data.FileInfo.Gid != 0 {
		uidGid = FormatUidGid(node.Data.FileInfo.Uid, node.Data.FileInfo.Gid)
	}

	var sizeStr string
	if !node.Data.FileInfo.IsDir() {
		sizeStr = utils.FormatSize(uint64(node.Data.FileInfo.Size))
	}

	// 3. Style the name
	name := node.Name
	if name == "" {
		name = "/"
	}
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	nameStyle := lipgloss.NewStyle().Foreground(color)
	if isSelected {
		nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
	}

	// 4. Build the row (similar to old code, but with list.Model width)
	width := m.Width()
	if width <= 0 {
		width = 80
	}

	// Metadata block (fixed width)
	metaColor := lipgloss.Color("#6e6e73")
	metaBg := lipgloss.Color("")
	if isSelected {
		metaBg = lipgloss.Color("#1C1C1E") // Dark background for selected row
	}

	// Create cell styles (code similar to old RenderNodeWithCursor)
	sizeCell := lipgloss.NewStyle().Width(SizeWidth).Align(lipgloss.Right).Foreground(metaColor).Background(metaBg).Render(sizeStr)
	uidCell := lipgloss.NewStyle().Width(UidGidWidth).Align(lipgloss.Right).Foreground(metaColor).Background(metaBg).Render(uidGid)
	permCell := lipgloss.NewStyle().Width(PermWidth).Align(lipgloss.Right).Foreground(metaColor).Background(metaBg).Render(perm)
	gap := lipgloss.NewStyle().Width(len(MetaGap)).Background(metaBg).Render(MetaGap)

	metaBlock := lipgloss.JoinHorizontal(lipgloss.Top, sizeCell, gap, uidCell, gap, permCell)
	metaWidth := lipgloss.Width(metaBlock)

	// Left part (tree + name)
	styledPrefix := styles.TreeGuideStyle.Render(item.prefix)
	if isSelected {
		styledPrefix = lipgloss.NewStyle().Foreground(styles.DarkGrayColor).Background(metaBg).Render(item.prefix)
	}

	// FIX: Use lipgloss.Width for styled strings (ignores ANSI codes)
	prefixWidth := lipgloss.Width(styledPrefix)
	iconWidth := lipgloss.Width(icon)
	// diffIcon is currently empty, but if used, measure with lipgloss.Width

	fixedLeftWidth := prefixWidth + iconWidth

	// Calculate space for name
	availableForName := width - fixedLeftWidth - metaWidth - 1

	displayName := name
	if runewidth.StringWidth(name) > availableForName && availableForName > 0 {
		displayName = runewidth.Truncate(name, availableForName, "…")
	}

	styledName := nameStyle.Background(metaBg).Render(displayName)
	nameWidth := lipgloss.Width(styledName)

	// Icons with background
	styledIcon := icon
	if isSelected {
		styledIcon = lipgloss.NewStyle().Background(metaBg).Render(icon)
	}

	// Apply background to diffIcon and icon if selected
	if isSelected {
		bg := lipgloss.Color("#1C1C1E")
		if diffIcon != "" {
			diffIcon = lipgloss.NewStyle().Background(bg).Render(diffIcon)
		}
		icon = lipgloss.NewStyle().Background(bg).Render(icon)
	}

	// Padding between name and metadata
	// FIX: Use lipgloss.Width for styled strings
	contentWidth := prefixWidth + iconWidth + nameWidth + metaWidth
	paddingNeeded := width - contentWidth
	if paddingNeeded < 1 {
		paddingNeeded = 1
	}
	padding := strings.Repeat(" ", paddingNeeded)
	if isSelected {
		padding = lipgloss.NewStyle().Background(metaBg).Render(padding)
	}

	// Final assembly
	fmt.Fprintf(w, "%s%s%s%s%s%s", styledPrefix, diffIcon, styledIcon, styledName, padding, metaBlock)
}
