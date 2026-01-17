package filetree

import (
	"strings"

	"github.com/wagoodman/dive/dive/filetree"
)

// FilterOptions holds the state for diff type visibility
type FilterOptions struct {
	ShowAdded      bool
	ShowRemoved    bool
	ShowModified   bool
	ShowUnmodified bool
}

// IsVisible checks if a specific node matches the active filters
func (opts FilterOptions) IsVisible(node *filetree.FileNode) bool {
	switch node.Data.DiffType {
	case filetree.Added:
		return opts.ShowAdded
	case filetree.Removed:
		return opts.ShowRemoved
	case filetree.Modified:
		return opts.ShowModified
	case filetree.Unmodified:
		return opts.ShowUnmodified
	}
	return true
}

// FilterFlatList filters a simple slice of nodes (for Flat Mode)
func FilterFlatList(nodes []VisibleNode, opts FilterOptions) []VisibleNode {
	var filtered []VisibleNode
	for _, vn := range nodes {
		if opts.IsVisible(vn.Node) {
			filtered = append(filtered, vn)
		}
	}
	return filtered
}

// CollectVisibleNodesWithFilter collects nodes for Tree View respecting visibility filters.
// A directory is shown if:
// 1. It matches the filter itself OR
// 2. It contains any visible children (recursive check)
func CollectVisibleNodesWithFilter(root *filetree.FileNode, opts FilterOptions) []VisibleNode {
	var nodes []VisibleNode

	// Recursive helper to determine if a directory HAS visible content
	// This is needed to show "Unmodified" directories that contain "Modified" files
	var hasVisibleContent func(*filetree.FileNode) bool
	hasVisibleContent = func(node *filetree.FileNode) bool {
		// If the node itself is visible per filter (e.g. Added Directory), return true
		if opts.IsVisible(node) {
			return true
		}
		// If it's a directory, check if any children are visible
		if node.Data.FileInfo.IsDir() {
			for _, child := range node.Children {
				if hasVisibleContent(child) {
					return true
				}
			}
		}
		return false
	}

	// Standard traversal helper
	var traverse func(*filetree.FileNode, []bool)
	traverse = func(node *filetree.FileNode, levels []bool) {
		if node == nil {
			return
		}

		// --- VISIBILITY CHECK ---
		// We show the node if it matches filters OR if it's a dir with visible descendants
		shouldShow := opts.IsVisible(node)
		if !shouldShow && node.Data.FileInfo.IsDir() {
			shouldShow = hasVisibleContent(node)
		}

		if !shouldShow {
			return // Skip this node entirely
		}
		// ------------------------

		// Generate tree prefix
		var prefixBuilder strings.Builder
		for i, isLast := range levels {
			if i == len(levels)-1 {
				if isLast {
					prefixBuilder.WriteString("╰─")
				} else {
					prefixBuilder.WriteString("├─")
				}
			} else {
				if isLast {
					prefixBuilder.WriteString(" ")
				} else {
					prefixBuilder.WriteString("│ ")
				}
			}
		}

		if node.Parent != nil {
			nodes = append(nodes, VisibleNode{
				Node:   node,
				Prefix: prefixBuilder.String(),
			})
		}

		// Recurse into children if directory and not collapsed
		if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed {
			sortedChildren := SortChildren(node.Children)

			// Filter children before iterating to determine "isLast" correctly
			// We only want to traverse children that will actually be shown
			var visibleChildren []*filetree.FileNode
			for _, child := range sortedChildren {
				// Peek ahead: is this child visible or does it have visible content?
				childVisible := opts.IsVisible(child)
				if !childVisible && child.Data.FileInfo.IsDir() {
					childVisible = hasVisibleContent(child)
				}

				if childVisible {
					visibleChildren = append(visibleChildren, child)
				}
			}

			count := len(visibleChildren)
			for i, child := range visibleChildren {
				newLevels := make([]bool, len(levels)+1)
				copy(newLevels, levels)
				newLevels[len(levels)] = (i == count-1)
				traverse(child, newLevels)
			}
		}
	}

	// Start traversal
	if !root.Data.ViewInfo.Collapsed {
		sortedChildren := SortChildren(root.Children)

		// Initial filtering for root children to get indentation right
		var visibleChildren []*filetree.FileNode
		for _, child := range sortedChildren {
			childVisible := opts.IsVisible(child)
			if !childVisible && child.Data.FileInfo.IsDir() {
				childVisible = hasVisibleContent(child)
			}
			if childVisible {
				visibleChildren = append(visibleChildren, child)
			}
		}

		count := len(visibleChildren)
		for i, child := range visibleChildren {
			traverse(child, []bool{i == count-1})
		}
	}

	return nodes
}
