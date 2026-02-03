//revive:disable:var-naming
package common

import (
	tea "github.com/charmbracelet/bubbletea"
)

// LocalMouseMsg is a mouse message with coordinates transformed to local pane space
// (0, 0) is the top-left corner of the pane's content area (inside borders)
// Parent model is responsible for coordinate transformations - children receive local coords
type LocalMouseMsg struct {
	tea.MouseMsg
	// Pane-relative coordinates (already transformed by parent)
	LocalX int
	LocalY int
}
