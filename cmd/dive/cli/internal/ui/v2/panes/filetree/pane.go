package filetree

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
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
	viewport  viewport.Model
	treeIndex int
}

// New creates a new tree pane
func New(treeVM *viewmodel.FileTreeViewModel) Pane {
	vp := viewport.New(80, 20)
	p := Pane{
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
func (m *Pane) SetSize(width, height int) {
	m.width = width
	m.height = height

	viewportWidth := width - 2
	viewportHeight := height - layout.BoxContentPadding
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Width = viewportWidth
	m.viewport.Height = viewportHeight

	// CRITICAL: Regenerate content with new width to prevent soft wrap
	// Without this, long paths will wrap when window is resized
	m.updateContent()
}

// SetTreeVM updates the tree viewmodel
func (m *Pane) SetTreeVM(treeVM *viewmodel.FileTreeViewModel) {
	m.treeVM = treeVM
	m.treeIndex = 0
	m.viewport.GotoTop()
	m.updateContent()
}

// SetTreeIndex sets the current tree index
func (m *Pane) SetTreeIndex(index int) {
	m.treeIndex = index
	m.syncScroll()
}

// GetTreeIndex returns the current tree index
func (m *Pane) GetTreeIndex() int {
	return m.treeIndex
}

// Focus sets the pane as active
func (m *Pane) Focus() {
	m.focused = true
}

// Blur sets the pane as inactive
func (m *Pane) Blur() {
	m.focused = false
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
func (m Pane) View() string {
	content := m.viewport.View()
	return styles.RenderBox("Current Layer Contents", m.width, m.height, content, m.focused)
}

// moveUp moves selection up
func (m *Pane) moveUp() tea.Cmd {
	if m.treeIndex > 0 {
		m.treeIndex--
		m.syncScroll()
	}
	return nil
}

// moveDown moves selection down
func (m *Pane) moveDown() tea.Cmd {
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
func (m *Pane) toggleCollapse() tea.Cmd {
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
func (m *Pane) handleClick(x, y int) tea.Cmd {
	if x < 0 || x >= m.width || y < 0 {
		return nil
	}

	relativeY := y - layout.ContentVisualOffset
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
func (m *Pane) syncScroll() {
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
func (m *Pane) updateContent() {
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
func (m *Pane) renderTreeContent() string {
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
func (m *Pane) GetViewport() *viewport.Model {
	return &m.viewport
}
