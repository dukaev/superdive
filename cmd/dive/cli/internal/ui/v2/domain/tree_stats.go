package domain

import (
	"github.com/wagoodman/dive/dive/filetree"
)

// FileStats holds file change statistics
type FileStats struct {
	Added    int
	Modified int
	Removed  int
}

// CalculateFileStats walks the file tree and counts file changes.
// This is a pure function that extracts business logic from the UI layer.
func CalculateFileStats(tree *filetree.FileTree) FileStats {
	stats := FileStats{}

	if tree == nil || tree.Root == nil {
		return stats
	}

	visitor := func(node *filetree.FileNode) error {
		// Only count leaf nodes (actual files, not directories)
		if len(node.Children) == 0 {
			switch node.Data.DiffType {
			case filetree.Added:
				stats.Added++
			case filetree.Modified:
				stats.Modified++
			case filetree.Removed:
				stats.Removed++
			}
		}
		return nil
	}

	_ = tree.VisitDepthChildFirst(visitor, nil)
	return stats
}
