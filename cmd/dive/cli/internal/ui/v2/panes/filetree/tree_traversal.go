package filetree

import (
	"regexp"
	"sort"
	"strings"

	"github.com/wagoodman/dive/dive/filetree"
)

// VisibleNode represents a node with its tree prefix for rendering
type VisibleNode struct {
	Node        *filetree.FileNode
	Prefix      string // Tree guide prefix, e.g. "│   ├── ", "└── "
	DisplayName string // Display name (empty for tree view, full path for flat view)
}

// CollectVisibleNodes collects all visible nodes with tree guide prefixes
func CollectVisibleNodes(root *filetree.FileNode) []VisibleNode {
	nodes := make([]VisibleNode, 0, 100)

	// levels tracks state for each nesting level:
	// true = this level is the last child (use spaces)
	// false = more children follow (use │)
	var traverse func(*filetree.FileNode, []bool)

	traverse = func(node *filetree.FileNode, levels []bool) {
		if node == nil {
			return
		}

		// Generate tree prefix based on levels (compact, 2 chars per level)
		// Example levels: [false, true] -> "│ └─"
		var prefixBuilder strings.Builder
		for i, isLast := range levels {
			if i == len(levels)-1 {
				// Current level (the node itself) - 2 chars
				if isLast {
					prefixBuilder.WriteString("└─") // Was "└── "
				} else {
					prefixBuilder.WriteString("├─") // Was "├── "
				}
			} else {
				// Parent levels (indentation) - 2 chars
				if isLast {
					prefixBuilder.WriteString(" ") // Was "    "
				} else {
					prefixBuilder.WriteString("│ ") // Was "│   "
				}
			}
		}

		// Add current node (skip root when rendering)
		if node.Parent != nil {
			nodes = append(nodes, VisibleNode{
				Node:   node,
				Prefix: prefixBuilder.String(),
			})
		}

		// Recurse into children if directory and not collapsed
		if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed {
			sortedChildren := SortChildren(node.Children)
			count := len(sortedChildren)
			for i, child := range sortedChildren {
				// Create new levels array for child
				isLastChild := i == count-1
				newLevels := make([]bool, len(levels)+1)
				copy(newLevels, levels)
				newLevels[len(levels)] = isLastChild

				traverse(child, newLevels)
			}
		}
	}

	// Start from root
	if !root.Data.ViewInfo.Collapsed {
		sortedChildren := SortChildren(root.Children)
		count := len(sortedChildren)
		for i, child := range sortedChildren {
			traverse(child, []bool{i == count-1})
		}
	}

	return nodes
}

// CollectFlatNodes collects all nodes as a flat list sorted by path
func CollectFlatNodes(root *filetree.FileNode) []VisibleNode {
	nodes := make([]VisibleNode, 0, 100)

	var traverse func(*filetree.FileNode)
	traverse = func(node *filetree.FileNode) {
		if node == nil {
			return
		}

		// Add node (except root, which has no parent)
		if node.Parent != nil {
			nodes = append(nodes, VisibleNode{
				Node:        node,
				Prefix:      "",          // No prefix in flat mode
				DisplayName: node.Path(), // Use full path
			})
		}

		// Recursively process all children (ignore collapsed state)
		sortedChildren := SortChildren(node.Children)
		for _, child := range sortedChildren {
			traverse(child)
		}
	}

	traverse(root)
	return nodes
}

// CollectSearchResults collects ONLY nodes that match the filter regex
// This implements "Google-style" clean search: no parent directories shown
// unless they themselves match the search pattern
func CollectSearchResults(root *filetree.FileNode, filter *regexp.Regexp) []VisibleNode {
	nodes := make([]VisibleNode, 0, 100)

	var traverse func(*filetree.FileNode)
	traverse = func(node *filetree.FileNode) {
		if node == nil {
			return
		}

		// Skip root node (it has no parent and represents the filesystem root)
		if node.Parent != nil {
			// Check if this node matches the filter
			// We use the full path for matching (e.g., "/etc/fstab")
			if filter.MatchString(node.Path()) {
				nodes = append(nodes, VisibleNode{
					Node:        node,
					Prefix:      "",          // No tree prefix in flat mode
					DisplayName: node.Path(), // Show full path
				})
			}
		}

		// IMPORTANT: Always recurse into children, even if current node doesn't match
		// This allows finding matches deep in the directory tree
		// Use ModelTree (not ViewTree) to search ALL files, not just visible ones
		sortedChildren := SortChildren(node.Children)
		for _, child := range sortedChildren {
			traverse(child)
		}
	}

	traverse(root)
	return nodes
}

// SortChildren sorts node children: directories first, then files, all alphabetically
func SortChildren(children map[string]*filetree.FileNode) []*filetree.FileNode {
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
	result := make([]*filetree.FileNode, 0, len(dirs)+len(files))
	result = append(result, dirs...)
	result = append(result, files...)
	return result
}

// FindParentIndex finds the index of the parent directory of the node at the given index
// Returns -1 if the node has no parent or parent is not visible
func FindParentIndex(visibleNodes []VisibleNode, currentIndex int) int {
	if currentIndex < 0 || currentIndex >= len(visibleNodes) {
		return -1
	}

	currentNode := visibleNodes[currentIndex].Node
	parentNode := currentNode.Parent

	// Root node has no parent
	if parentNode == nil || parentNode.Parent == nil {
		// parent.Parent == nil means parent is actually the root node
		return -1
	}

	// Find the parent in the visible nodes
	for i, vn := range visibleNodes {
		if vn.Node == parentNode {
			return i
		}
	}

	// Parent exists but is not visible (e.g., collapsed ancestor)
	return -1
}
