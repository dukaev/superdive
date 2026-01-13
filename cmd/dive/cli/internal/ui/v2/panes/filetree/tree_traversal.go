package filetree

import (
	"sort"
	"strings"

	"github.com/wagoodman/dive/dive/filetree"
)

// VisibleNode represents a node with its tree prefix for rendering
type VisibleNode struct {
	Node   *filetree.FileNode
	Prefix string // Tree guide prefix, e.g. "│   ├── ", "└── "
}

// CollectVisibleNodes collects all visible nodes with tree guide prefixes
func CollectVisibleNodes(root *filetree.FileNode) []VisibleNode {
	var nodes []VisibleNode

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
					prefixBuilder.WriteString("  ") // Was "    "
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
			traverse(child, []bool{i == count - 1})
		}
	}

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
	result := append(dirs, files...)
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
