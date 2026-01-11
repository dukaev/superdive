package app


// LayerChangedMsg is sent when the active layer changes
type LayerChangedMsg struct {
	LayerIndex int
}

// NodeToggledMsg is sent when a tree node is collapsed/expanded
type NodeToggledMsg struct {
	NodeIndex int
}

// PaneChangedMsg is sent when the active pane changes
type PaneChangedMsg struct {
	Pane Pane
}

// LayerSelectionChangedMsg is sent when a layer is selected (via click or keyboard)
type LayerSelectionChangedMsg struct {
	LayerIndex int
}

// TreeSelectionChangedMsg is sent when a tree node is selected
type TreeSelectionChangedMsg struct {
	NodeIndex int
}

// PaneFocusRequestMsg requests focus to be moved to a specific pane
type PaneFocusRequestMsg struct {
	Pane Pane
}

// RefreshTreeContentMsg requests tree content to be refreshed
type RefreshTreeContentMsg struct {
	LayerIndex int
}
