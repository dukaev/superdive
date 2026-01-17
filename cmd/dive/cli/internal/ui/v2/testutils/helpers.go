package testutils

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	"github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
	"github.com/wagoodman/dive/dive/image/docker"
)

var repoRootCache string

// TestImageData provides test image data for snapshot tests
type TestImageData struct {
	Analysis *image.Analysis
	TreeVM   *viewmodel.FileTreeViewModel
	Comparer filetree.Comparer
}

// repoRoot returns the root directory of the git repository
func repoRoot(t testing.TB) string {
	t.Helper()
	if repoRootCache != "" {
		return repoRootCache
	}
	// use git to find the root of the repo
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("failed to get repo root: %v", err)
	}
	repoRootCache = strings.TrimSpace(string(out))
	return repoRootCache
}

// repoPath returns a path relative to the repository root
func repoPath(t testing.TB, path string) string {
	t.Helper()
	root := repoRoot(t)
	return root + "/" + strings.TrimPrefix(path, "./")
}

// LoadTestImage loads test image data from the test archive
func LoadTestImage(t *testing.T) TestImageData {
	t.Helper()

	// Load test image - use repoPath to get absolute path
	result := docker.TestAnalysisFromArchive(t, repoPath(t, ".data/test-docker-image.tar"))
	require.NotNil(t, result, "unable to load test data")

	// Create filetree viewmodel
	vm, err := viewmodel.NewFileTreeViewModel(
		v1.Config{
			Analysis:    *result,
			Preferences: v1.DefaultPreferences(),
		},
		0,
	)
	require.NoError(t, err, "unable to create viewmodel")

	// Initialize ViewTree by calling Update (this sets ViewTree = ModelTree.Copy())
	err = vm.Update(nil, 100, 100)
	require.NoError(t, err, "unable to update viewmodel")

	// Get comparer
	comparer := filetree.NewComparer(result.RefTrees)
	errs := comparer.BuildCache()
	require.Empty(t, errs, "unable to build comparer")

	return TestImageData{
		Analysis: result,
		TreeVM:   vm,
		Comparer: comparer,
	}
}

// TestLayout provides a test layout message for panes
func TestLayout() common.LayoutMsg {
	return common.LayoutMsg{
		LeftWidth:     50,
		RightWidth:    50,
		LayersHeight:  10,
		DetailsHeight: 8,
		ImageHeight:   12,
		TreeHeight:    15,
	}
}
