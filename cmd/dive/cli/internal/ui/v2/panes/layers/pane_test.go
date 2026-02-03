package layers

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/testutils"
)

func TestPane_View_EmptyState(t *testing.T) {
	// Test with nil layerVM
	pane := New(nil, testutils.LoadTestImage(t).Comparer)
	pane.Resize(50, 20)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_WithLayers(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)
	pane.Resize(80, 20)

	// Initialize the pane
	cmd := pane.Init()
	require.Nil(t, cmd)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_Focused(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)
	pane.Resize(80, 20)

	// Send focus message
	updatedPane, _ := pane.Update(FocusStateMsg{Focused: true})

	view := updatedPane.(*Pane).View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallWidth(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)
	pane.Resize(40, 20) // Narrow width

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallHeight(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)
	pane.Resize(80, 5) // Very short height

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_LargeSize(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)
	pane.Resize(120, 40) // Large dimensions

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SecondLayerSelected(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel with second layer selected
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 1, // Select second layer
	}

	pane := New(layerVM, testData.Comparer)
	pane.Resize(80, 20)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_Update_WithLayoutMsg(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)

	// Send layout message
	layoutMsg := testutils.TestLayout()
	updatedPane, _ := pane.Update(layoutMsg)

	// Verify size was updated
	view := updatedPane.(*Pane).View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_ShortHelp(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	// Create layer viewmodel
	layerVM := &viewmodel.LayerSetState{
		Layers:     testData.Analysis.Layers,
		LayerIndex: 0,
	}

	pane := New(layerVM, testData.Comparer)

	// Test that ShortHelp returns empty list (no special keys for Layers pane)
	keys := pane.ShortHelp()
	require.NotNil(t, keys)
	require.Len(t, keys, 0)
}
