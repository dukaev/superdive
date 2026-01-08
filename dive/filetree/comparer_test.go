package filetree

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTreeIndexKey(t *testing.T) {
	t.Run("all zeros", func(t *testing.T) {
		key := NewTreeIndexKey(0, 0, 0, 0)
		assert.Equal(t, 0, key.bottomTreeStart)
		assert.Equal(t, 0, key.bottomTreeStop)
		assert.Equal(t, 0, key.topTreeStart)
		assert.Equal(t, 0, key.topTreeStop)
	})

	t.Run("with values", func(t *testing.T) {
		key := NewTreeIndexKey(1, 2, 3, 4)
		assert.Equal(t, 1, key.bottomTreeStart)
		assert.Equal(t, 2, key.bottomTreeStop)
		assert.Equal(t, 3, key.topTreeStart)
		assert.Equal(t, 4, key.topTreeStop)
	})
}

func TestTreeIndexKey_String(t *testing.T) {
	t.Run("single layer on both sides", func(t *testing.T) {
		key := NewTreeIndexKey(0, 0, 0, 0)
		assert.Equal(t, "Index(0:0)", key.String())
	})

	t.Run("single bottom, multiple top", func(t *testing.T) {
		key := NewTreeIndexKey(0, 0, 0, 3)
		assert.Equal(t, "Index(0:0-3)", key.String())
	})

	t.Run("multiple bottom, single top", func(t *testing.T) {
		key := NewTreeIndexKey(0, 3, 0, 0)
		assert.Equal(t, "Index(0-3:0)", key.String())
	})

	t.Run("multiple on both sides", func(t *testing.T) {
		key := NewTreeIndexKey(0, 2, 3, 5)
		assert.Equal(t, "Index(0-2:3-5)", key.String())
	})

	t.Run("different ranges", func(t *testing.T) {
		key := NewTreeIndexKey(1, 1, 2, 2)
		assert.Equal(t, "Index(1:2)", key.String())
	})
}

func TestNewComparer(t *testing.T) {
	t.Run("empty ref trees", func(t *testing.T) {
		cmp := NewComparer([]*FileTree{})
		assert.NotNil(t, cmp)
		assert.Empty(t, cmp.refTrees)
		assert.Empty(t, cmp.trees)
		assert.Empty(t, cmp.pathErrors)
	})

	t.Run("with ref trees", func(t *testing.T) {
		trees := []*FileTree{
			{Id: uuid.New()},
			{Id: uuid.New()},
		}
		cmp := NewComparer(trees)
		assert.NotNil(t, cmp)
		assert.Len(t, cmp.refTrees, 2)
		assert.Empty(t, cmp.trees)
		assert.Empty(t, cmp.pathErrors)
	})
}

func TestComparer_NaturalIndexes(t *testing.T) {
	t.Run("no trees", func(t *testing.T) {
		cmp := NewComparer([]*FileTree{})
		indexes := make([]TreeIndexKey, 0)

		for idx := range cmp.NaturalIndexes() {
			indexes = append(indexes, idx)
		}

		assert.Empty(t, indexes)
	})

	t.Run("single tree", func(t *testing.T) {
		trees := []*FileTree{{Id: uuid.New()}}
		cmp := NewComparer(trees)
		indexes := make([]TreeIndexKey, 0)

		for idx := range cmp.NaturalIndexes() {
			indexes = append(indexes, idx)
		}

		assert.Len(t, indexes, 1)
		assert.Equal(t, NewTreeIndexKey(0, 0, 0, 0), indexes[0])
	})

	t.Run("multiple trees", func(t *testing.T) {
		trees := []*FileTree{
			{Id: uuid.New()},
			{Id: uuid.New()},
			{Id: uuid.New()},
		}
		cmp := NewComparer(trees)
		indexes := make([]TreeIndexKey, 0)

		for idx := range cmp.NaturalIndexes() {
			indexes = append(indexes, idx)
		}

		assert.Len(t, indexes, 3)
		// Index 0: (0:0)
		assert.Equal(t, NewTreeIndexKey(0, 0, 0, 0), indexes[0])
		// Index 1: (0:1)
		assert.Equal(t, NewTreeIndexKey(0, 0, 1, 1), indexes[1])
		// Index 2: (0-1:2)
		assert.Equal(t, NewTreeIndexKey(0, 1, 2, 2), indexes[2])
	})
}

func TestComparer_AggregatedIndexes(t *testing.T) {
	t.Run("no trees", func(t *testing.T) {
		cmp := NewComparer([]*FileTree{})
		indexes := make([]TreeIndexKey, 0)

		for idx := range cmp.AggregatedIndexes() {
			indexes = append(indexes, idx)
		}

		assert.Empty(t, indexes)
	})

	t.Run("single tree", func(t *testing.T) {
		trees := []*FileTree{{Id: uuid.New()}}
		cmp := NewComparer(trees)
		indexes := make([]TreeIndexKey, 0)

		for idx := range cmp.AggregatedIndexes() {
			indexes = append(indexes, idx)
		}

		assert.Len(t, indexes, 1)
		assert.Equal(t, NewTreeIndexKey(0, 0, 0, 0), indexes[0])
	})

	t.Run("multiple trees", func(t *testing.T) {
		trees := []*FileTree{
			{Id: uuid.New()},
			{Id: uuid.New()},
			{Id: uuid.New()},
		}
		cmp := NewComparer(trees)
		indexes := make([]TreeIndexKey, 0)

		for idx := range cmp.AggregatedIndexes() {
			indexes = append(indexes, idx)
		}

		assert.Len(t, indexes, 3)
		// Index 0: (0:0)
		assert.Equal(t, NewTreeIndexKey(0, 0, 0, 0), indexes[0])
		// Index 1: (0:1) - bottom stays at 0, top starts at 1
		assert.Equal(t, NewTreeIndexKey(0, 0, 1, 1), indexes[1])
		// Index 2: (0:1-2)
		assert.Equal(t, NewTreeIndexKey(0, 0, 1, 2), indexes[2])
	})
}

func TestComparer_GetPathErrors(t *testing.T) {
	t.Run("get path errors from comparer", func(t *testing.T) {
		// Create simple ref trees
		tree1 := NewFileTree()
		tree1.Name = "tree1"

		tree2 := NewFileTree()
		tree2.Name = "tree2"

		cmp := NewComparer([]*FileTree{tree1, tree2})

		// Test getting path errors for a key
		key := NewTreeIndexKey(0, 0, 0, 0)

		pathErrors, err := cmp.GetPathErrors(key)

		// Should not error (even if tree is empty)
		assert.NoError(t, err)
		assert.NotNil(t, pathErrors)
		// Empty tree should have no path errors
		assert.Empty(t, pathErrors)
	})
}

func TestComparer_GetTree(t *testing.T) {
	t.Run("get tree from comparer", func(t *testing.T) {
		// Create simple ref trees
		tree1 := NewFileTree()
		tree1.Name = "tree1"

		tree2 := NewFileTree()
		tree2.Name = "tree2"

		cmp := NewComparer([]*FileTree{tree1, tree2})

		// Test getting tree for a key
		key := NewTreeIndexKey(0, 0, 0, 0)

		resultTree, err := cmp.GetTree(key)

		// Should not error
		assert.NoError(t, err)
		assert.NotNil(t, resultTree)
	})

	t.Run("cached tree returns same instance", func(t *testing.T) {
		tree1 := NewFileTree()
		tree1.Name = "tree1"

		cmp := NewComparer([]*FileTree{tree1})
		key := NewTreeIndexKey(0, 0, 0, 0)

		// Call GetTree twice
		tree1, err1 := cmp.GetTree(key)
		tree2, err2 := cmp.GetTree(key)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotNil(t, tree1)
		assert.NotNil(t, tree2)
		// Should return the same cached instance
		assert.Same(t, tree1, tree2)
	})
}

func TestComparer_BuildCache(t *testing.T) {
	t.Run("build cache with empty ref trees", func(t *testing.T) {
		cmp := NewComparer([]*FileTree{})

		errors := cmp.BuildCache()

		// Should not error
		assert.Empty(t, errors)
	})

	t.Run("build cache with single tree", func(t *testing.T) {
		tree := NewFileTree()
		tree.Name = "tree1"

		cmp := NewComparer([]*FileTree{tree})

		errors := cmp.BuildCache()

		// Should not error
		assert.Empty(t, errors)
	})

	t.Run("build cache with multiple trees", func(t *testing.T) {
		trees := []*FileTree{
			NewFileTree(),
			NewFileTree(),
			NewFileTree(),
		}

		for i, tree := range trees {
			tree.Name = fmt.Sprintf("tree%d", i)
		}

		cmp := NewComparer(trees)

		errors := cmp.BuildCache()

		// Should not error
		assert.Empty(t, errors)
	})
}

func TestEfficiencySlice_Len(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		efs := make(EfficiencySlice, 0)
		assert.Equal(t, 0, efs.Len())
	})

	t.Run("non-empty slice", func(t *testing.T) {
		efs := EfficiencySlice{
			&EfficiencyData{Path: "/path1"},
			&EfficiencyData{Path: "/path2"},
		}
		assert.Equal(t, 2, efs.Len())
	})
}

func TestEfficiencySlice_Swap(t *testing.T) {
	efs := EfficiencySlice{
		&EfficiencyData{Path: "/path1", CumulativeSize: 100},
		&EfficiencyData{Path: "/path2", CumulativeSize: 200},
	}

	efs.Swap(0, 1)

	assert.Equal(t, "/path2", efs[0].Path)
	assert.Equal(t, int64(200), efs[0].CumulativeSize)
	assert.Equal(t, "/path1", efs[1].Path)
	assert.Equal(t, int64(100), efs[1].CumulativeSize)
}

func TestEfficiencySlice_Less(t *testing.T) {
	t.Run("first is smaller", func(t *testing.T) {
		efs := EfficiencySlice{
			&EfficiencyData{Path: "/path1", CumulativeSize: 100},
			&EfficiencyData{Path: "/path2", CumulativeSize: 200},
		}
		assert.True(t, efs.Less(0, 1))
	})

	t.Run("second is smaller", func(t *testing.T) {
		efs := EfficiencySlice{
			&EfficiencyData{Path: "/path1", CumulativeSize: 200},
			&EfficiencyData{Path: "/path2", CumulativeSize: 100},
		}
		assert.False(t, efs.Less(0, 1))
	})

	t.Run("equal sizes", func(t *testing.T) {
		efs := EfficiencySlice{
			&EfficiencyData{Path: "/path1", CumulativeSize: 100},
			&EfficiencyData{Path: "/path2", CumulativeSize: 100},
		}
		assert.False(t, efs.Less(0, 1))
	})
}
