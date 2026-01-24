// Package common provides shared types and messages for UI panes
//
//revive:disable:var-naming
package common

// LayoutMsg contains pane dimensions calculated by the parent
// This replaces direct SetSize() calls with message passing
type LayoutMsg struct {
	// For left column panes (Layers, Details, Image)
	LeftWidth     int
	LayersHeight  int
	DetailsHeight int
	ImageHeight   int

	// For right column pane (Tree)
	RightWidth int
	TreeHeight int
}
