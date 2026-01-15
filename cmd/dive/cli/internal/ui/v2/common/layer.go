package common

import (
	"github.com/wagoodman/dive/dive/image"
)

// LayerSelectedMsg is sent when a layer is selected
// This replaces direct SetLayer() calls with message passing
type LayerSelectedMsg struct {
	Layer     *image.Layer
	LayerIndex int
}
