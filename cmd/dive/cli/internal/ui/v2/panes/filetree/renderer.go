package filetree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/dive/filetree"
)

// VisibleNode represents a node with its depth for rendering
type VisibleNode struct {
	Node  *filetree.FileNode
	Depth int
}

// collectVisibleNodes collects all visible nodes in a flat list
func collectVisibleNodes(root *filetree.FileNode) []VisibleNode {
	var nodes []VisibleNode

	var traverse func(*filetree.FileNode, int)
	traverse = func(node *filetree.FileNode, depth int) {
		if node == nil {
			return
		}

		// Skip root node itself, start from children
		if node.Parent != nil {
			nodes = append(nodes, VisibleNode{Node: node, Depth: depth})
		}

		// Recurse into children if directory and not collapsed
		if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed {
			sortedChildren := sortChildren(node.Children)
			for _, child := range sortedChildren {
				traverse(child, depth+1)
			}
		}
	}

	// Start from root's children if root is not collapsed
	if !root.Data.ViewInfo.Collapsed {
		sortedChildren := sortChildren(root.Children)
		for _, child := range sortedChildren {
			traverse(child, 0)
		}
	}

	return nodes
}

// renderNodeWithCursor renders a node with optional cursor indicator
func renderNodeWithCursor(sb *strings.Builder, node *filetree.FileNode, depth int, isSelected bool, width int) {
	if node == nil {
		return
	}

	// 1. Cursor indicator
	var cursor string
	if isSelected {
		cursor = "▸ "
	} else {
		cursor = "  "
	}

	// 2. Icon and color
	icon := styles.IconFile
	diffIcon := ""
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

	// 3. Diff status
	switch node.Data.DiffType {
	case filetree.Added:
		color = styles.DiffAddedColor
		diffIcon = styles.IconAdded
	case filetree.Removed:
		color = styles.DiffRemovedColor
		diffIcon = styles.IconRemoved
	case filetree.Modified:
		color = styles.DiffModifiedColor
		diffIcon = styles.IconModified
	}

	// 4. Format name
	name := node.Name
	if name == "" {
		name = "/"
	}
	// Symlinks
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// 5. Indent
	indent := strings.Repeat("  ", depth)

	// 6. Build line (without styles yet)
	// Add space after diffIcon if present
	if diffIcon != "" {
		diffIcon += " "
	}

	rawText := fmt.Sprintf("%s%s%s%s", indent+cursor, diffIcon, icon, name)

	// ВАЖНО: Truncate to prevent line wrapping which breaks scroll alignment
	// CRITICAL: Leave 1 char margin for terminal cursor to prevent auto-scroll
	maxTextWidth := width - 1
	if maxTextWidth < 10 {
		maxTextWidth = 10 // Protection
	}

	truncatedText := runewidth.Truncate(rawText, maxTextWidth, "…")

	// 7. Apply style
	style := lipgloss.NewStyle().Foreground(color)
	if isSelected {
		// For selected items, fill background to full width BUT don't add padding
		// Using MaxWidth instead of Width to prevent adding extra whitespace
		style = style.
			Background(styles.PrimaryColor).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			MaxWidth(width) // Prevent exceeding width, but don't add padding
	}

	sb.WriteString(style.Render(truncatedText))
	sb.WriteString("\n")

	// Note: no recursion here since we're using collectVisibleNodes instead
}

// renderNode recursively renders a tree node with icons and colors
func renderNode(sb *strings.Builder, node *filetree.FileNode, depth int, prefix string) {
	if node == nil {
		return
	}

	// Don't render root element (it's usually empty)
	if node.Parent == nil {
		// Render root's children
		if !node.Data.ViewInfo.Collapsed {
			sortedChildren := sortChildren(node.Children)
			for _, child := range sortedChildren {
				renderNode(sb, child, depth, "")
			}
		}
		return
	}

	// 1. Determine icon
	icon := styles.IconFile
	diffIcon := ""

	// Determine file type
	if node.Data.FileInfo.IsDir() {
		if node.Data.ViewInfo.Collapsed {
			icon = styles.IconDirClosed
		} else {
			icon = styles.IconDirOpen
		}
	} else if node.Data.FileInfo.TypeFlag == 16 { // tar.TypeSymlink
		icon = styles.IconSymlink
	}

	// Determine Diff (Added/Removed/Modified)
	color := styles.DiffNormalColor

	switch node.Data.DiffType {
	case filetree.Added:
		color = styles.DiffAddedColor
		diffIcon = styles.IconAdded
	case filetree.Removed:
		color = styles.DiffRemovedColor
		diffIcon = styles.IconRemoved
	case filetree.Modified:
		color = styles.DiffModifiedColor
		diffIcon = styles.IconModified
	}

	// 2. Build line
	name := node.Name
	if name == "" {
		name = "/"
	}

	// Add symlink target if present
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// Build line with prefix (indent)
	line := prefix + diffIcon + " " + icon + " " + name

	// Apply color
	style := lipgloss.NewStyle().Foreground(color)
	sb.WriteString(style.Render(line))
	sb.WriteString("\n")

	// 3. Recursion for children (if folder not collapsed)
	if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed && !node.IsLeaf() {
		// Calculate prefix for children
		childPrefix := prefix + "  "

		// Sort and render children
		sortedChildren := sortChildren(node.Children)
		for _, child := range sortedChildren {
			renderNode(sb, child, depth+1, childPrefix)
		}
	}
}

// sortChildren sorts node children: directories first, then files, all alphabetically
func sortChildren(children map[string]*filetree.FileNode) []*filetree.FileNode {
	if children == nil {
		return nil
	}

	// Split into directories and files
	var dirs []*filetree.FileNode
	var files []*filetree.FileNode

	for _, child := range children {
		if child.Data.FileInfo.IsDir() {
			dirs = append(dirs, child)
		} else {
			files = append(files, child)
		}
	}

	// Sort directories
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name < dirs[j].Name
	})

	// Sort files
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Combine: directories first, then files
	result := append(dirs, files...)
	return result
}
