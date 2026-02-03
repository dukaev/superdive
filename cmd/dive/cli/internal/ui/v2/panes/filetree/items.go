package filetree

import (
	"charm.land/bubbles/v2/list"
	"github.com/wagoodman/dive/dive/filetree"
)

// TreeItem wraps VisibleNode for compatibility with bubbles/list
type TreeItem struct {
	node   *filetree.FileNode
	prefix string // Tree graphic prefix (│ ├── )
}

// FilterValue returns the value for fuzzy search (built into list)
func (i TreeItem) FilterValue() string {
	return i.node.Name
}

// ID returns a unique identifier for the item
func (i TreeItem) ID() string {
	return i.node.Path()
}

// ConvertToItems converts VisibleNode slices to list.Item slices
func ConvertToItems(nodes []VisibleNode) []list.Item {
	items := make([]list.Item, len(nodes))
	for i, n := range nodes {
		items[i] = TreeItem{
			node:   n.Node,
			prefix: n.Prefix,
		}
	}
	return items
}
