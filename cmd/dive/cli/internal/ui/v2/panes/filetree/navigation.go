package filetree

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Navigation handles tree navigation movements
type Navigation struct {
	selection       *Selection
	viewportMgr     *ViewportManager
	visibleNodesFn  func() []VisibleNode
	refreshFn       func()
	toggleCollapseFn func() tea.Cmd
}

// NewNavigation creates a new navigation handler
func NewNavigation(selection *Selection, viewportMgr *ViewportManager) *Navigation {
	return &Navigation{
		selection:   selection,
		viewportMgr: viewportMgr,
	}
}

// SetVisibleNodesFunc sets the callback to get visible nodes
func (n *Navigation) SetVisibleNodesFunc(fn func() []VisibleNode) {
	n.visibleNodesFn = fn
}

// SetRefreshFunc sets the callback to refresh content
func (n *Navigation) SetRefreshFunc(fn func()) {
	n.refreshFn = fn
}

// SetToggleCollapseFunc sets the callback to toggle directory collapse state
func (n *Navigation) SetToggleCollapseFunc(fn func() tea.Cmd) {
	n.toggleCollapseFn = fn
}

// MoveUp moves selection up
func (n *Navigation) MoveUp() tea.Cmd {
	if n.selection.GetTreeIndex() > 0 {
		n.selection.SetTreeIndex(n.selection.GetTreeIndex() - 1)
		n.SyncScroll()
		n.doRefresh()
	}
	return nil
}

// MoveDown moves selection down
func (n *Navigation) MoveDown() tea.Cmd {
	if n.visibleNodesFn == nil {
		return nil
	}

	visibleNodes := n.visibleNodesFn()
	if n.selection.GetTreeIndex() < len(visibleNodes)-1 {
		n.selection.SetTreeIndex(n.selection.GetTreeIndex() + 1)
		n.SyncScroll()
		n.doRefresh()
	}
	return nil
}

// MovePageUp moves selection up by one page
func (n *Navigation) MovePageUp() tea.Cmd {
	if n.visibleNodesFn == nil {
		return nil
	}

	// Move up by viewport height
	pageSize := n.viewportMgr.GetHeight()
	if pageSize < 1 {
		pageSize = 10
	}

	newIndex := n.selection.GetTreeIndex() - pageSize
	if newIndex < 0 {
		newIndex = 0
	}
	n.selection.MoveToIndex(newIndex)
	n.SyncScroll()
	n.doRefresh()
	return nil
}

// MovePageDown moves selection down by one page
func (n *Navigation) MovePageDown() tea.Cmd {
	if n.visibleNodesFn == nil {
		return nil
	}

	visibleNodes := n.visibleNodesFn()
	if len(visibleNodes) == 0 {
		return nil
	}

	// Move down by viewport height
	pageSize := n.viewportMgr.GetHeight()
	if pageSize < 1 {
		pageSize = 10
	}

	newIndex := n.selection.GetTreeIndex() + pageSize
	if newIndex >= len(visibleNodes) {
		newIndex = len(visibleNodes) - 1
	}
	n.selection.MoveToIndex(newIndex)
	n.SyncScroll()
	n.doRefresh()
	return nil
}

// MoveToTop moves selection to the first item
func (n *Navigation) MoveToTop() tea.Cmd {
	n.selection.SetTreeIndex(0)
	n.viewportMgr.GotoTop()
	n.SyncScroll()
	n.doRefresh()
	return nil
}

// MoveToBottom moves selection to the last item
func (n *Navigation) MoveToBottom() tea.Cmd {
	if n.visibleNodesFn == nil {
		return nil
	}

	visibleNodes := n.visibleNodesFn()
	if len(visibleNodes) == 0 {
		return nil
	}

	n.selection.MoveToIndex(len(visibleNodes) - 1)
	n.viewportMgr.GotoBottom()
	n.SyncScroll()
	n.doRefresh()
	return nil
}

// SyncScroll ensures the cursor is always visible
func (n *Navigation) SyncScroll() {
	if n.visibleNodesFn == nil {
		return
	}

	visibleNodes := n.visibleNodesFn()
	if len(visibleNodes) == 0 {
		return
	}

	n.selection.SetMaxIndex(len(visibleNodes))
	n.selection.ValidateBounds()

	visibleHeight := n.viewportMgr.GetHeight()
	if visibleHeight <= 0 {
		return
	}

	treeIndex := n.selection.GetTreeIndex()
	yOffset := n.viewportMgr.GetYOffset()

	if treeIndex < yOffset {
		n.viewportMgr.SetYOffset(treeIndex)
	}

	if treeIndex >= yOffset+visibleHeight {
		n.viewportMgr.SetYOffset(treeIndex - visibleHeight + 1)
	}
}

// doRefresh calls the refresh callback if set
func (n *Navigation) doRefresh() {
	if n.refreshFn != nil {
		n.refreshFn()
	}
}

// Refresh is a public method to trigger content refresh
func (n *Navigation) Refresh() {
	n.doRefresh()
}

// MoveLeft navigates to parent directory or collapses current directory
func (n *Navigation) MoveLeft() tea.Cmd {
	if n.visibleNodesFn == nil {
		return nil
	}

	visibleNodes := n.visibleNodesFn()
	if len(visibleNodes) == 0 {
		return nil
	}

	currentIndex := n.selection.GetTreeIndex()
	if currentIndex < 0 || currentIndex >= len(visibleNodes) {
		return nil
	}

	currentNode := visibleNodes[currentIndex].Node

	// If current node is an expanded directory, collapse it
	if currentNode.Data.FileInfo.IsDir() && !currentNode.Data.ViewInfo.Collapsed {
		if n.toggleCollapseFn != nil {
			return n.toggleCollapseFn()
		}
		return nil
	}

	// Otherwise, move to parent directory (for files or collapsed dirs)
	parentIndex := FindParentIndex(visibleNodes, currentIndex)
	if parentIndex >= 0 {
		n.selection.MoveToIndex(parentIndex)
		n.SyncScroll()
		n.doRefresh()
	}

	return nil
}

// MoveRight expands collapsed directory
func (n *Navigation) MoveRight() tea.Cmd {
	if n.visibleNodesFn == nil {
		return nil
	}

	visibleNodes := n.visibleNodesFn()
	if len(visibleNodes) == 0 {
		return nil
	}

	currentIndex := n.selection.GetTreeIndex()
	if currentIndex < 0 || currentIndex >= len(visibleNodes) {
		return nil
	}

	currentNode := visibleNodes[currentIndex].Node

	// If current node is a collapsed directory, expand it
	if currentNode.Data.FileInfo.IsDir() && currentNode.Data.ViewInfo.Collapsed {
		if n.toggleCollapseFn != nil {
			return n.toggleCollapseFn()
		}
	}

	return nil
}
