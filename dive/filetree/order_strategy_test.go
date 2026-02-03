package filetree

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSortOrderStrategy(t *testing.T) {
	t.Run("ByName strategy", func(t *testing.T) {
		strategy := GetSortOrderStrategy(ByName)
		assert.IsType(t, orderByNameStrategy{}, strategy)
	})

	t.Run("BySizeDesc strategy", func(t *testing.T) {
		strategy := GetSortOrderStrategy(BySizeDesc)
		assert.IsType(t, orderBySizeDescStrategy{}, strategy)
	})

	t.Run("invalid value defaults to ByName", func(t *testing.T) {
		strategy := GetSortOrderStrategy(SortOrder(99))
		assert.IsType(t, orderByNameStrategy{}, strategy)
	})
}

func TestOrderByNameStrategy_OrderKeys(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		strategy := orderByNameStrategy{}
		files := make(map[string]*FileNode)

		result := strategy.orderKeys(files)

		assert.Empty(t, result)
	})

	t.Run("single file", func(t *testing.T) {
		strategy := orderByNameStrategy{}
		files := map[string]*FileNode{
			"file.txt": {},
		}

		result := strategy.orderKeys(files)

		assert.Len(t, result, 1)
		assert.Equal(t, "file.txt", result[0])
	})

	t.Run("multiple files sorted alphabetically", func(t *testing.T) {
		strategy := orderByNameStrategy{}
		files := map[string]*FileNode{
			"zebra.txt": {},
			"apple.txt": {},
			"banana.txt": {},
		}

		result := strategy.orderKeys(files)

		assert.Equal(t, []string{"apple.txt", "banana.txt", "zebra.txt"}, result)
	})

	t.Run("files with similar names", func(t *testing.T) {
		strategy := orderByNameStrategy{}
		files := map[string]*FileNode{
			"file1.txt":  {},
			"file2.txt":  {},
			"file10.txt": {},
			"file20.txt": {},
		}

		result := strategy.orderKeys(files)

		// Lexicographic sort (not numeric)
		assert.Equal(t, []string{"file1.txt", "file10.txt", "file2.txt", "file20.txt"}, result)
	})

	t.Run("files with paths", func(t *testing.T) {
		strategy := orderByNameStrategy{}
		files := map[string]*FileNode{
			"/usr/bin/file":  {},
			"/etc/config":    {},
			"/var/log/app":   {},
			"/home/user/doc": {},
		}

		result := strategy.orderKeys(files)

		assert.Equal(t, []string{"/etc/config", "/home/user/doc", "/usr/bin/file", "/var/log/app"}, result)
	})

	t.Run("case sensitive sorting", func(t *testing.T) {
		strategy := orderByNameStrategy{}
		files := map[string]*FileNode{
			"FILE.TXT": {},
			"file.txt": {},
			"File.Txt": {},
		}

		result := strategy.orderKeys(files)

		// Uppercase comes before lowercase in ASCII
		assert.Equal(t, []string{"FILE.TXT", "File.Txt", "file.txt"}, result)
	})
}

func TestOrderBySizeDescStrategy_OrderKeys(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		strategy := orderBySizeDescStrategy{}
		files := make(map[string]*FileNode)

		result := strategy.orderKeys(files)

		assert.Empty(t, result)
	})

	t.Run("single file", func(t *testing.T) {
		strategy := orderBySizeDescStrategy{}
		node := &FileNode{}
		node.Size = 1024
		files := map[string]*FileNode{
			"file.txt": node,
		}

		result := strategy.orderKeys(files)

		assert.Len(t, result, 1)
		assert.Equal(t, "file.txt", result[0])
	})

	t.Run("multiple files sorted by size descending", func(t *testing.T) {
		strategy := orderBySizeDescStrategy{}

		smallNode := &FileNode{}
		smallNode.Size = 100

		mediumNode := &FileNode{}
		mediumNode.Size = 500

		largeNode := &FileNode{}
		largeNode.Size = 1000

		files := map[string]*FileNode{
			"small.txt":  smallNode,
			"large.txt":  largeNode,
			"medium.txt": mediumNode,
		}

		result := strategy.orderKeys(files)

		assert.Equal(t, []string{"large.txt", "medium.txt", "small.txt"}, result)
	})

	t.Run("files with same size sorted alphabetically", func(t *testing.T) {
		strategy := orderBySizeDescStrategy{}

		node1 := &FileNode{}
		node1.Size = 500

		node2 := &FileNode{}
		node2.Size = 500

		node3 := &FileNode{}
		node3.Size = 500

		files := map[string]*FileNode{
			"zebra.txt":  node1,
			"apple.txt":  node2,
			"banana.txt": node3,
		}

		result := strategy.orderKeys(files)

		assert.Equal(t, []string{"apple.txt", "banana.txt", "zebra.txt"}, result)
	})

	t.Run("files with zero size", func(t *testing.T) {
		strategy := orderBySizeDescStrategy{}

		zeroNode1 := &FileNode{}
		zeroNode1.Size = 0

		zeroNode2 := &FileNode{}
		zeroNode2.Size = 0

		smallNode := &FileNode{}
		smallNode.Size = 100

		files := map[string]*FileNode{
			"file1.txt": zeroNode1,
			"file2.txt": zeroNode2,
			"file3.txt": smallNode,
		}

		result := strategy.orderKeys(files)

		// file3.txt comes first (100 bytes), then file1.txt and file2.txt (both 0 bytes, alphabetically)
		assert.Equal(t, []string{"file3.txt", "file1.txt", "file2.txt"}, result)
	})

	t.Run("large and small files mixed", func(t *testing.T) {
		strategy := orderBySizeDescStrategy{}

		// Create explicit files with known sizes
		nodeA := &FileNode{}
		nodeA.Size = 10

		nodeB := &FileNode{}
		nodeB.Size = 1000

		nodeC := &FileNode{}
		nodeC.Size = 500

		nodeD := &FileNode{}
		nodeD.Size = 50

		nodeE := &FileNode{}
		nodeE.Size = 5000

		nodeF := &FileNode{}
		nodeF.Size = 100

		nodes := map[string]*FileNode{
			"a.txt": nodeA,
			"b.txt": nodeB,
			"c.txt": nodeC,
			"d.txt": nodeD,
			"e.txt": nodeE,
			"f.txt": nodeF,
		}

		result := strategy.orderKeys(nodes)

		// Verify descending order by size
		sizesResult := make([]int64, len(result))
		for i, key := range result {
			sizesResult[i] = nodes[key].Size
		}

		// Check that sizes are in descending order
		assert.Equal(t, []int64{5000, 1000, 500, 100, 50, 10}, sizesResult)
	})
}
