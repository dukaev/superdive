package filetree

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Tests for diff.go

func TestDiffType_String(t *testing.T) {
	t.Run("Unmodified", func(t *testing.T) {
		assert.Equal(t, "Unmodified", Unmodified.String())
	})

	t.Run("Modified", func(t *testing.T) {
		assert.Equal(t, "Modified", Modified.String())
	})

	t.Run("Added", func(t *testing.T) {
		assert.Equal(t, "Added", Added.String())
	})

	t.Run("Removed", func(t *testing.T) {
		assert.Equal(t, "Removed", Removed.String())
	})

	t.Run("unknown value", func(t *testing.T) {
		unknownDiff := DiffType(99)
		assert.Equal(t, "99", unknownDiff.String())
	})
}

func TestDiffType_Merge(t *testing.T) {
	t.Run("same values - Unmodified", func(t *testing.T) {
		result := Unmodified.merge(Unmodified)
		assert.Equal(t, Unmodified, result)
	})

	t.Run("same values - Modified", func(t *testing.T) {
		result := Modified.merge(Modified)
		assert.Equal(t, Modified, result)
	})

	t.Run("same values - Added", func(t *testing.T) {
		result := Added.merge(Added)
		assert.Equal(t, Added, result)
	})

	t.Run("same values - Removed", func(t *testing.T) {
		result := Removed.merge(Removed)
		assert.Equal(t, Removed, result)
	})

	t.Run("different values - Added and Removed", func(t *testing.T) {
		result := Added.merge(Removed)
		assert.Equal(t, Modified, result)
	})

	t.Run("different values - Unmodified and Added", func(t *testing.T) {
		result := Unmodified.merge(Added)
		assert.Equal(t, Modified, result)
	})

	t.Run("different values - Removed and Modified", func(t *testing.T) {
		result := Removed.merge(Modified)
		assert.Equal(t, Modified, result)
	})

	t.Run("different values - Added and Unmodified", func(t *testing.T) {
		result := Added.merge(Unmodified)
		assert.Equal(t, Modified, result)
	})
}

// Tests for path_error.go

func TestFileAction_String(t *testing.T) {
	t.Run("ActionAdd", func(t *testing.T) {
		assert.Equal(t, "add", ActionAdd.String())
	})

	t.Run("ActionRemove", func(t *testing.T) {
		assert.Equal(t, "remove", ActionRemove.String())
	})

	t.Run("unknown value", func(t *testing.T) {
		unknownAction := FileAction(99)
		assert.Equal(t, "<unknown file action>", unknownAction.String())
	})
}

func TestNewPathError(t *testing.T) {
	t.Run("create path error with all fields", func(t *testing.T) {
		err := errors.New("test error")
		pathErr := NewPathError("/test/path", ActionAdd, err)

		assert.Equal(t, "/test/path", pathErr.Path)
		assert.Equal(t, ActionAdd, pathErr.Action)
		assert.Equal(t, err, pathErr.Err)
	})

	t.Run("create path error with remove action", func(t *testing.T) {
		err := errors.New("remove error")
		pathErr := NewPathError("/old/path", ActionRemove, err)

		assert.Equal(t, "/old/path", pathErr.Path)
		assert.Equal(t, ActionRemove, pathErr.Action)
		assert.Equal(t, err, pathErr.Err)
	})

	t.Run("create path error with nil error", func(t *testing.T) {
		pathErr := NewPathError("/test/path", ActionAdd, nil)

		assert.Equal(t, "/test/path", pathErr.Path)
		assert.Equal(t, ActionAdd, pathErr.Action)
		assert.Nil(t, pathErr.Err)
	})
}

func TestPathError_String(t *testing.T) {
	t.Run("with add action", func(t *testing.T) {
		err := errors.New("file not found")
		pathErr := NewPathError("/test/file.txt", ActionAdd, err)

		expected := "unable to add '/test/file.txt': file not found"
		assert.Equal(t, expected, pathErr.String())
	})

	t.Run("with remove action", func(t *testing.T) {
		err := errors.New("permission denied")
		pathErr := NewPathError("/test/file.txt", ActionRemove, err)

		expected := "unable to remove '/test/file.txt': permission denied"
		assert.Equal(t, expected, pathErr.String())
	})

	t.Run("with nil error", func(t *testing.T) {
		pathErr := NewPathError("/test/file.txt", ActionAdd, nil)

		expected := "unable to add '/test/file.txt': <nil>"
		assert.Equal(t, expected, pathErr.String())
	})

	t.Run("with complex path", func(t *testing.T) {
		err := errors.New("disk full")
		pathErr := NewPathError("/very/long/path/to/file.txt", ActionAdd, err)

		expected := "unable to add '/very/long/path/to/file.txt': disk full"
		assert.Equal(t, expected, pathErr.String())
	})
}

// Tests for node_data.go

func TestNewNodeData(t *testing.T) {
	t.Run("creates new node data with defaults", func(t *testing.T) {
		data := NewNodeData()

		assert.NotNil(t, data)
		assert.Equal(t, Unmodified, data.DiffType)
		// ViewInfo and FileInfo should have their zero values
		assert.NotNil(t, data)
	})
}

func TestNodeData_Copy(t *testing.T) {
	t.Run("copy node data", func(t *testing.T) {
		original := NewNodeData()
		original.DiffType = Added
		original.FileInfo.Size = 1024

		copied := original.Copy()

		assert.NotNil(t, copied)
		assert.Equal(t, original.DiffType, copied.DiffType)
		assert.Equal(t, original.FileInfo.Size, copied.FileInfo.Size)

		// Verify it's a deep copy
		copied.DiffType = Modified
		assert.Equal(t, Added, original.DiffType)
	})

	t.Run("copy creates new instance", func(t *testing.T) {
		original := NewNodeData()
		original.DiffType = Removed
		original.FileInfo.Path = "/test/path"

		copied := original.Copy()

		// Pointers should be different
		assert.NotSame(t, original, copied)

		// But values should be the same
		assert.Equal(t, original.DiffType, copied.DiffType)
		assert.Equal(t, original.FileInfo.Path, copied.FileInfo.Path)
	})

	t.Run("copy with unmodified diff type", func(t *testing.T) {
		original := NewNodeData()
		original.DiffType = Unmodified
		original.FileInfo.Size = 2048

		copied := original.Copy()

		assert.Equal(t, Unmodified, copied.DiffType)
		assert.NotSame(t, original, copied)
		assert.Equal(t, int64(2048), copied.FileInfo.Size)
	})
}
