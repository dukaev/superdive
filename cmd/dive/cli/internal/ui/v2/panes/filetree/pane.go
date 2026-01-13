package filetree

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	focused bool
	width   int
	height  int
	treeVM  *viewmodel.FileTreeViewModel

	// Cache of currently visible nodes to avoid re-traversal every frame
	visibleNodes []VisibleNode

	// Components
	selection    *Selection
	viewportMgr  *ViewportManager
	navigation   *Navigation
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
		visibleNodes: []VisibleNode{},
	}

	// Set up callbacks
	p.navigation.SetVisibleNodesFunc(func() []VisibleNode {
		// Return the cached nodes if available to speed up navigation checks
		if p.visibleNodes != nil {
			return p.visibleNodes
		}
		if p.treeVM == nil || p.treeVM.ViewTree == nil {
			return nil
		}
		return CollectVisibleNodes(p.treeVM.ViewTree.Root)
	})

	p.navigation.SetRefreshFunc(p.updateContent)
	p.navigation.SetToggleCollapseFunc(p.toggleCollapse)

	// Set up callback for InputHandler to use Pane's toggleCollapse implementation
	// This ensures it uses the cached visibleNodes instead of re-traversing the tree
	p.inputHandler.SetToggleCollapseFunc(p.toggleCollapse)

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

	// Calculate viewport height accounting for:
	// - BoxContentPadding: borders (2) + box header (2) = 4
	// - TreeTableHeaderHeight: "Name   Size   Permissions" header (1)
	viewportHeight := height - layout.BoxContentPadding - layout.TreeTableHeaderHeight
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

	// CRITICAL: Also update the reference in InputHandler to prevent desync
	// Without this, InputHandler would continue operating on the old tree reference
	m.inputHandler.SetTreeVM(treeVM)

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

	case NodeToggledMsg:
		// A folder was collapsed/expanded, need to refresh visibleNodes cache
		// CRITICAL: This fixes the copy-on-write issue where InputHandler updates
		// the old copy of the Pane. By handling this message in the active Pane's
		// Update method, we ensure the visible copy of the Pane refreshes its cache.
		m.updateContent()

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

	// 2. Render ONLY the visible rows based on viewport state (Virtualization)
	// We do NOT use m.viewportMgr.GetViewport().View() for the content because
	// we only want to render the lines that are currently on screen to avoid lag.
	content := m.renderVisibleContent()

	// 3. Combine: Header + Content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	return styles.RenderBox("Current Layer Contents", m.width, m.height, fullContent, m.focused)
}

// updateContent refreshes the cache and updates the viewport scroll bounds
func (m *Pane) updateContent() {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		m.visibleNodes = nil
		m.viewportMgr.SetContent("No tree data")
		return
	}

	// 1. Cache the visible nodes structure (fast pointer traversal)
	m.visibleNodes = CollectVisibleNodes(m.treeVM.ViewTree.Root)

	// 2. Set "dummy" content to the viewport to establish correct scrollbar math
	// We don't render the text here. We just give the viewport a string with
	// the correct number of newlines so it knows how tall the content *would* be.
	// This makes PageDown/Up and scrolling work correctly.
	count := len(m.visibleNodes)
	if count > 0 {
		dummyContent := strings.Repeat("\n", count-1)
		m.viewportMgr.SetContent(dummyContent)
	} else {
		m.viewportMgr.SetContent("")
	}
}

// renderVisibleContent generates strings only for the rows currently visible in the viewport
func (m *Pane) renderVisibleContent() string {
	if len(m.visibleNodes) == 0 {
		return "No files"
	}

	// Get current scroll window
	yOffset := m.viewportMgr.GetYOffset()
	height := m.viewportMgr.GetHeight()

	// Calculate slice bounds
	start := yOffset
	end := start + height

	// Clamp bounds
	if start < 0 {
		start = 0
	}
	if start > len(m.visibleNodes) {
		start = len(m.visibleNodes)
	}
	if end > len(m.visibleNodes) {
		end = len(m.visibleNodes)
	}

	// Render loop - only for visible items (e.g., 20 items instead of 10,000)
	var sb strings.Builder
	viewportWidth := m.viewportMgr.GetViewport().Width

	for i := start; i < end; i++ {
		vn := m.visibleNodes[i]
		isSelected := (i == m.selection.GetTreeIndex())
		RenderNodeWithCursor(&sb, vn.Node, vn.Prefix, isSelected, viewportWidth)
	}

	// If the rendered content is shorter than the viewport (e.g. end of list),
	// pad with empty lines to maintain box size
	renderedLines := end - start
	if renderedLines < height {
		// padding := height - renderedLines
		// sb.WriteString(strings.Repeat("\n", padding))
	}

	return sb.String()
}

// toggleCollapse toggles the current node's collapse state
func (m *Pane) toggleCollapse() tea.Cmd {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return nil
	}

	// Use cached nodes for index lookup
	if len(m.visibleNodes) == 0 {
		return nil
	}

	treeIndex := m.selection.GetTreeIndex()

	// Bounds check
	if treeIndex >= len(m.visibleNodes) {
		m.selection.MoveToIndex(len(m.visibleNodes) - 1)
		treeIndex = m.selection.GetTreeIndex()
	}
	if treeIndex < 0 {
		m.selection.SetTreeIndex(0)
		treeIndex = m.selection.GetTreeIndex()
	}

	if treeIndex < len(m.visibleNodes) {
		selectedNode := m.visibleNodes[treeIndex].Node

		if selectedNode.Data.FileInfo.IsDir() {
			// Toggle the collapsed flag directly on the node
			selectedNode.Data.ViewInfo.Collapsed = !selectedNode.Data.ViewInfo.Collapsed

			// Refresh content (re-collect nodes and update viewport bounds)
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
