package filetree

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

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

// SetLayerInfoMsg is sent to update layer information for title display
type SetLayerInfoMsg struct {
	LayerIndex  int // Current layer index (0-based)
	TotalLayers int
}

// CopyFinishedMsg is sent after a copy operation completes
type CopyFinishedMsg struct {
	Path    string
	Success bool
}

// HideCopyNoticeMsg is sent to hide the copy notification after timeout
type HideCopyNoticeMsg struct{}

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

	// Layer information for title display
	currentLayerIndex int // Current layer index (0-based)
	totalLayers       int // Total number of layers

	// Diff type filter states (default: show all)
	showAdded      bool
	showRemoved    bool
	showModified   bool
	showUnmodified bool

	// Copy notification state
	copyNoticePath  string // Path that was copied (for display in title)
	copiedNodeIndex int    // Index of the node that was copied (for icon change)

	// Last collapse state for toggle functionality
	lastCollapseState bool // true = collapsed, false = expanded
}

// New creates a new tree pane with viewport for smooth scrolling
func New(treeVM *viewmodel.FileTreeViewModel) Pane {
	v := viewport.New(80, 20)

	p := Pane{
		treeVM:            treeVM,
		focused:           false,
		width:             80,
		height:            20,
		viewport:          v,
		nodes:             []VisibleNode{},
		cursor:            0,
		scrollOff:         3, // Keep 3 lines visible above/below cursor (like vim scrolloff)
		showAdded:         true,
		showRemoved:       true,
		showModified:      true,
		showUnmodified:    true,
		copyNoticePath:    "",   // Initialize empty
		copiedNodeIndex:   -1,   // -1 means no node copied
		lastCollapseState: true, // Start with collapsed state (default is collapsed)
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

	// Set all directories as collapsed by default
	p.setAllCollapsed(true)

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
//
//nolint:funlen,gocyclo,gocognit
func (p *Pane) Update(msg tea.Msg) (common.Pane, tea.Cmd) {
	switch msg := msg.(type) {
	case common.LayoutMsg:
		p.Resize(msg.RightWidth, msg.TreeHeight)
		return p, nil

	case FocusStateMsg:
		p.SetFocused(msg.Focused)
		return p, nil

	case tea.KeyMsg:
		// 1. First handle GLOBAL keys (work without focus)
		// These keys can be forwarded from app/model.go, even when pane is not active
		switch msg.String() {
		// Diff type filter toggles (exclusive/radio button behavior)
		case "a":
			p.setExclusiveFilter("added")
			return p, nil
		case "r":
			p.setExclusiveFilter("removed")
			return p, nil
		case "m":
			p.setExclusiveFilter("modified")
			return p, nil
		case "u":
			p.setExclusiveFilter("unmodified")
			return p, nil
		// Tree folding controls
		case "C":
			p.setAllCollapsed(true)
			p.lastCollapseState = true // Remember state
			p.rebuildNodes()
			return p, nil
		case "O":
			p.setAllCollapsed(false)
			p.lastCollapseState = false // Remember state
			p.rebuildNodes()
			return p, nil
		case "c":
			// Toggle: collapse all if expanded, expand all if collapsed
			p.toggleAllCollapse()
			return p, nil
		}

		// 2. If pane is NOT focused, ignore the rest of keys (navigation)
		if !p.focused {
			return p, nil
		}

		// 3. Handle navigation (only when focused)
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
		}

	// --- MOUSE HANDLING ---
	case common.LocalMouseMsg:
		if !p.focused {
			return p, nil
		}

		mouseMsg := msg.MouseMsg
		if mouseMsg.Action == tea.MouseActionPress { //nolint:nestif
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
				// Y=0: Border Top (title is embedded in border)
				// Y=1: Table Header (RenderHeader)
				// Y=2: Content Start
				const contentOffsetY = 2

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

			// Handle Right Click (Copy file path)
			if mouseMsg.Button == tea.MouseButtonRight {
				// Calculate Y offset for the content
				const contentOffsetY = 2

				// Calculate which row was clicked relative to viewport top
				clickY := msg.LocalY - contentOffsetY

				if clickY >= 0 {
					// Add viewport scroll offset to get absolute index
					targetIndex := clickY + p.viewport.YOffset

					// Validate index
					if targetIndex >= 0 && targetIndex < len(p.nodes) {
						// Get the full path of the clicked node
						node := p.nodes[targetIndex].Node
						filePath := node.Path()

						// Store the index to show copy icon
						p.copiedNodeIndex = targetIndex

						// Copy to clipboard and trigger UI update
						return p, copyPathToClipboard(filePath)
					}
				}
			}
		}

	case CopyFinishedMsg:
		// Handle copy completion - show notification
		if msg.Success {
			p.copyNoticePath = msg.Path
			// copiedNodeIndex is already set by the right-click handler
			// Hide notification after 2 seconds
			return p, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
				return HideCopyNoticeMsg{}
			})
		}

	case HideCopyNoticeMsg:
		// Hide copy notification and reset icon
		p.copyNoticePath = ""
		p.copiedNodeIndex = -1
		return p, nil

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

	case SetLayerInfoMsg:
		// Update layer information for title display
		p.currentLayerIndex = msg.LayerIndex
		p.totalLayers = msg.TotalLayers
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
	baseTitle := "Current Layer Contents"

	// Add layer number if we have layer info
	if p.totalLayers > 0 {
		layerNum := p.currentLayerIndex + 1 // Convert to 1-based
		baseTitle = fmt.Sprintf("[%d/%d] %s", layerNum, p.totalLayers, baseTitle)
	}

	// Visualize filters in title: [A---] for only Added, [--M-] for only Modified, etc.
	filters := ""
	if p.showAdded {
		filters += "A"
	} else {
		filters += "-"
	}
	if p.showRemoved {
		filters += "R"
	} else {
		filters += "-"
	}
	if p.showModified {
		filters += "M"
	} else {
		filters += "-"
	}
	if p.showUnmodified {
		filters += "U"
	} else {
		filters += "-"
	}

	// Build final title
	var title string

	// If there's an active copy notification, show it
	if p.copyNoticePath != "" {
		// Truncate path if too long
		displayPath := p.copyNoticePath
		if len(displayPath) > 40 {
			displayPath = "..." + displayPath[len(displayPath)-40:]
		}
		title = fmt.Sprintf("COPIED: %s", displayPath)
	} else {
		// Standard title
		title = baseTitle
		// If filters are not all enabled, show filter indicator in title
		if filters != "ARMU" {
			title = fmt.Sprintf("[%d/%d] Layer Contents [%s]", p.currentLayerIndex+1, p.totalLayers, filters)
		}

		if p.flatMode {
			title += " (Flat)"
		}
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
	// Find current node index in the nodes array
	currentIndex := -1
	for i, n := range p.nodes {
		if n.Node == node.Node {
			currentIndex = i
			break
		}
	}

	// Determine if this node was just copied
	isCopied := currentIndex >= 0 && currentIndex == p.copiedNodeIndex

	if node.DisplayName != "" {
		return RenderNodeLineWithDisplayName(node.Node, node.Prefix, node.DisplayName, isSelected, p.width-2, p.filterRegex, isCopied)
	}
	return RenderNodeLine(node.Node, node.Prefix, isSelected, p.width-2, p.filterRegex, isCopied)
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

// toggleAllCollapse toggles all directories between collapsed and expanded
// Uses lastCollapseState to determine next action (toggle behavior)
func (p *Pane) toggleAllCollapse() {
	if p.treeVM == nil || p.treeVM.ViewTree == nil || p.treeVM.ViewTree.Root == nil {
		return
	}

	// Toggle based on last state
	// If last was collapsed → expand all
	// If last was expanded → collapse all
	newState := !p.lastCollapseState
	p.setAllCollapsed(newState)

	// Update last state
	p.lastCollapseState = newState

	// Rebuild nodes to reflect the new state
	p.rebuildNodes()
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

// setExclusiveFilter enables "solo" mode for the selected filter type.
// If this type is already in solo mode, resets to show all files.
// This provides radio-button-like behavior instead of toggle/checkbox.
func (p *Pane) setExclusiveFilter(filterType string) {
	// Check if currently ONLY this filter is enabled
	isOnlyAdded := p.showAdded && !p.showRemoved && !p.showModified && !p.showUnmodified
	isOnlyRemoved := !p.showAdded && p.showRemoved && !p.showModified && !p.showUnmodified
	isOnlyModified := !p.showAdded && !p.showRemoved && p.showModified && !p.showUnmodified
	isOnlyUnmodified := !p.showAdded && !p.showRemoved && !p.showModified && p.showUnmodified

	var currentlyExclusive bool
	switch filterType {
	case "added":
		currentlyExclusive = isOnlyAdded
	case "removed":
		currentlyExclusive = isOnlyRemoved
	case "modified":
		currentlyExclusive = isOnlyModified
	case "unmodified":
		currentlyExclusive = isOnlyUnmodified
	}

	if currentlyExclusive {
		// If already in "only this" mode, press again -> RESET (show all)
		p.showAdded = true
		p.showRemoved = true
		p.showModified = true
		p.showUnmodified = true
	} else {
		// Otherwise -> ENABLE ONLY THIS
		p.showAdded = (filterType == "added")
		p.showRemoved = (filterType == "removed")
		p.showModified = (filterType == "modified")
		p.showUnmodified = (filterType == "unmodified")
	}

	// Rebuild the tree
	p.rebuildNodes()
}

// copyPathToClipboard copies a file path to the system clipboard
// Returns CopyFinishedMsg to trigger UI notification
func copyPathToClipboard(path string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		ctx := context.Background()

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.CommandContext(ctx, "pbcopy")
		case "linux":
			// Try xclip first, then wl-copy, then xsel
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.CommandContext(ctx, "wl-copy")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.CommandContext(ctx, "xsel", "--clipboard", "--input")
			}
		case "windows":
			cmd = exec.CommandContext(ctx, "clip")
		}

		if cmd != nil {
			cmd.Stdin = bytes.NewBufferString(path)
			err := cmd.Run()
			// Return result - success if no error
			return CopyFinishedMsg{Path: path, Success: err == nil}
		}

		// No clipboard tool found
		return CopyFinishedMsg{Path: "Clipboard tool not found (install xclip/xsel)", Success: false}
	}
}

// ShortHelp returns key bindings specific to the file tree pane.
// File tree has unique navigation keys for collapsing/expanding folders.
// Note: ToggleView is NOT included here since it's already in global keys (keys.ShortHelp())
func (p *Pane) ShortHelp() []key.Binding {
	// Create a dynamic toggle key binding based on current state
	toggleKey := keys.Keys.ToggleCollapse

	// Dynamic hint: show what the button will do when pressed
	if p.lastCollapseState {
		// If currently collapsed, button will "expand all"
		toggleKey.SetHelp("c", "expand all")
	} else {
		// If currently expanded, button will "collapse all"
		toggleKey.SetHelp("c", "collapse all")
	}

	return []key.Binding{
		toggleKey,                  // Dynamic toggle collapse/expand
		keys.Keys.ToggleAdded,      // Toggle added files
		keys.Keys.ToggleRemoved,    // Toggle removed files
		keys.Keys.ToggleModified,   // Toggle modified files
		keys.Keys.ToggleUnmodified, // Toggle unmodified files
	}
}
