package filetree

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

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

// Pane manages the file tree
type Pane struct {
	focused   bool
	width     int
	height    int
	treeVM    *viewmodel.FileTreeViewModel

	// Components
	selection   *Selection
	viewportMgr *ViewportManager
	navigation  *Navigation
	inputHandler *InputHandler
}

// New creates a new tree pane
func New(treeVM *viewmodel.FileTreeViewModel) Pane {
	// Initialize components
	selection := NewSelection()
	viewportMgr := NewViewportManager(80, 20)
	navigation := NewNavigation(selection, viewportMgr)
	inputHandler := NewInputHandler(navigation, selection, viewportMgr, treeVM)

	p := Pane{
		treeVM:       treeVM,
		selection:    selection,
		viewportMgr:  viewportMgr,
		navigation:   navigation,
		inputHandler: inputHandler,
		focused:      false,
		width:        80,
		height:       20,
	}

	// Set up callbacks
	p.navigation.SetVisibleNodesFunc(func() []VisibleNode {
		if p.treeVM == nil || p.treeVM.ViewTree == nil {
			return nil
		}
		return CollectVisibleNodes(p.treeVM.ViewTree.Root)
	})

	p.navigation.SetRefreshFunc(p.updateContent)
	p.navigation.SetToggleCollapseFunc(p.toggleCollapse)

	// IMPORTANT: Generate content immediately so viewport is not empty on startup
	p.updateContent()
	return p
}

// SetSize updates the pane dimensions
func (m *Pane) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.inputHandler.SetSize(width, height)

	viewportWidth := width - 2
	viewportHeight := height - layout.BoxContentPadding
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewportMgr.SetSize(viewportWidth, viewportHeight)

	// CRITICAL: Regenerate content with new width to prevent soft wrap
	// Without this, long paths will wrap when window is resized
	m.updateContent()
}

// SetTreeVM updates the tree viewmodel
func (m *Pane) SetTreeVM(treeVM *viewmodel.FileTreeViewModel) {
	m.treeVM = treeVM
	m.selection.SetTreeIndex(0)
	m.viewportMgr.GotoTop()
	m.updateContent()
}

// SetTreeIndex sets the current tree index
func (m *Pane) SetTreeIndex(index int) {
	m.selection.SetTreeIndex(index)
	m.navigation.SyncScroll()
}

// GetTreeIndex returns the current tree index
func (m *Pane) GetTreeIndex() int {
	return m.selection.GetTreeIndex()
}

// Focus sets the pane as active
func (m *Pane) Focus() {
	m.focused = true
	m.inputHandler.SetFocused(true)
}

// Blur sets the pane as inactive
func (m *Pane) Blur() {
	m.focused = false
	m.inputHandler.SetFocused(false)
}

// IsFocused returns true if the pane is focused
func (m *Pane) IsFocused() bool {
	return m.focused
}

// Init initializes the pane
func (m Pane) Init() tea.Cmd {
	m.updateContent()
	return nil
}

// Update handles messages
func (m Pane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmds, consumed := m.inputHandler.HandleKeyPress(msg)
		if consumed {
			// Don't pass to viewport
			return m, tea.Batch(cmds...)
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			var keyCmds []tea.Cmd

			if msg.Button == tea.MouseButtonWheelUp {
				keyCmds = append(keyCmds, m.navigation.MoveUp())
			} else if msg.Button == tea.MouseButtonWheelDown {
				keyCmds = append(keyCmds, m.navigation.MoveDown())
			}

			if msg.Button == tea.MouseButtonLeft {
				if cmd := m.inputHandler.HandleMouseClick(msg); cmd != nil {
					keyCmds = append(keyCmds, cmd)
				}
			}

			if len(keyCmds) > 0 {
				return m, tea.Batch(keyCmds...)
			}
		}

	case RefreshTreeContentMsg:
		m.updateContent()
	}

	// Always update viewport
	_, cmd := m.viewportMgr.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the pane
func (m Pane) View() string {
	// 1. Generate static header
	header := RenderHeader(m.width)

	// 2. Get viewport content
	content := m.viewportMgr.GetViewport().View()

	// 3. Combine: Header + Content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	return styles.RenderBox("Current Layer Contents", m.width, m.height, fullContent, m.focused)
}

// updateContent regenerates the viewport content
func (m *Pane) updateContent() {
	if m.treeVM == nil {
		m.viewportMgr.SetContent("No tree data")
		return
	}

	content := m.renderTreeContent()
	if content == "" {
		content = "(File tree rendering in progress...)"
	}
	m.viewportMgr.SetContent(content)
}

// renderTreeContent generates the tree content
func (m *Pane) renderTreeContent() string {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return "No tree data"
	}

	var sb strings.Builder
	visibleNodes := CollectVisibleNodes(m.treeVM.ViewTree.Root)
	viewportWidth := m.viewportMgr.GetViewport().Width

	for i, vn := range visibleNodes {
		isSelected := (i == m.selection.GetTreeIndex())
		RenderNodeWithCursor(&sb, vn.Node, vn.Prefix, isSelected, viewportWidth)
	}

	return sb.String()
}

// toggleCollapse toggles the current node's collapse state
func (m *Pane) toggleCollapse() tea.Cmd {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return nil
	}

	visibleNodes := CollectVisibleNodes(m.treeVM.ViewTree.Root)

	treeIndex := m.selection.GetTreeIndex()
	if treeIndex >= len(visibleNodes) {
		m.selection.MoveToIndex(len(visibleNodes) - 1)
		treeIndex = m.selection.GetTreeIndex()
	}
	if treeIndex < 0 {
		m.selection.SetTreeIndex(0)
		treeIndex = m.selection.GetTreeIndex()
	}

	if treeIndex < len(visibleNodes) {
		selectedNode := visibleNodes[treeIndex].Node

		if selectedNode.Data.FileInfo.IsDir() {
			// Toggle the collapsed flag directly on the node
			selectedNode.Data.ViewInfo.Collapsed = !selectedNode.Data.ViewInfo.Collapsed

			// Just refresh the UI - don't call treeVM.Update() as it rebuilds the tree
			m.updateContent()

			return func() tea.Msg {
				return NodeToggledMsg{NodeIndex: treeIndex}
			}
		}
	}

	return nil
}

// GetViewport returns the underlying viewport
func (m *Pane) GetViewport() *viewport.Model {
	return m.viewportMgr.GetViewport()
}
