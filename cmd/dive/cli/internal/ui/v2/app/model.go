package app

import (
	"context"
	"fmt"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/lrstanley/bubblezone"
	"github.com/charmbracelet/lipgloss"
	v1 "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/keys"
	filetreepane "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/filetree"
	imagepane "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/image"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/details"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/layers"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	filetree "github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
)

// Pane represents a UI pane
type Pane int

const (
	PaneLayer Pane = iota
	PaneDetails
	PaneImage
	PaneTree
)

// Use layout package constants
const (
	BorderHeight         = layout.BorderHeight
	HeaderHeight         = layout.HeaderHeight
	BoxContentPadding    = layout.BoxContentPadding
	ContentVisualOffset  = layout.ContentVisualOffset
)

func (p Pane) String() string {
	switch p {
	case PaneLayer:
		return "Layers"
	case PaneDetails:
		return "Details"
	case PaneImage:
		return "Image"
	case PaneTree:
		return "Tree"
	}
	return "Unknown"
}

// keyMapWrapper wraps a slice of key bindings to implement help.KeyMap interface
// This allows us to dynamically combine global and pane-specific keys
type keyMapWrapper struct {
	keys []key.Binding
}

func (k keyMapWrapper) ShortHelp() []key.Binding {
	return k.keys
}

func (k keyMapWrapper) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.keys}
}

// Model is the bubbletea Model for V2UI
type Model struct {
	// Configuration
	analysis image.Analysis
	content  image.ContentReader
	prefs    v1.Preferences
	ctx      context.Context

	// Viewmodels (SHARED with V1UI!)
	layerVM *viewmodel.LayerSetState
	treeVM  *viewmodel.FileTreeViewModel

	// UI state
	width    int
	height   int
	quitting bool
	layout   layout.Result // Stores calculated pane dimensions from layout engine

	// Panes stored by interface (polymorphic access)
	// No need for concrete types - interface handles everything
	panes map[Pane]common.Pane

	// Active pane state
	activePane Pane

	// Search state
	searching      bool            // Whether search mode is active
	previousPane   Pane            // Pane that was active before search (for Esc to restore focus)
	searchInput   textinput.Model // Search input field
	filterRegex   *regexp.Regexp  // Compiled regex for tree filtering
	currentMatch  int             // Index of currently selected match (-1 if no match)
	totalMatches  int             // Total number of matches

	// Help and key bindings
	keys keys.KeyMap
	help help.Model
}

// NewModel creates a new bubbletea model with configuration
func NewModel(analysis image.Analysis, content image.ContentReader, prefs v1.Preferences, ctx context.Context) Model {
	// Initialize layer viewmodel
	var layerVM *viewmodel.LayerSetState
	if analysis.Layers != nil && len(analysis.Layers) > 0 {
		layerVM = viewmodel.NewLayerSetState(
			analysis.Layers,
			viewmodel.CompareSingleLayer,
		)
	}

	// Initialize filetree viewmodel
	var treeVM *viewmodel.FileTreeViewModel
	var comparer filetree.Comparer
	if len(analysis.RefTrees) > 0 {
		v1cfg := v1.Config{
			Analysis:    analysis,
			Content:     content,
			Preferences: prefs,
		}
		// Get comparer for layer tree comparison
		comparer, _ = v1cfg.TreeComparer()
		// Note: we ignore the error here since treeVM.Update() will be called in Init()
		treeVM, _ = viewmodel.NewFileTreeViewModel(v1cfg, 0)
	}

	h := help.New()
	h.Width = 80
	h.Styles.ShortKey = styles.HelpStyle
	h.Styles.ShortDesc = styles.HelpStyle
	h.Styles.Ellipsis = styles.HelpStyle

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 156
	ti.Prompt = "Filter: "
	ti.PromptStyle = styles.SearchPrefixStyle
	ti.SetValue("")

	// Create pane components
	layersPane := layers.New(layerVM, comparer)
	detailsPane := details.New()
	imagePane := imagepane.New(&analysis)
	treePane := filetreepane.New(treeVM)

	// Create model with initial dimensions
	// POLYMORPHISM: Store all panes as common.Pane interface
	model := Model{
		analysis:         analysis,
		content:          content,
		prefs:            prefs,
		ctx:              ctx,
		layerVM:          layerVM,
		treeVM:           treeVM,
		panes: map[Pane]common.Pane{
			PaneLayer:   &layersPane,
			PaneDetails: &detailsPane,
			PaneImage:   &imagePane,
			PaneTree:    &treePane,
		},
		width:            80,
		height:           24,
		quitting:         false,
		activePane:       PaneLayer,
		searching:        false,
		searchInput:      ti,
		currentMatch:     -1,
		totalMatches:     0,
		keys:             keys.Keys,
		help:             h,
	}

	// CRITICAL: Calculate initial layout immediately
	// This ensures panes have correct dimensions before first render
	model.recalculateLayout()

	// Send LayoutMsg to set initial pane sizes via message passing
	layoutMsg := common.LayoutMsg{
		LeftWidth:     model.layout.LeftWidth,
		LayersHeight:  model.layout.LayersHeight,
		DetailsHeight: model.layout.DetailsHeight,
		ImageHeight:   model.layout.ImageHeight,
		RightWidth:    model.layout.RightWidth,
		TreeHeight:    model.layout.TreeHeight,
	}

	// Update all panes with initial layout
	// POLYMORPHISM: Same code for all panes, no type assertions needed!
	for paneType, pane := range model.panes {
		updatedPane, _ := pane.Update(layoutMsg)
		model.panes[paneType] = updatedPane
	}

	return model
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// IMPORTANT: Initialize tree on startup so it's not empty!
	if m.layerVM != nil && m.treeVM != nil {
		m.updateTreeForCurrentLayer()
	}

	// Initialize details pane with current layer via message
	if m.layerVM != nil && len(m.layerVM.Layers) > 0 {
		layerIndex := m.layerVM.LayerIndex
		if layerIndex >= 0 && layerIndex < len(m.layerVM.Layers) {
			layerMsg := common.LayerSelectedMsg{
				Layer:      m.layerVM.Layers[layerIndex],
				LayerIndex: layerIndex,
			}
			// POLYMORPHISM: Update pane through interface, no type assertion
			newDetails, _ := m.panes[PaneDetails].Update(layerMsg)
			m.panes[PaneDetails] = newDetails
		}
	}

	// CRITICAL: Set initial focus state
	// Parent tells children which pane is focused via FocusStateMsg
	m.sendFocusStates()

	return tea.Batch(cmds...)
}

// recalculateLayout uses the layout engine to calculate pane dimensions
func (m *Model) recalculateLayout() {
	engine := layout.NewEngine()
	result := engine.Calculate(m.width, m.height)

	m.layout.ContentStartY = result.ContentStartY
	m.layout.LeftWidth = result.LeftWidth
	m.layout.RightWidth = result.RightWidth
	m.layout.LayersHeight = result.LayersHeight
	m.layout.DetailsHeight = result.DetailsHeight
	m.layout.ImageHeight = result.ImageHeight
	m.layout.TreeHeight = result.TreeHeight
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If searching is active, handle search mode
		if m.searching {
			return m.updateSearch(msg)
		}

		// Route keys to focused pane
		// POLYMORPHISM: No switch-case needed - just get the active pane from the map!
		if activePane, ok := m.panes[m.activePane]; ok {
			// Details pane is read-only, skip it
			if m.activePane != PaneDetails {
				updatedPane, cmd := activePane.Update(msg)
				m.panes[m.activePane] = updatedPane
				cmds = append(cmds, cmd)
			}
		}

		// Global key bindings
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			// ESC only quits if not searching (searching handles ESC separately)
			m.quitting = true
			return m, tea.Quit

		case "tab", "shift+tab":
			m.togglePane()

		case "ctrl+f", "/":
			// Enter search mode
			// Save current pane so Esc can restore focus later
			m.previousPane = m.activePane
			m.searching = true
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			m.currentMatch = -1
			m.totalMatches = 0
			return m, nil
		}

	case layers.LayerChangedMsg:
		// Layer changed - update details pane and tree via messages
		if m.layerVM != nil && msg.LayerIndex >= 0 && msg.LayerIndex < len(m.layerVM.Layers) {
			layerMsg := common.LayerSelectedMsg{
				Layer:      m.layerVM.Layers[msg.LayerIndex],
				LayerIndex: msg.LayerIndex,
			}
			// POLYMORPHISM: Update through interface
			newDetails, _ := m.panes[PaneDetails].Update(layerMsg)
			m.panes[PaneDetails] = newDetails
		}
		m.updateTreeForCurrentLayer()

	case filetreepane.NodeToggledMsg:
		// Forward message to tree pane to refresh its visibleNodes cache
		// CRITICAL: This fixes the copy-on-write issue. The InputHandler's callback
		// modified the collapsed flag in the tree data, but the visible copy of
		// treePane (stored in this Model) needs to refresh its cache to show changes.
		// POLYMORPHISM: Update through interface
		newPane, cmd := m.panes[PaneTree].Update(msg)
		m.panes[PaneTree] = newPane
		cmds = append(cmds, cmd)

	case filetreepane.RefreshTreeContentMsg:
		// Request to refresh tree content
		// POLYMORPHISM: Send message through interface, no type assertion
		newPane, _ := m.panes[PaneTree].Update(filetreepane.UpdateViewModelMsg{TreeVM: m.treeVM})
		m.panes[PaneTree] = newPane

	case tea.MouseMsg:
		// BUBBLEZONE: Check which pane was clicked using zone hit testing
		// This is more robust than manual coordinate calculations
		for paneID, pane := range m.panes {
			id := paneID.String() // "Layers", "Details", "Image", "Tree"

			if zone.Get(id).InBounds(msg) {
				// Change focus if needed
				if m.activePane != paneID {
					m.activePane = paneID
					m.sendFocusStates()
				}

				// Skip Details pane (read-only, no mouse handling)
				if paneID != PaneDetails {
					// Get the zone to calculate local coordinates
					z := zone.Get(id)

					// Calculate local coordinates relative to the pane
					// Pos() returns the x, y coordinates relative to the zone's origin
					localX, localY := z.Pos(msg)

					// Create local mouse message with transformed coordinates
					localMsg := common.LocalMouseMsg{
						MouseMsg: msg,
						LocalX:   localX,
						LocalY:   localY,
					}

					// POLYMORPHISM: Update through interface
					newPane, cmd := pane.Update(localMsg)
					m.panes[paneID] = newPane
					cmds = append(cmds, cmd)
				}

				// Found the clicked pane, stop checking
				break
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.recalculateLayout()

		// Send LayoutMsg to all panes with their new dimensions
		// This replaces direct SetSize() calls with message passing
		layoutMsg := common.LayoutMsg{
			LeftWidth:     m.layout.LeftWidth,
			LayersHeight:  m.layout.LayersHeight,
			DetailsHeight: m.layout.DetailsHeight,
			ImageHeight:   m.layout.ImageHeight,
			RightWidth:    m.layout.RightWidth,
			TreeHeight:    m.layout.TreeHeight,
		}

		// Broadcast to all panes - they will extract what they need
		// POLYMORPHISM: Same code for all panes, no type assertions needed!
		var layoutCmds []tea.Cmd
		for paneType, pane := range m.panes {
			updatedPane, cmd := pane.Update(layoutMsg)
			m.panes[paneType] = updatedPane
			layoutCmds = append(layoutCmds, cmd)
		}
		cmds = append(cmds, layoutCmds...)
	}

	// Update help
	var cmd tea.Cmd
	m.help, cmd = m.help.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) togglePane() {
	// Cycle through panes: Layer -> Details -> Image -> Tree -> Layer
	m.activePane++
	if m.activePane > PaneTree {
		m.activePane = PaneLayer
	}

	m.sendFocusStates()
}

func (m *Model) sendFocusStates() {
	// Send FocusStateMsg to all panes based on current active pane
	// Parent is the Single Source of Truth - children receive focus state via messages

	// POLYMORPHISM: Iterate over all panes and update their focus state
	for paneType, pane := range m.panes {
		focused := (paneType == m.activePane)

		// Create the appropriate FocusStateMsg for this pane type
		var focusMsg tea.Msg
		switch paneType {
		case PaneLayer:
			focusMsg = layers.FocusStateMsg{Focused: focused}
		case PaneDetails:
			focusMsg = details.FocusStateMsg{Focused: focused}
		case PaneImage:
			focusMsg = imagepane.FocusStateMsg{Focused: focused}
		case PaneTree:
			focusMsg = filetreepane.FocusStateMsg{Focused: focused}
		}

		// POLYMORPHISM: Update through interface, no type assertion
		updatedPane, _ := pane.Update(focusMsg)
		m.panes[paneType] = updatedPane
	}
}

func (m *Model) updateTreeForCurrentLayer() {
	if m.treeVM == nil || m.layerVM == nil {
		return
	}

	bottomTreeStart, _, topTreeStart, topTreeStop := m.layerVM.GetCompareIndexes()
	_ = m.treeVM.SetTreeByLayer(bottomTreeStart, bottomTreeStart, topTreeStart, topTreeStop)

	// IMPORTANT: Call Update() to create ViewTree from ModelTree!
	// This applies filters and creates the visible tree
	_ = m.treeVM.Update(nil, m.layout.RightWidth, m.layout.TreeHeight)

	// Update tree pane with new tree data
	// POLYMORPHISM: Send message through interface, no type assertion
	newPane, _ := m.panes[PaneTree].Update(filetreepane.UpdateViewModelMsg{TreeVM: m.treeVM})
	m.panes[PaneTree] = newPane
}

// View implements tea.Model (PURE FUNCTION - no side effects!)
func (m Model) View() string {
	if m.quitting {
		return styles.TitleStyle.Foreground(styles.SuccessColor).Render("Thanks for using Dive V2UI!")
	}

	// Calculate layout if not yet calculated (first run)
	if m.layout.LayersHeight == 0 {
		m.recalculateLayout()
	}

	// Render UI components
	// DYNAMIC HELP: Combine global keys with active pane's specific keys
	// This shows context-relevant help based on which pane is focused
	globalKeys := m.keys.ShortHelp()

	var activePaneKeys []key.Binding
	if activePane, ok := m.panes[m.activePane]; ok {
		activePaneKeys = activePane.ShortHelp()
	}

	// Combine global and pane-specific keys
	// IMPORTANT: Create new slice to avoid mutating global keys
	allKeys := append(globalKeys, activePaneKeys...)

	// Render status bar: search input or help
	var statusBar string
	if m.searching {
		// Render search input in status bar
		// Check if regex is valid to determine styling
		patternText := m.searchInput.Value()
		var inputStyle lipgloss.Style

		if patternText == "" {
			inputStyle = styles.SearchInputStyle
		} else {
			// Try to compile the regex to check validity
			if _, err := regexp.Compile(patternText); err != nil {
				inputStyle = styles.SearchErrorStyle
			} else {
				inputStyle = styles.SearchInputStyle
			}
		}

		// Re-render input with proper style (override textinput's default styling)
		styledInput := inputStyle.Render(patternText)

		// Add match counter if we have matches
		var matchCounter string
		if m.totalMatches > 0 {
			if m.currentMatch >= 0 && m.currentMatch < m.totalMatches {
				matchCounter = lipgloss.NewStyle().
					Foreground(styles.MutedTextColor).
					Render(fmt.Sprintf(" [%d/%d]", m.currentMatch+1, m.totalMatches))
			}
		}

		// Join: styled prompt + styled input + counter
		statusBar = lipgloss.JoinHorizontal(lipgloss.Left,
			styles.SearchPrefixStyle.Render("Filter: "),
			styledInput,
			matchCounter,
		)
		// Fill the rest of the line
		statusBar = lipgloss.NewStyle().Width(m.width).MaxWidth(m.width).Render(statusBar)
	} else {
		statusBar = m.help.View(keyMapWrapper{keys: allKeys})
	}

	// Render panes directly using their View() methods
	// POLYMORPHISM: Access panes through interface from map
	// BUBBLEZONE: Mark each pane with a unique ID for mouse hit testing
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		zone.Mark(PaneLayer.String(), m.panes[PaneLayer].View()),
		zone.Mark(PaneDetails.String(), m.panes[PaneDetails].View()),
		zone.Mark(PaneImage.String(), m.panes[PaneImage].View()),
	)
	treePane := zone.Mark(PaneTree.String(), m.panes[PaneTree].View())

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, treePane)

	base := lipgloss.JoinVertical(lipgloss.Left,
		mainContent,
		statusBar,
	)

	// BUGFIX: Always call zone.Scan(), even when showing modals
	// Early returns before zone.Scan() caused click zones to break
	// after closing modals (zone map wasn't being updated).
	//
	// Solution: Build the final view in a variable, then always scan it.
	finalView := base

	// BUBBLEZONE: Scan the entire output to register zones for hit testing
	// CRITICAL: This must be called ONCE on the FINAL rendered string, regardless of modals
	return zone.Scan(finalView)
}

// applyFilter applies a filter pattern to the file tree
func (m *Model) applyFilter(pattern string) {
	if m.treeVM == nil {
		return
	}

	// Compile the regex pattern (safely - don't use MustCompile)
	var filterRegex *regexp.Regexp
	if pattern != "" {
		var err error
		filterRegex, err = regexp.Compile(pattern)
		if err != nil {
			// Invalid regex - don't update filter, keep previous state
			// The UI will show the error in red
			// IMPORTANT: Don't reset counters to avoid UI desync
			return
		}
	}
	m.filterRegex = filterRegex

	// Update the tree viewmodel with the filter
	// This will update ViewTree based on the filter
	_ = m.treeVM.Update(filterRegex, m.layout.RightWidth, m.layout.TreeHeight)

	// Update tree pane with filtered tree data
	// POLYMORPHISM: Send message through interface, no type assertion
	newPane, _ := m.panes[PaneTree].Update(filetreepane.UpdateViewModelMsg{TreeVM: m.treeVM})
	m.panes[PaneTree] = newPane

	// Auto-Flat Mode: switch to flat mode when filtering, back to tree when empty
	flatMode := (pattern != "")
	newPane, _ = m.panes[PaneTree].Update(filetreepane.SetFlatModeMsg{Flat: flatMode})
	m.panes[PaneTree] = newPane

	// Send filter regex to tree pane for clean search results
	newPane, _ = m.panes[PaneTree].Update(filetreepane.SetFilterRegexMsg{Regex: filterRegex})
	m.panes[PaneTree] = newPane

	// Count matches efficiently - get visible node count from pane
	// Avoid double tree traversal
	m.totalMatches = m.getVisibleNodeCount()
	m.currentMatch = -1
}

// countMatches counts the number of files matching the current filter
// DEPRECATED: Use getVisibleNodeCount instead for better performance
func (m *Model) countMatches() int {
	return m.getVisibleNodeCount()
}

// getVisibleNodeCount efficiently gets the visible node count from the tree pane
// This avoids double tree traversal
func (m *Model) getVisibleNodeCount() int {
	if treePane, ok := m.panes[PaneTree]; ok {
		// Type assertion is acceptable here for reading a property
		// This is much faster than re-traversing the tree
		if tp, ok := treePane.(*filetreepane.Pane); ok {
			return tp.GetVisibleNodeCount()
		}
	}
	return 0
}

// findMatchIndices returns indices of all matching nodes in the current filtered tree
func (m *Model) findMatchIndices() []int {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return nil
	}

	// Collect all visible nodes (these are the filtered results)
	nodes := filetreepane.CollectVisibleNodes(m.treeVM.ViewTree.Root)

	// If no filter, return all indices
	if m.filterRegex == nil {
		indices := make([]int, len(nodes))
		for i := range nodes {
			indices[i] = i
		}
		return indices
	}

	// Find nodes that match the filter
	var indices []int
	for i, node := range nodes {
		if m.filterRegex.MatchString(node.Node.Path()) {
			indices = append(indices, i)
		}
	}

	return indices
}

// updateSearch handles key presses when in search mode
// Implements "passthrough navigation" - arrow keys are forwarded to tree pane
func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Jump to first match and exit search mode
			m.jumpToMatch(0)
			m.searching = false
			m.searchInput.Blur()

			// CRITICAL FIX: Switch focus to the tree pane
			// If user started search from Layers pane, focus should go to Tree after Enter
			m.activePane = PaneTree
			m.sendFocusStates()

			return m, nil

		case "esc":
			// Exit search mode, clear filter
			m.searching = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.currentMatch = -1
			m.totalMatches = 0
			m.applyFilter("") // Clear filter
			// Disable flat mode
			newPane, _ := m.panes[PaneTree].Update(filetreepane.SetFlatModeMsg{Flat: false})
			m.panes[PaneTree] = newPane
			return m, nil

		// REMOVED: case "n" - was blocking input of letter 'n' (e.g., "nginx", "kernel")
		// In clean search mode, "next match" is just "down" since non-matching files are hidden

		case "up", "down":
			// PASSTHROUGH: Forward navigation keys to tree pane
			// This allows navigating the filtered tree while still in search mode
			// REMOVED: "j", "k" - they were blocking input (e.g., "json")
			if treePane, ok := m.panes[PaneTree]; ok {
				updatedPane, cmd := treePane.Update(msg)
				m.panes[PaneTree] = updatedPane
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case "ctrl+n":
			// Alternative down key (Standard CLI navigation)
			if treePane, ok := m.panes[PaneTree]; ok {
				downMsg := tea.KeyMsg{Type: tea.KeyDown}
				updatedPane, cmd := treePane.Update(downMsg)
				m.panes[PaneTree] = updatedPane
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case "ctrl+p":
			// Alternative up key (Standard CLI navigation)
			if treePane, ok := m.panes[PaneTree]; ok {
				upMsg := tea.KeyMsg{Type: tea.KeyUp}
				updatedPane, cmd := treePane.Update(upMsg)
				m.panes[PaneTree] = updatedPane
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		default:
			// Regular text input - update search field and apply filter in real-time
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)

			// Apply filter immediately (real-time filtering)
			patternText := m.searchInput.Value()

			if patternText != "" {
				m.applyFilter(patternText)
			} else {
				// Empty pattern - clear filter
				m.totalMatches = 0
				m.currentMatch = -1
			}

			return m, cmd
		}
	}

	// Update help model
	var cmd tea.Cmd
	m.help, cmd = m.help.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// jumpToMatch moves the cursor to the specified match index
func (m *Model) jumpToMatch(matchIndex int) {
	if m.totalMatches == 0 {
		return
	}

	// Ensure match index is in bounds
	if matchIndex < 0 || matchIndex >= m.totalMatches {
		return
	}

	m.currentMatch = matchIndex

	// Get match indices
	matchIndices := m.findMatchIndices()
	if matchIndex >= len(matchIndices) {
		return
	}

	// Set cursor to the match position
	targetIndex := matchIndices[matchIndex]

	// FIXED: Send message instead of type assertion
	// This respects the Elm Architecture and proper data flow
	if treePane, ok := m.panes[PaneTree]; ok {
		updatedPane, _ := treePane.Update(filetreepane.SetCursorMsg{Index: targetIndex})
		m.panes[PaneTree] = updatedPane
	}
}

