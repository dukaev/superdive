package filetree

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// FocusStateMsg is sent by parent to tell the pane whether it's focused or not
type FocusStateMsg struct {
	Focused bool
}

// NodeToggledMsg is sent when a tree node is collapsed/expanded
type NodeToggledMsg struct {
	NodeIndex int
}

// TreeSelectionChangedMsg is sent when a tree node is selected
type TreeSelectionChangedMsg struct {
	NodeIndex int
}

// RefreshTreeContentMsg requests tree content to be refreshed
type RefreshTreeContentMsg struct {
	LayerIndex int
}

// Pane manages the file tree using bubbles/list for automatic scrolling and navigation
type Pane struct {
	focused bool
	width   int
	height  int
	treeVM  *viewmodel.FileTreeViewModel

	// list.Model handles scrolling, cursor, and viewport automatically
	list list.Model
}

// New creates a new tree pane with bubbles/list
func New(treeVM *viewmodel.FileTreeViewModel) Pane {
	// Initialize list with custom delegate
	delegate := NewTreeDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)

	// Configure list appearance
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false) // Disable bubbles pagination, we scroll ourselves

	// Custom key bindings for page navigation
	l.KeyMap.NextPage.SetKeys("pgdown", " ", "f")
	l.KeyMap.PrevPage.SetKeys("pgup", "b")

	p := Pane{
		treeVM:  treeVM,
		focused: false,
		width:   80,
		height:  20,
		list:    l,
	}

	// Build initial list items
	p.rebuildListItems()
	return p
}

// Resize updates the pane dimensions
func (p *Pane) Resize(width, height int) {
	p.width = width
	p.height = height

	// Calculate available height for the list content
	// Layout Padding: 2 (Top Border) + 2 (Bottom Border/Title gap) = 4
	// Header visual height: 1 (not layout.TreeTableHeaderHeight which is 3)
	const visualHeaderHeight = 1

	availableHeight := height - layout.BoxContentPadding - visualHeaderHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	// Update list size (handles viewport automatically)
	p.list.SetSize(width-2, availableHeight)
}

// SetTreeVM updates the tree viewmodel
func (p *Pane) SetTreeVM(treeVM *viewmodel.FileTreeViewModel) {
	p.treeVM = treeVM
	p.rebuildListItems()
	p.list.Select(0)
}

// SetTreeIndex sets the current tree index
func (p *Pane) SetTreeIndex(index int) {
	p.list.Select(index)
}

// GetTreeIndex returns the current tree index
func (p *Pane) GetTreeIndex() int {
	return p.list.Index()
}

// Init initializes the pane
func (p *Pane) Init() tea.Cmd {
	return nil
}

// SetFocused sets the focus state of the pane
func (p *Pane) SetFocused(focused bool) {
	p.focused = focused
}

// Update handles messages
func (p *Pane) Update(msg tea.Msg) (common.Pane, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case common.LayoutMsg:
		p.Resize(msg.RightWidth, msg.TreeHeight)
		return p, nil

	case FocusStateMsg:
		p.SetFocused(msg.Focused)
		return p, nil

	case common.LocalMouseMsg:
		// Handle mouse events manually since bubbles/list doesn't understand LocalMouseMsg
		if msg.Action == tea.MouseActionPress {
			// Content offsets relative to the panel:
			// Y: 1 (top border) + 1 (box title) + 1 (empty line) + 1 (table header) = 4
			// X: 1 (left border)
			const contentOffsetY = 4
			const contentOffsetX = 1

			switch msg.Button {
			case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
				// CRITICAL FIX:
				// bubbles/list has Hit Test: if Y < 0 or Y > Height, event is ignored.
				// We must pass coordinates RELATIVE TO THE LIST ITSELF (accounting for offsets),
				// otherwise scroll only works at the top of the list.
				localMsg := msg.MouseMsg
				localMsg.X = msg.LocalX - contentOffsetX
				localMsg.Y = msg.LocalY - contentOffsetY

				var cmd tea.Cmd
				p.list, cmd = p.list.Update(localMsg)
				return p, cmd

			case tea.MouseButtonLeft:
				// For clicks, use the same Y offset logic
				clickY := msg.LocalY - contentOffsetY

				// Ignore clicks on headers (negative coordinates relative to list)
				if clickY >= 0 {
					// Calculate absolute item index
					firstVisibleIndex := p.list.Index() - p.list.Cursor()
					targetIndex := firstVisibleIndex + clickY

					// Check bounds
					if targetIndex >= 0 && targetIndex < len(p.list.Items()) {
						p.list.Select(targetIndex)
						// Click selects file and toggles folder
						return p, tea.Batch(
							func() tea.Msg { return TreeSelectionChangedMsg{NodeIndex: targetIndex} },
							p.toggleCollapse(),
						)
					}
				}
			}
		}

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}

		// Handle special keys before delegating to list
		switch msg.String() {
		case "enter", "space", "right", "l":
			// Toggle folder collapse/expand
			return p, p.toggleCollapse()
		case "left", "h":
			return p, p.handleLeftKey()
		case "up", "k":
			// Let list handle navigation
		case "down", "j":
			// Let list handle navigation
		case "home", "g":
			p.list.Select(0)
			return p, nil
		case "end", "G":
			items := p.list.Items()
			if len(items) > 0 {
				p.list.Select(len(items) - 1)
			}
			return p, nil
		}

	case NodeToggledMsg, RefreshTreeContentMsg:
		p.rebuildListItems()
	}

	// Delegate all other messages to list (handles navigation, scrolling, mouse)
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

// View renders the pane
func (p Pane) View() string {
	// 1. Static table header
	header := RenderHeader(p.width)

	// 2. List view (bubbles/list renders only visible items)
	listView := p.list.View()

	// 3. Combine header and list
	fullContent := lipgloss.JoinVertical(lipgloss.Left, header, listView)

	return styles.RenderBox("Current Layer Contents", p.width, p.height, fullContent, p.focused)
}

// ========================================
// TREE OPERATIONS
// ========================================

// rebuildListItems rebuilds the list when tree structure changes
func (p *Pane) rebuildListItems() {
	if p.treeVM == nil || p.treeVM.ViewTree == nil {
		p.list.SetItems(nil)
		return
	}

	// Flatten tree structure into visible nodes
	nodes := CollectVisibleNodes(p.treeVM.ViewTree.Root)

	// Convert to list.Item slice
	items := ConvertToItems(nodes)
	p.list.SetItems(items)
}

// toggleCollapse toggles the collapsed state of the selected directory
func (p *Pane) toggleCollapse() tea.Cmd {
	item := p.list.SelectedItem()
	if item == nil {
		return nil
	}

	treeItem := item.(TreeItem)
	node := treeItem.node

	// Only directories can be collapsed/expanded
	if node.Data.FileInfo.IsDir() {
		node.Data.ViewInfo.Collapsed = !node.Data.ViewInfo.Collapsed
		p.rebuildListItems()

		// Preserve selection position if possible
		currentIndex := p.list.Index()
		if currentIndex >= 0 && currentIndex < len(p.list.Items()) {
			p.list.Select(currentIndex)
		}

		return func() tea.Msg {
			return NodeToggledMsg{NodeIndex: p.list.Index()}
		}
	}

	return nil
}

// handleLeftKey handles left arrow key behavior
func (p *Pane) handleLeftKey() tea.Cmd {
	item := p.list.SelectedItem()
	if item == nil {
		return nil
	}

	treeItem := item.(TreeItem)
	node := treeItem.node

	// If current node is an expanded directory, collapse it
	if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed {
		return p.toggleCollapse()
	}

	// Otherwise, navigate to parent directory
	if node.Parent != nil {
		items := p.list.Items()
		for i, it := range items {
			if it.(TreeItem).node == node.Parent {
				p.list.Select(i)
				return func() tea.Msg {
					return TreeSelectionChangedMsg{NodeIndex: i}
				}
			}
		}
	}

	return nil
}

// GetList returns the underlying list model
func (p *Pane) GetList() *list.Model {
	return &p.list
}
