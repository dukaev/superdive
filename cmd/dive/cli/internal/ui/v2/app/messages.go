// Package app provides the main application model and message types
package app

// Note: LocalMouseMsg is now defined in common package to avoid import cycles
// Note: LayerChangedMsg is now defined in panes/layers package
// Note: NodeToggledMsg, TreeSelectionChangedMsg, RefreshTreeContentMsg are now defined in panes/filetree package

// The following message types are kept for potential future use or for app-level coordination

// PaneChangedMsg is sent when the active pane changes (for future use)
type PaneChangedMsg struct {
	Pane Pane
}

// LayerSelectionChangedMsg is sent when a layer is selected (for future use)
type LayerSelectionChangedMsg struct {
	LayerIndex int
}

// PaneFocusRequestMsg requests focus to be moved to a specific pane (for future use)
type PaneFocusRequestMsg struct {
	Pane Pane
}
