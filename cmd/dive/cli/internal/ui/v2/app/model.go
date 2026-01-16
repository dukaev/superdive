package app

import (
	"context"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
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

// mapLayoutPaneToAppPane safely converts layout.PaneID to app.Pane
// This prevents bugs if the order of constants changes in either package
func mapLayoutPaneToAppPane(id layout.PaneID) Pane {
	switch id {
	case layout.PaneIDLayer:
		return PaneLayer
	case layout.PaneIDDetails:
		return PaneDetails
	case layout.PaneIDImage:
		return PaneImage
	case layout.PaneIDTree:
		return PaneTree
	default:
		// Fallback to Layers if unknown
		return PaneLayer
	}
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

	// Filter state
	filter       FilterModel
	filterRegex  *regexp.Regexp // Compiled regex for tree filtering

	// Layer detail modal
	layerDetailModal LayerDetailModal

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

	f := NewFilterModel()
	layerDetailModal := NewLayerDetailModal()

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
		keys:             keys.Keys,
		help:             h,
		filter:           f,
		layerDetailModal: layerDetailModal,
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
		// If layer detail modal is visible, let it handle keys first
		if m.layerDetailModal.IsVisible() {
			var cmd tea.Cmd
			m.layerDetailModal, cmd = m.layerDetailModal.Update(msg)
			cmds = append(cmds, cmd)
			break
		}

		// If filter is visible, let filter handle keys
		if m.filter.IsVisible() {
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			cmds = append(cmds, cmd)
			break
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
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "tab", "shift+tab":
			m.togglePane()

		case "ctrl+f", "/":
			m.filter.Show()
		}

	case FilterAppliedMsg:
		// User applied a filter pattern
		m.applyFilter(msg.Pattern)

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
		// NOTE: SetTreeVM is tree-specific, so we need a type assertion here
		// This is acceptable since it's a one-time operation for tree-specific functionality
		if treePane, ok := m.panes[PaneTree].(*filetreepane.Pane); ok {
			treePane.SetTreeVM(m.treeVM)
		}

	case layers.ShowLayerDetailMsg:
		// Show layer detail modal
		m.layerDetailModal.Show(msg.Layer)

	case tea.MouseMsg:
		// Layout engine determines which pane was clicked and provides local coordinates
		// This encapsulates hit testing logic in the layout layer
		paneID, localX, localY, found := m.layout.GetPaneAt(msg.X, msg.Y, m.width)

		if found {
			// SAFETY: Use explicit mapping instead of type conversion
			// This prevents bugs if constant order changes in either package
			targetPane := mapLayoutPaneToAppPane(paneID)

			// Change focus if needed
			if m.activePane != targetPane {
				m.activePane = targetPane
				m.sendFocusStates()
			}

			// Skip Details pane (read-only, no mouse handling)
			if targetPane != PaneDetails {
				// Create local mouse message with transformed coordinates
				localMsg := common.LocalMouseMsg{
					MouseMsg: msg,
					LocalX:   localX,
					LocalY:   localY,
				}
				// POLYMORPHISM: Update through interface
				newPane, cmd := m.panes[targetPane].Update(localMsg)
				m.panes[targetPane] = newPane
				cmds = append(cmds, cmd)
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
	// NOTE: SetTreeVM is tree-specific, so we need a type assertion here
	if treePane, ok := m.panes[PaneTree].(*filetreepane.Pane); ok {
		treePane.SetTreeVM(m.treeVM)
	}
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
	statusBar := m.help.View(m.keys)

	// Render panes directly using their View() methods
	// POLYMORPHISM: Access panes through interface from map
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		m.panes[PaneLayer].View(),
		m.panes[PaneDetails].View(),
		m.panes[PaneImage].View(),
	)
	treePane := m.panes[PaneTree].View()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, treePane)

	base := lipgloss.JoinVertical(lipgloss.Left,
		mainContent,
		statusBar,
	)

	// Overlay layer detail modal if visible
	if m.layerDetailModal.IsVisible() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.layerDetailModal.View(m.width, m.height))
	}

	// Overlay filter modal if visible
	if m.filter.IsVisible() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.filter.View(m.width, m.height))
	}

	return base
}

// applyFilter applies a filter pattern to the file tree
func (m *Model) applyFilter(pattern string) {
	if m.treeVM == nil {
		return
	}

	// Compile the regex pattern
	var filterRegex *regexp.Regexp
	if pattern != "" {
		filterRegex = regexp.MustCompile(pattern)
	}
	m.filterRegex = filterRegex

	// Update the tree viewmodel with the filter
	// This will update ViewTree based on the filter
	_ = m.treeVM.Update(filterRegex, m.layout.RightWidth, m.layout.TreeHeight)

	// Update tree pane with filtered tree data
	// NOTE: SetTreeVM is tree-specific, so we need a type assertion here
	if treePane, ok := m.panes[PaneTree].(*filetreepane.Pane); ok {
		treePane.SetTreeVM(m.treeVM)
	}
}

