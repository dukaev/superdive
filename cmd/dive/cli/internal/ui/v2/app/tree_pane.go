package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// TreePane manages the file tree
type TreePane struct {
	focused   bool
	width     int
	height    int
	treeVM    *viewmodel.FileTreeViewModel
	viewport  viewport.Model
	treeIndex int
}

// NewTreePane creates a new tree pane
func NewTreePane(treeVM *viewmodel.FileTreeViewModel) TreePane {
	vp := viewport.New(80, 20)
	p := TreePane{
		treeVM:    treeVM,
		viewport:  vp,
		treeIndex: 0,
		width:     80,
		height:    20,
	}
	// IMPORTANT: Generate content immediately so viewport is not empty on startup
	p.updateContent()
	return p
}

// SetSize updates the pane dimensions
func (m *TreePane) SetSize(width, height int) {
	m.width = width
	m.height = height

	viewportWidth := width - 2
	viewportHeight := height - BoxContentPadding
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Width = viewportWidth
	m.viewport.Height = viewportHeight
}

// SetTreeVM updates the tree viewmodel
func (m *TreePane) SetTreeVM(treeVM *viewmodel.FileTreeViewModel) {
	m.treeVM = treeVM
	m.treeIndex = 0
	m.viewport.GotoTop()
	m.updateContent()
}

// SetTreeIndex sets the current tree index
func (m *TreePane) SetTreeIndex(index int) {
	m.treeIndex = index
	m.syncScroll()
}

// GetTreeIndex returns the current tree index
func (m *TreePane) GetTreeIndex() int {
	return m.treeIndex
}

// Focus sets the pane as active
func (m *TreePane) Focus() {
	m.focused = true
}

// Blur sets the pane as inactive
func (m *TreePane) Blur() {
	m.focused = false
}

// IsFocused returns true if the pane is focused
func (m *TreePane) IsFocused() bool {
	return m.focused
}

// Init initializes the pane
func (m TreePane) Init() tea.Cmd {
	m.updateContent()
	return nil
}

// Update handles messages
func (m TreePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			cmds = append(cmds, m.moveUp())
		case "down", "j":
			cmds = append(cmds, m.moveDown())
		case "enter", " ":
			cmds = append(cmds, m.toggleCollapse())
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			if msg.Button == tea.MouseButtonWheelUp {
				cmds = append(cmds, m.moveUp())
			} else if msg.Button == tea.MouseButtonWheelDown {
				cmds = append(cmds, m.moveDown())
			}

			if msg.Button == tea.MouseButtonLeft {
				if cmd := m.handleClick(msg.X, msg.Y); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case RefreshTreeContentMsg:
		m.updateContent()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the pane
func (m TreePane) View() string {
	content := m.viewport.View()
	return v2styles.RenderBox("Current Layer Contents", m.width, m.height, content, m.focused)
}

// moveUp moves selection up
func (m *TreePane) moveUp() tea.Cmd {
	if m.treeIndex > 0 {
		m.treeIndex--
		m.syncScroll()
	}
	return nil
}

// moveDown moves selection down
func (m *TreePane) moveDown() tea.Cmd {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return nil
	}

	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)
	if m.treeIndex < len(visibleNodes)-1 {
		m.treeIndex++
		m.syncScroll()
	}
	return nil
}

// toggleCollapse toggles the current node's collapse state
func (m *TreePane) toggleCollapse() tea.Cmd {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return nil
	}

	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)

	if m.treeIndex >= len(visibleNodes) {
		m.treeIndex = len(visibleNodes) - 1
	}
	if m.treeIndex < 0 {
		m.treeIndex = 0
	}

	if m.treeIndex < len(visibleNodes) {
		selectedNode := visibleNodes[m.treeIndex].Node

		if selectedNode.Data.FileInfo.IsDir() {
			selectedNode.Data.ViewInfo.Collapsed = !selectedNode.Data.ViewInfo.Collapsed
			_ = m.treeVM.Update(nil, m.width, m.height)
			m.updateContent()

			return func() tea.Msg {
				return NodeToggledMsg{NodeIndex: m.treeIndex}
			}
		}
	}

	return nil
}

// handleClick processes a mouse click
func (m *TreePane) handleClick(x, y int) tea.Cmd {
	if x < 0 || x >= m.width || y < 0 {
		return nil
	}

	relativeY := y - ContentVisualOffset
	if relativeY < 0 || relativeY >= m.viewport.Height {
		return nil
	}

	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return nil
	}

	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)
	targetIndex := relativeY + m.viewport.YOffset

	if targetIndex >= 0 && targetIndex < len(visibleNodes) {
		if m.treeIndex == targetIndex {
			return m.toggleCollapse()
		} else {
			m.treeIndex = targetIndex
			m.syncScroll()
			return func() tea.Msg {
				return TreeSelectionChangedMsg{NodeIndex: m.treeIndex}
			}
		}
	}

	return nil
}

// syncScroll ensures the cursor is always visible
func (m *TreePane) syncScroll() {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return
	}

	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)
	if len(visibleNodes) == 0 {
		return
	}

	if m.treeIndex >= len(visibleNodes) {
		m.treeIndex = len(visibleNodes) - 1
	}
	if m.treeIndex < 0 {
		m.treeIndex = 0
	}

	visibleHeight := m.viewport.Height
	if visibleHeight <= 0 {
		return
	}

	if m.treeIndex < m.viewport.YOffset {
		m.viewport.SetYOffset(m.treeIndex)
	}

	if m.treeIndex >= m.viewport.YOffset+visibleHeight {
		m.viewport.SetYOffset(m.treeIndex - visibleHeight + 1)
	}
}

// updateContent regenerates the viewport content
func (m *TreePane) updateContent() {
	if m.treeVM == nil {
		m.viewport.SetContent("No tree data")
		return
	}

	content := m.renderTreeContent()
	if content == "" {
		content = "(File tree rendering in progress...)"
	}
	m.viewport.SetContent(content)
}

// renderTreeContent generates the tree content
func (m *TreePane) renderTreeContent() string {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return "No tree data"
	}

	var sb strings.Builder
	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)

	for i, vn := range visibleNodes {
		isSelected := (i == m.treeIndex)
		renderNodeWithCursor(&sb, vn.Node, vn.Depth, isSelected, m.viewport.Width)
	}

	return sb.String()
}

// GetViewport returns the underlying viewport
func (m *TreePane) GetViewport() *viewport.Model {
	return &m.viewport
}
