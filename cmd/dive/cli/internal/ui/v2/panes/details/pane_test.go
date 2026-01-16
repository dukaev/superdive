package details

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/testutils"
	"github.com/wagoodman/dive/dive/image"
)

func TestPane_View_NoLayer(t *testing.T) {
	pane := New()
	pane.SetSize(50, 10)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_WithLayer(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Get first layer
	if len(testData.Analysis.Layers) == 0 {
		t.Skip("No layers in test data")
	}

	layer := testData.Analysis.Layers[0]

	pane := New()
	pane.SetLayer(layer)
	pane.SetSize(80, 15)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_Focused(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Get first layer
	if len(testData.Analysis.Layers) == 0 {
		t.Skip("No layers in test data")
	}

	layer := testData.Analysis.Layers[0]

	pane := New()
	pane.SetLayer(layer)
	pane.SetSize(80, 15)

	// Send focus message
	updatedPane, _ := pane.Update(FocusStateMsg{Focused: true})

	view := updatedPane.(Pane).View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallWidth(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Get first layer
	if len(testData.Analysis.Layers) == 0 {
		t.Skip("No layers in test data")
	}

	layer := testData.Analysis.Layers[0]

	pane := New()
	pane.SetLayer(layer)
	pane.SetSize(30, 15) // Very narrow width

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallHeight(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Get first layer
	if len(testData.Analysis.Layers) == 0 {
		t.Skip("No layers in test data")
	}

	layer := testData.Analysis.Layers[0]

	pane := New()
	pane.SetLayer(layer)
	pane.SetSize(80, 6) // Very short height

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_LargeSize(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Get first layer
	if len(testData.Analysis.Layers) == 0 {
		t.Skip("No layers in test data")
	}

	layer := testData.Analysis.Layers[0]

	pane := New()
	pane.SetLayer(layer)
	pane.SetSize(120, 30) // Large dimensions

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_LongCommand(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Find a layer with a long command
	var targetLayer *image.Layer
	for _, layer := range testData.Analysis.Layers {
		if len(layer.Command) > 100 {
			targetLayer = layer
			break
		}
	}

	if targetLayer == nil {
		t.Skip("No layer with long command found")
	}

	pane := New()
	pane.SetLayer(targetLayer)
	pane.SetSize(80, 15)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_Update_WithLayoutMsg(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Get first layer
	if len(testData.Analysis.Layers) == 0 {
		t.Skip("No layers in test data")
	}

	layer := testData.Analysis.Layers[0]

	pane := New()
	pane.SetLayer(layer)

	// Send layout message
	layoutMsg := testutils.TestLayout()
	updatedPane, _ := pane.Update(layoutMsg)

	// Verify size was updated
	view := updatedPane.(Pane).View()
	snaps.MatchSnapshot(t, view)
}
