package filetree

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
)

// InputHandler handles keyboard and mouse input events
type InputHandler struct {
	navigation       *Navigation
	selection        *Selection
	viewportMgr      *ViewportManager
	treeVM           *viewmodel.FileTreeViewModel
	toggleCollapseFn func() tea.Cmd // Callback for toggle collapse operation
	focused          bool
	width            int
	height           int
}

// NewInputHandler creates a new input handler
func NewInputHandler(nav *Navigation, sel *Selection, vp *ViewportManager, treeVM *viewmodel.FileTreeViewModel) *InputHandler {
	return &InputHandler{
		navigation:  nav,
		selection:   sel,
		viewportMgr: vp,
		treeVM:      treeVM,
		focused:     false,
		width:       80,
		height:      20,
	}
}

// SetFocused updates the focused state
func (h *InputHandler) SetFocused(focused bool) {
	h.focused = focused
}

// SetSize updates the dimensions
func (h *InputHandler) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// SetToggleCollapseFunc sets the callback function for toggle collapse operation
func (h *InputHandler) SetToggleCollapseFunc(fn func() tea.Cmd) {
	h.toggleCollapseFn = fn
}

// SetTreeVM updates the tree viewmodel reference
func (h *InputHandler) SetTreeVM(treeVM *viewmodel.FileTreeViewModel) {
	h.treeVM = treeVM
}

// HandleKeyPress processes keyboard input
// Returns (commands, consumed) - if consumed is true, event should not propagate to viewport
func (h *InputHandler) HandleKeyPress(msg tea.KeyMsg) (cmds []tea.Cmd, consumed bool) {
	if !h.focused {
		return nil, false
	}

	switch msg.String() {
	case "up", "k":
		return []tea.Cmd{h.navigation.MoveUp()}, true
	case "down", "j":
		return []tea.Cmd{h.navigation.MoveDown()}, true
	case "pgup":
		return []tea.Cmd{h.navigation.MovePageUp()}, true
	case "pgdown":
		return []tea.Cmd{h.navigation.MovePageDown()}, true
	case "home":
		return []tea.Cmd{h.navigation.MoveToTop()}, true
	case "end":
		return []tea.Cmd{h.navigation.MoveToBottom()}, true
	case "left", "h":
		cmd := h.navigation.MoveLeft()
		if cmd != nil {
			return []tea.Cmd{cmd}, true
		}
		return nil, true
	case "right", "l":
		cmd := h.navigation.MoveRight()
		if cmd != nil {
			return []tea.Cmd{cmd}, true
		}
		return nil, true
	case "enter", " ":
		cmd := h.toggleCollapse()
		if cmd != nil {
			return []tea.Cmd{cmd}, true
		}
		return nil, true
	}

	return nil, false
}

// HandleMouseClick processes mouse click events
func (h *InputHandler) HandleMouseClick(msg tea.MouseMsg) tea.Cmd {
	x, y := msg.X, msg.Y

	// Bounds check
	if x < 0 || x >= h.width || y < 0 {
		return nil
	}

	// CRITICAL: Account for the table header row
	const tableHeaderHeight = 1
	relativeY := y - layout.ContentVisualOffset - tableHeaderHeight

	if relativeY < 0 || relativeY >= h.viewportMgr.GetHeight() {
		return nil
	}

	if h.treeVM == nil || h.treeVM.ViewTree == nil {
		return nil
	}

	visibleNodes := CollectVisibleNodes(h.treeVM.ViewTree.Root)
	targetIndex := relativeY + h.viewportMgr.GetYOffset()

	if targetIndex >= 0 && targetIndex < len(visibleNodes) {
		// First click: just focus (move cursor)
		// Second click on same row: toggle collapse
		if h.selection.GetTreeIndex() == targetIndex {
			return h.toggleCollapse()
		}

		h.selection.MoveToIndex(targetIndex)
		h.navigation.SyncScroll()
		h.navigation.Refresh()

		return func() tea.Msg {
			return TreeSelectionChangedMsg{NodeIndex: h.selection.GetTreeIndex()}
		}
	}

	return nil
}

// toggleCollapse toggles the current node's collapse state
func (h *InputHandler) toggleCollapse() tea.Cmd {
	// Use callback if available (delegates to Pane.toggleCollapse with cached nodes)
	if h.toggleCollapseFn != nil {
		return h.toggleCollapseFn()
	}

	// Fallback to old implementation if callback not set
	if h.treeVM == nil || h.treeVM.ViewTree == nil {
		return nil
	}

	visibleNodes := CollectVisibleNodes(h.treeVM.ViewTree.Root)

	treeIndex := h.selection.GetTreeIndex()
	if treeIndex >= len(visibleNodes) {
		h.selection.MoveToIndex(len(visibleNodes) - 1)
		treeIndex = h.selection.GetTreeIndex()
	}
	if treeIndex < 0 {
		h.selection.SetTreeIndex(0)
		treeIndex = h.selection.GetTreeIndex()
	}

	if treeIndex < len(visibleNodes) {
		selectedNode := visibleNodes[treeIndex].Node

		if selectedNode.Data.FileInfo.IsDir() {
			// Toggle the collapsed flag directly on the node
			selectedNode.Data.ViewInfo.Collapsed = !selectedNode.Data.ViewInfo.Collapsed

			// Just refresh the UI - don't call treeVM.Update() as it rebuilds the tree
			h.navigation.Refresh()

			return func() tea.Msg {
				return NodeToggledMsg{NodeIndex: treeIndex}
			}
		}
	}

	return nil
}
