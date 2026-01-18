package filetree

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/keys"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/dive/filetree"
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

// UpdateViewModelMsg is sent by parent when the tree viewmodel has been updated
type UpdateViewModelMsg struct {
	TreeVM *viewmodel.FileTreeViewModel
}

// SetFlatModeMsg is sent to toggle flat mode on/off
type SetFlatModeMsg struct {
	Flat bool
}

// SetCursorMsg is sent to set the cursor to a specific position
type SetCursorMsg struct {
	Index int
}

// SetFilterRegexMsg is sent to update the filter regex for search/hide logic
type SetFilterRegexMsg struct {
	Regex *regexp.Regexp
}

// Pane manages the file tree using viewport for smooth scrolling
type Pane struct {
	focused bool
	width   int
	height  int
	treeVM  *viewmodel.FileTreeViewModel

	// Viewport for state (YOffset) only
	viewport viewport.Model

	// Data
	nodes       []VisibleNode
	cursor      int            // Current selected index in nodes
	scrollOff   int            // Number of lines to keep visible above/below cursor (scrolloff)
	flatMode    bool           // true = Flat View, false = Tree View
	filterRegex *regexp.Regexp // Active filter regex for highlighting and clean search

	// Diff type filter states (default: show all)
	showAdded      bool
	showRemoved    bool
	showModified   bool
	showUnmodified bool
}

// New creates a new tree pane with viewport for smooth scrolling
func New(treeVM *viewmodel.FileTreeViewModel) Pane {
	v := viewport.New(80, 20)

	p := Pane{
		treeVM:         treeVM,
		focused:        false,
		width:          80,
		height:         20,
		viewport:       v,
		nodes:          []VisibleNode{},
		cursor:         0,
		scrollOff:      3, // Keep 3 lines visible above/below cursor (like vim scrolloff)
		showAdded:      true,
		showRemoved:    true,
		showModified:   true,
		showUnmodified: true,
	}

	// Build initial list items
	p.rebuildNodes()
	return p
}

// Resize updates the pane dimensions
func (p *Pane) Resize(width, height int) {
	p.width = width
	p.height = height

	// Calculate available height for the viewport content
	// Layout Padding: 2 (Top Border) + 2 (Bottom Border/Title gap) = 4
	// Header visual height: 1
	// Total: 4 (BoxContentPadding) + 1 (visualHeaderHeight) = 5
	const visualHeaderHeight = 1
	availableHeight := height - layout.BoxContentPadding - visualHeaderHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	// Update viewport size (v1 uses direct field access)
	p.viewport.Width = width - 2
	p.viewport.Height = availableHeight

	// Re-set dummy content to ensure viewport logic works with new height
	p.updateViewportHeight()
}

// SetTreeVM updates the tree viewmodel
func (p *Pane) SetTreeVM(treeVM *viewmodel.FileTreeViewModel) {
	p.treeVM = treeVM
	p.rebuildNodes()
	p.cursor = 0
	p.ensureCursorVisible()
}

// SetTreeIndex sets the current tree index
func (p *Pane) SetTreeIndex(index int) {
	if index >= 0 && index < len(p.nodes) {
		p.cursor = index
		p.ensureCursorVisible()
	}
}

// GetTreeIndex returns the current tree index
func (p *Pane) GetTreeIndex() int {
	return p.cursor
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
	switch msg := msg.(type) {
	case common.LayoutMsg:
		p.Resize(msg.RightWidth, msg.TreeHeight)
		return p, nil

	case FocusStateMsg:
		p.SetFocused(msg.Focused)
		return p, nil

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}

		// Handle keyboard navigation
		switch msg.String() {
		case "up", "k":
			p.moveCursor(-1)
		case "down", "j":
			p.moveCursor(1)
		case "home", "g":
			p.cursor = 0
			p.ensureCursorVisible()
		case "end", "G":
			if len(p.nodes) > 0 {
				p.cursor = len(p.nodes) - 1
			}
			p.ensureCursorVisible()
		case "enter", "space", "right", "l":
			return p, p.toggleCollapse()
		case "left", "h":
			return p, p.handleLeftKey()
		case "f":
			p.flatMode = !p.flatMode
			p.rebuildNodes()
			return p, nil
		// Tree folding controls
		case "C":
			p.setAllCollapsed(true)
			p.rebuildNodes()
			return p, nil
		case "O":
			p.setAllCollapsed(false)
			p.rebuildNodes()
			return p, nil
		// Diff type filter toggles
		case "a":
			p.showAdded = !p.showAdded
			p.rebuildNodes()
			return p, nil
		case "r":
			p.showRemoved = !p.showRemoved
			p.rebuildNodes()
			return p, nil
		case "m":
			p.showModified = !p.showModified
			p.rebuildNodes()
			return p, nil
		case "u":
			p.showUnmodified = !p.showUnmodified
			p.rebuildNodes()
			return p, nil
		}

	// --- MOUSE HANDLING ---
	case common.LocalMouseMsg:
		if !p.focused {
			return p, nil
		}

		mouseMsg := msg.MouseMsg
		if mouseMsg.Action == tea.MouseActionPress {
			// Handle Scrolling
			if mouseMsg.Button == tea.MouseButtonWheelUp {
				p.viewport.ScrollUp(1)
				return p, nil
			}
			if mouseMsg.Button == tea.MouseButtonWheelDown {
				p.viewport.ScrollDown(1)
				return p, nil
			}

			// Handle Left Click (Selection)
			if mouseMsg.Button == tea.MouseButtonLeft {
				// Calculate Y offset for the content
				// Y=0: Border Top
				// Y=1: Title
				// Y=2: Padding (Space)
				// Y=3: Table Header (RenderHeader)
				// Y=4: Content Start
				const contentOffsetY = 4

				// Calculate which row was clicked relative to viewport top
				clickY := msg.LocalY - contentOffsetY

				if clickY >= 0 {
					// Add viewport scroll offset to get absolute index
					targetIndex := clickY + p.viewport.YOffset

					// Validate index
					if targetIndex >= 0 && targetIndex < len(p.nodes) {
						p.cursor = targetIndex
						p.ensureCursorVisible()

						// Check if clicked node is a directory
						node := p.nodes[p.cursor].Node
						if node.Data.FileInfo.IsDir() {
							// Toggle collapse/expand on directory click
							return p, p.toggleCollapse()
						}

						// For files, just notify of selection change
						return p, func() tea.Msg { return TreeSelectionChangedMsg{NodeIndex: p.cursor} }
					}
				}
			}
		}

	case NodeToggledMsg, RefreshTreeContentMsg:
		p.rebuildNodes()

	case UpdateViewModelMsg:
		p.SetTreeVM(msg.TreeVM)
		return p, nil

	case SetFlatModeMsg:
		// Toggle flat mode based on message
		if p.flatMode != msg.Flat {
			p.flatMode = msg.Flat
			p.rebuildNodes()
		}
		return p, nil

	case SetCursorMsg:
		// Set cursor to specific position
		p.SetTreeIndex(msg.Index)
		return p, nil

	case SetFilterRegexMsg:
		// Update filter regex and rebuild nodes with clean search logic
		if p.filterRegex != msg.Regex {
			p.filterRegex = msg.Regex
			p.rebuildNodes()
		}
		return p, nil
	}

	// Update viewport
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the pane
func (p Pane) View() string {
	// OPTIMIZATION: Render ONLY visible nodes based on viewport state
	content := p.renderVisibleContent()

	// DO NOT call p.viewport.SetContent(content) here. It's too slow for large trees.
	// DO NOT call p.viewport.View() here (it would return the dummy newlines).

	// 1. Static table header
	header := RenderHeader(p.width)

	// 2. Combine header and rendered visible slice
	fullContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// 3. Build title based on current mode
	title := "Current Layer Contents"
	if p.flatMode {
		title += " (Flat)"
	}

	return styles.RenderBox(title, p.width, p.height, fullContent, p.focused)
}

// moveCursor moves the cursor by delta and ensures visibility
func (p *Pane) moveCursor(delta int) {
	if len(p.nodes) == 0 {
		return
	}

	newCursor := p.cursor + delta
	if newCursor < 0 {
		newCursor = 0
	} else if newCursor >= len(p.nodes) {
		newCursor = len(p.nodes) - 1
	}

	p.cursor = newCursor
	p.ensureCursorVisible()
}

// ensureCursorVisible ensures the cursor is visible in viewport with scrolloff
func (p *Pane) ensureCursorVisible() {
	if len(p.nodes) == 0 {
		return
	}

	// Each node is 1 line high
	cursorLine := p.cursor

	// Get current viewport bounds
	viewportHeight := p.viewport.Height
	if viewportHeight <= 0 {
		return
	}

	// Calculate desired top position with scrolloff
	desiredTop := cursorLine - p.scrollOff
	if desiredTop < 0 {
		desiredTop = 0
	}

	// Calculate desired bottom position
	desiredBottom := cursorLine + p.scrollOff

	// Get current line offset
	currentTop := p.viewport.YOffset

	// Adjust viewport YOffset manually
	// We operate on the viewport model directly to sync state
	if cursorLine < currentTop+p.scrollOff {
		p.viewport.SetYOffset(desiredTop)
	} else if cursorLine >= currentTop+viewportHeight-p.scrollOff {
		// Calculate new top to make cursor visible at bottom
		newTop := desiredBottom - viewportHeight + 1
		if newTop < 0 {
			newTop = 0
		}
		p.viewport.SetYOffset(newTop)
	}
}

// renderVisibleContent renders ONLY the nodes currently visible in the viewport
func (p Pane) renderVisibleContent() string {
	if len(p.nodes) == 0 {
		return ""
	}

	start := p.viewport.YOffset
	height := p.viewport.Height
	end := start + height

	// Bounds checks
	if start < 0 {
		start = 0
	}
	if start >= len(p.nodes) {
		return "" // Scrolled past end
	}
	if end > len(p.nodes) {
		end = len(p.nodes)
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		// Render only the visible slice
		// i is the absolute index in p.nodes
		line := p.renderNodeLine(p.nodes[i], i == p.cursor)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderNodeLine renders a single node line
func (p Pane) renderNodeLine(node VisibleNode, isSelected bool) string {
	if node.DisplayName != "" {
		return RenderNodeLineWithDisplayName(node.Node, node.Prefix, node.DisplayName, isSelected, p.width-2, p.filterRegex)
	}
	return RenderNodeLine(node.Node, node.Prefix, isSelected, p.width-2, p.filterRegex)
}

// rebuildNodes rebuilds the visible nodes list when tree structure changes
func (p *Pane) rebuildNodes() {
	if p.treeVM == nil || p.treeVM.ViewTree == nil {
		p.nodes = []VisibleNode{}
		p.updateViewportHeight()
		return
	}

	// Create filter options based on current toggle states
	opts := FilterOptions{
		ShowAdded:      p.showAdded,
		ShowRemoved:    p.showRemoved,
		ShowModified:   p.showModified,
		ShowUnmodified: p.showUnmodified,
	}

	// Flatten tree structure into visible nodes
	// Use different strategy based on current view mode and filter state
	switch {
	case p.flatMode && p.filterRegex != nil:
		// Clean search mode: Flat View + Active Filter
		// Show ONLY matching nodes, no parent directories (Google-style)
		p.nodes = CollectSearchResults(p.treeVM.ViewTree.Root, p.filterRegex)
		// Apply diff type filtering to search results
		p.nodes = FilterFlatList(p.nodes, opts)
	case p.flatMode:
		// Flat View without filter (manual toggle with 'f')
		p.nodes = CollectFlatNodes(p.treeVM.ViewTree.Root)
		// Apply diff type filtering
		p.nodes = FilterFlatList(p.nodes, opts)
	default:
		// Regular Tree View with diff type filtering
		p.nodes = CollectVisibleNodesWithFilter(p.treeVM.ViewTree.Root, opts)
	}

	// Ensure cursor is valid
	if p.cursor >= len(p.nodes) {
		p.cursor = max(0, len(p.nodes)-1)
	}

	// Update viewport dummy content so it knows how to scroll
	p.updateViewportHeight()
	p.ensureCursorVisible()
}

// updateViewportHeight sets dummy content to the viewport so it calculates YOffset correctly
func (p *Pane) updateViewportHeight() {
	if len(p.nodes) == 0 {
		p.viewport.SetContent("")
		return
	}
	// We feed the viewport a string of just newlines.
	// This is extremely cheap (fast) compared to rendering the tree.
	// The viewport uses this to calculate scroll percentage and boundaries.
	p.viewport.SetContent(strings.Repeat("\n", len(p.nodes)-1))
}

// setAllCollapsed recursively sets the collapsed state of all directories
func (p *Pane) setAllCollapsed(collapsed bool) {
	if p.treeVM == nil || p.treeVM.ViewTree == nil || p.treeVM.ViewTree.Root == nil {
		return
	}

	var traverse func(node *filetree.FileNode)
	traverse = func(node *filetree.FileNode) {
		if node.Data.FileInfo.IsDir() {
			node.Data.ViewInfo.Collapsed = collapsed
			for _, child := range node.Children {
				traverse(child)
			}
		}
	}

	// Don't collapse the root itself, usually user wants to see top level
	// But we do traverse its children
	root := p.treeVM.ViewTree.Root
	for _, child := range root.Children {
		traverse(child)
	}
}

// toggleCollapse toggles the collapsed state of the selected directory
func (p *Pane) toggleCollapse() tea.Cmd {
	if p.cursor < 0 || p.cursor >= len(p.nodes) {
		return nil
	}

	node := p.nodes[p.cursor].Node

	// Only directories can be collapsed/expanded
	if node.Data.FileInfo.IsDir() {
		node.Data.ViewInfo.Collapsed = !node.Data.ViewInfo.Collapsed
		p.rebuildNodes()

		// Try to maintain cursor position
		if p.cursor >= len(p.nodes) {
			p.cursor = max(0, len(p.nodes)-1)
		}

		p.ensureCursorVisible()

		return func() tea.Msg {
			return NodeToggledMsg{NodeIndex: p.cursor}
		}
	}

	return nil
}

// handleLeftKey handles left arrow key behavior
func (p *Pane) handleLeftKey() tea.Cmd {
	if p.cursor < 0 || p.cursor >= len(p.nodes) {
		return nil
	}

	node := p.nodes[p.cursor].Node

	// If current node is an expanded directory, collapse it
	if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed {
		return p.toggleCollapse()
	}

	// Otherwise, navigate to parent directory
	if node.Parent != nil {
		for i, n := range p.nodes {
			if n.Node == node.Parent {
				p.cursor = i
				p.ensureCursorVisible()
				return func() tea.Msg {
					return TreeSelectionChangedMsg{NodeIndex: i}
				}
			}
		}
	}

	return nil
}

// GetViewport returns the underlying viewport model
func (p *Pane) GetViewport() *viewport.Model {
	return &p.viewport
}

// GetVisibleNodeCount returns the number of visible nodes in the tree
// This is used by the search system to count matches efficiently
func (p *Pane) GetVisibleNodeCount() int {
	return len(p.nodes)
}

// GetFilterState returns the current filter state for the help bar styling
func (p *Pane) GetFilterState() (showAdded, showRemoved, showModified, showUnmodified bool) {
	return p.showAdded, p.showRemoved, p.showModified, p.showUnmodified
}

// ShortHelp returns key bindings specific to the file tree pane.
// File tree has unique navigation keys for collapsing/expanding folders.
func (p *Pane) ShortHelp() []key.Binding {
	return []key.Binding{
		keys.Keys.ToggleAdded,      // Toggle added files
		keys.Keys.ToggleRemoved,    // Toggle removed files
		keys.Keys.ToggleModified,   // Toggle modified files
		keys.Keys.ToggleUnmodified, // Toggle unmodified files
		keys.Keys.ToggleView,       // Toggle flat/tree view
	}
}
