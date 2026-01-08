package filetree

import (
	"archive/tar"
	"fmt"
	"github.com/wagoodman/dive/internal/log"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/phayes/permbits"
)

const (
	AttributeFormat = "%s%s %11s %10s "
)

var diffTypeColor = map[DiffType]*color.Color{
	Added:      color.New(color.FgGreen),
	Removed:    color.New(color.FgRed),
	Modified:   color.New(color.FgYellow),
	Unmodified: color.New(color.Reset),
}

// FileNode represents a single file, its relation to files beneath it, the tree it exists in, and the metadata of the given file.
type FileNode struct {
	Tree     *FileTree
	Parent   *FileNode
	Size     int64 // memoized total size of file or directory
	Name     string
	Data     NodeData
	Children map[string]*FileNode
	path     string // OPTIMIZATION: Cached path (computed once, reused many times in CI mode)
}

// NewNode creates a new FileNode relative to the given parent node with a payload.
// OPTIMIZATION: Uses struct literal and does NOT allocate Children map (lazy initialization).
func NewNode(parent *FileNode, name string, data FileInfo) *FileNode {
	var tree *FileTree
	if parent != nil {
		tree = parent.Tree
	}

	// Create object with struct literal to avoid extra allocations and assignments
	return &FileNode{
		Tree:   tree,
		Parent: parent,
		Size:   -1, // signal lazy load later
		Name:   name,
		// Initialize Data directly, avoiding NewNodeData() call and extra struct copying
		Data: NodeData{
			FileInfo: *data.Copy(),
			// DiffType defaults to Unmodified (0), explicit initialization not needed
		},
		// Children: nil, // Explicitly leave nil for memory savings (lazy initialization)
	}
}

// renderTreeLine returns a string representing this FileNode in the context of a greater ASCII tree.
func (node *FileNode) renderTreeLine(spaces []bool, last bool, collapsed bool) string {
	var otherBranches string
	for _, space := range spaces {
		if space {
			otherBranches += noBranchSpace
		} else {
			otherBranches += branchSpace
		}
	}

	thisBranch := middleItem
	if last {
		thisBranch = lastItem
	}

	collapsedIndicator := uncollapsedItem
	if collapsed {
		collapsedIndicator = collapsedItem
	}

	return otherBranches + thisBranch + collapsedIndicator + node.String() + newLine
}

// Copy duplicates the existing node relative to a new parent node.
// OPTIMIZATION: Pre-allocation of Children map with correct size.
func (node *FileNode) Copy(parent *FileNode) *FileNode {
	newNode := NewNode(parent, node.Name, node.Data.FileInfo)
	newNode.Data.ViewInfo = node.Data.ViewInfo
	newNode.Data.DiffType = node.Data.DiffType

	// If source node has children, initialize map with correct capacity upfront
	if len(node.Children) > 0 {
		newNode.Children = make(map[string]*FileNode, len(node.Children))
		for name, child := range node.Children {
			// Recursively copy children
			newNode.Children[name] = child.Copy(newNode)
		}
	}
	return newNode
}

// AddChild creates a new node relative to the current FileNode.
// OPTIMIZATION: Lazy initialization of Children map only when needed.
func (node *FileNode) AddChild(name string, data FileInfo) *FileNode {
	// never allow processing of purely whiteout flag files (for now)
	if strings.HasPrefix(name, doubleWhiteoutPrefix) {
		return nil
	}

	// 1. Lazy initialization: create map only when first child is added
	if node.Children == nil {
		node.Children = make(map[string]*FileNode)
	}

	// 2. Use "ok" idiom for existence check (faster and safer)
	if existingNode, ok := node.Children[name]; ok {
		// Node already exists, just update the data
		existingNode.Data.FileInfo = *data.Copy()
		return existingNode // Return existing node to avoid duplicates
	}

	// 3. Create new node and add to tree
	child := NewNode(node, name, data)
	node.Children[name] = child
	node.Tree.Size++

	return child
}

// Remove deletes the current FileNode from it's parent FileNode's relations.
func (node *FileNode) Remove() error {
	if node == node.Tree.Root {
		return fmt.Errorf("cannot remove the tree root")
	}
	for _, child := range node.Children {
		err := child.Remove()
		if err != nil {
			return err
		}
	}
	delete(node.Parent.Children, node.Name)
	node.Tree.Size--
	return nil
}

// String shows the filename formatted into the proper color (by DiffType), additionally indicating if it is a symlink.
func (node *FileNode) String() string {
	var display string
	if node == nil {
		return ""
	}

	display = node.Name
	if node.Data.FileInfo.TypeFlag == tar.TypeSymlink || node.Data.FileInfo.TypeFlag == tar.TypeLink {
		display += " → " + node.Data.FileInfo.Linkname
	}
	return diffTypeColor[node.Data.DiffType].Sprint(display)
}

// MetadatString returns the FileNode metadata in a columnar string.
func (node *FileNode) MetadataString() string {
	if node == nil {
		return ""
	}

	dir := "-"
	if node.Data.FileInfo.IsDir {
		dir = "d"
	}

	fm := permbits.FileMode(node.Data.FileInfo.Mode)
	var fileMode strings.Builder
	fileMode.Grow(9)
	cond := func(c bool, x, y byte) byte {
		if c {
			return x
		} else {
			return y
		}
	}
	fileMode.WriteByte(cond(fm.UserRead(), 'r', '-'))
	fileMode.WriteByte(cond(fm.UserWrite(), 'w', '-'))
	fileMode.WriteByte(cond(fm.UserExecute(), cond(fm.Setuid(), 's', 'x'), cond(fm.Setuid(), 'S', '-')))

	fileMode.WriteByte(cond(fm.GroupRead(), 'r', '-'))
	fileMode.WriteByte(cond(fm.GroupWrite(), 'w', '-'))
	fileMode.WriteByte(cond(fm.GroupExecute(), cond(fm.Setgid(), 's', 'x'), cond(fm.Setgid(), 'S', '-')))

	fileMode.WriteByte(cond(fm.OtherRead(), 'r', '-'))
	fileMode.WriteByte(cond(fm.OtherWrite(), 'w', '-'))
	fileMode.WriteByte(cond(fm.OtherExecute(), cond(fm.Sticky(), 't', 'x'), cond(fm.Sticky(), 'T', '-')))

	user := node.Data.FileInfo.Uid
	group := node.Data.FileInfo.Gid
	userGroup := fmt.Sprintf("%d:%d", user, group)

	// don't include file sizes of children that have been removed (unless the node in question is a removed dir,
	// then show the accumulated size of removed files)
	sizeBytes := node.GetSize()

	size := humanize.Bytes(uint64(sizeBytes))

	return diffTypeColor[node.Data.DiffType].Sprint(fmt.Sprintf(AttributeFormat, dir, fileMode.String(), userGroup, size))
}

func (node *FileNode) GetSize() int64 {
	if 0 <= node.Size {
		return node.Size
	}
	var sizeBytes int64

	if node.IsLeaf() {
		sizeBytes = node.Data.FileInfo.Size
	} else {
		sizer := func(curNode *FileNode) error {

			if curNode.Data.DiffType != Removed || node.Data.DiffType == Removed {
				sizeBytes += curNode.Data.FileInfo.Size
			}
			return nil
		}
		err := node.VisitDepthChildFirst(sizer, nil, nil)
		if err != nil {
			log.WithFields("error", err).Debug("unable to propagate tree to get file size")
		}
	}
	node.Size = sizeBytes
	return node.Size
}

// VisitDepthChildFirst iterates a tree depth-first (starting at this FileNode), evaluating the deepest depths first (visit on bubble up)
func (node *FileNode) VisitDepthChildFirst(visitor Visitor, evaluator VisitEvaluator, sorter OrderStrategy) error {
	if sorter == nil {
		sorter = GetSortOrderStrategy(ByName)
	}
	keys := sorter.orderKeys(node.Children)
	for _, name := range keys {
		child := node.Children[name]
		err := child.VisitDepthChildFirst(visitor, evaluator, sorter)
		if err != nil {
			return err
		}
	}
	// never visit the root node
	if node == node.Tree.Root {
		return nil
	} else if evaluator != nil && evaluator(node) || evaluator == nil {
		return visitor(node)
	}

	return nil
}

// VisitDepthParentFirst iterates a tree depth-first (starting at this FileNode), evaluating the shallowest depths first (visit while sinking down)
func (node *FileNode) VisitDepthParentFirst(visitor Visitor, evaluator VisitEvaluator, sorter OrderStrategy) error {
	var err error

	doVisit := evaluator != nil && evaluator(node) || evaluator == nil

	if !doVisit {
		return nil
	}

	// never visit the root node
	if node != node.Tree.Root {
		err = visitor(node)
		if err != nil {
			return err
		}
	}

	if sorter == nil {
		sorter = GetSortOrderStrategy(ByName)
	}
	keys := sorter.orderKeys(node.Children)
	for _, name := range keys {
		child := node.Children[name]
		err = child.VisitDepthParentFirst(visitor, evaluator, sorter)
		if err != nil {
			return err
		}
	}
	return err
}

// IsWhiteout returns an indication if this file may be a overlay-whiteout file.
func (node *FileNode) IsWhiteout() bool {
	return strings.HasPrefix(node.Name, whiteoutPrefix)
}

// IsLeaf returns true is the current node has no child nodes.
func (node *FileNode) IsLeaf() bool {
	// Map is nil or empty - this is a leaf node
	return node.Children == nil || len(node.Children) == 0
}

// Path returns a slash-delimited string from the root of the greater tree to the current node (e.g. /a/path/to/here)
// OPTIMIZATION: Uses caching with lazy evaluation.
// Path is computed once and cached, then reused for subsequent calls.
// This is beneficial for CI mode where Path() is called frequently during comparison.
func (node *FileNode) Path() string {
	if node.path == "" {
		// Pre-allocate slice for path segments (capacity 10 covers most cases)
		segments := make([]string, 0, 10)

		// Walk up the tree collecting names
		curNode := node
		for curNode.Parent != nil {
			name := curNode.Name
			if curNode == node {
				// white out prefixes are fictitious on leaf nodes
				name = strings.TrimPrefix(name, whiteoutPrefix)
			}
			// Append in reverse order (will reverse later)
			segments = append(segments, name)
			curNode = curNode.Parent
		}

		// Reverse the slice (O(n) but very cheap)
		for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
			segments[i], segments[j] = segments[j], segments[i]
		}

		// Build and cache final path string
		node.path = "/" + strings.Join(segments, "/")
	}
	return node.path
}

// deriveDiffType determines a DiffType to the current FileNode. Note: the DiffType of a node is always the DiffType of
// its attributes and its contents. The contents are the bytes of the file of the children of a directory.
func (node *FileNode) deriveDiffType(diffType DiffType) error {
	if node.IsLeaf() {
		return node.AssignDiffType(diffType)
	}

	myDiffType := diffType
	for _, v := range node.Children {
		myDiffType = myDiffType.merge(v.Data.DiffType)
	}

	return node.AssignDiffType(myDiffType)
}

// AssignDiffType will assign the given DiffType to this node, possibly affecting child nodes.
func (node *FileNode) AssignDiffType(diffType DiffType) error {
	var err error

	node.Data.DiffType = diffType

	if diffType == Removed {
		// if we've removed this node, then all children have been removed as well
		for _, child := range node.Children {
			err = child.AssignDiffType(diffType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// compare the current node against the given node, returning a definitive DiffType.
func (node *FileNode) compare(other *FileNode) DiffType {
	if node == nil && other == nil {
		return Unmodified
	}

	if node == nil && other != nil {
		return Added
	}

	if node != nil && other == nil {
		return Removed
	}

	if other.IsWhiteout() {
		return Removed
	}
	if node.Name != other.Name {
		panic("comparing mismatched nodes")
	}

	return node.Data.FileInfo.Compare(other.Data.FileInfo)
}
