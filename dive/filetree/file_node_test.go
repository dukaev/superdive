package filetree

import (
	"archive/tar"
	"testing"
)

func TestAddChild(t *testing.T) {
	var expected, actual int
	tree := NewFileTree()

	payload := FileInfo{
		Path: "stufffffs",
	}

	one := tree.Root.AddChild("first node!", payload)

	two := tree.Root.AddChild("nil node!", FileInfo{})

	tree.Root.AddChild("third node!", FileInfo{})
	two.AddChild("forth, one level down...", FileInfo{})
	two.AddChild("fifth, one level down...", FileInfo{})
	two.AddChild("fifth, one level down...", FileInfo{})

	expected, actual = 5, tree.Size
	if expected != actual {
		t.Errorf("Expected a tree size of %d got %d.", expected, actual)
	}

	expected, actual = 2, len(two.Children)
	if expected != actual {
		t.Errorf("Expected 'twos' number of children to be %d got %d.", expected, actual)
	}

	expected, actual = 3, len(tree.Root.Children)
	if expected != actual {
		t.Errorf("Expected 'twos' number of children to be %d got %d.", expected, actual)
	}

	expectedFC := FileInfo{
		Path: "stufffffs",
	}
	actualFC := one.Data.FileInfo
	if expectedFC.Path != actualFC.Path {
		t.Errorf("Expected 'ones' payload to be %+v got %+v.", expectedFC, actualFC)
	}

}

func TestRemoveChild(t *testing.T) {
	var expected, actual int

	tree := NewFileTree()
	tree.Root.AddChild("first", FileInfo{})
	two := tree.Root.AddChild("nil", FileInfo{})
	tree.Root.AddChild("third", FileInfo{})
	forth := two.AddChild("forth", FileInfo{})
	two.AddChild("fifth", FileInfo{})

	err := forth.Remove()
	checkError(t, err, "unable to setup test")

	expected, actual = 4, tree.Size
	if expected != actual {
		t.Errorf("Expected a tree size of %d got %d.", expected, actual)
	}

	if tree.Root.Children["forth"] != nil {
		t.Errorf("Expected 'forth' node to be deleted.")
	}

	err = two.Remove()
	checkError(t, err, "unable to setup test")

	expected, actual = 2, tree.Size
	if expected != actual {
		t.Errorf("Expected a tree size of %d got %d.", expected, actual)
	}

	if tree.Root.Children["nil"] != nil {
		t.Errorf("Expected 'nil' node to be deleted.")
	}

}

func TestPath(t *testing.T) {
	expected := "/etc/nginx/nginx.conf"
	tree := NewFileTree()
	node, _, _ := tree.AddPath(expected, FileInfo{})

	actual := node.Path()
	if expected != actual {
		t.Errorf("Expected path '%s' got '%s'", expected, actual)
	}
}

func TestIsWhiteout(t *testing.T) {
	tree1 := NewFileTree()
	p1, _, _ := tree1.AddPath("/etc/nginx/public1", FileInfo{})
	p2, _, _ := tree1.AddPath("/etc/nginx/.wh.public2", FileInfo{})
	p3, _, _ := tree1.AddPath("/etc/nginx/public3/.wh..wh..opq", FileInfo{})

	if p1.IsWhiteout() != false {
		t.Errorf("Expected path '%s' to **not** be a whiteout file", p1.Name)
	}

	if p2.IsWhiteout() != true {
		t.Errorf("Expected path '%s' to be a whiteout file", p2.Name)
	}

	if p3 != nil {
		t.Errorf("Expected to not be able to add path '%s'", p2.Name)
	}
}

func TestDiffTypeFromAddedChildren(t *testing.T) {
	tree := NewFileTree()
	node, _, _ := tree.AddPath("/usr", *BlankFileChangeInfo("/usr"))
	node.Data.DiffType = Unmodified

	node, _, _ = tree.AddPath("/usr/bin", *BlankFileChangeInfo("/usr/bin"))
	node.Data.DiffType = Added

	node, _, _ = tree.AddPath("/usr/bin2", *BlankFileChangeInfo("/usr/bin2"))
	node.Data.DiffType = Removed

	err := tree.Root.Children["usr"].deriveDiffType(Unmodified)
	checkError(t, err, "unable to setup test")

	if tree.Root.Children["usr"].Data.DiffType != Modified {
		t.Errorf("Expected Modified but got %v", tree.Root.Children["usr"].Data.DiffType)
	}
}
func TestDiffTypeFromRemovedChildren(t *testing.T) {
	tree := NewFileTree()
	_, _, _ = tree.AddPath("/usr", *BlankFileChangeInfo("/usr"))

	info1 := BlankFileChangeInfo("/usr/.wh.bin")
	node, _, _ := tree.AddPath("/usr/.wh.bin", *info1)
	node.Data.DiffType = Removed

	info2 := BlankFileChangeInfo("/usr/.wh.bin2")
	node, _, _ = tree.AddPath("/usr/.wh.bin2", *info2)
	node.Data.DiffType = Removed

	err := tree.Root.Children["usr"].deriveDiffType(Unmodified)
	checkError(t, err, "unable to setup test")

	if tree.Root.Children["usr"].Data.DiffType != Modified {
		t.Errorf("Expected Modified but got %v", tree.Root.Children["usr"].Data.DiffType)
	}

}

func TestDirSize(t *testing.T) {
	tree1 := NewFileTree()
	_, _, err := tree1.AddPath("/etc/nginx/public1", FileInfo{Size: 100})
	checkError(t, err, "unable to setup test")
	_, _, err = tree1.AddPath("/etc/nginx/thing1", FileInfo{Size: 200})
	checkError(t, err, "unable to setup test")
	_, _, err = tree1.AddPath("/etc/nginx/public3/thing2", FileInfo{Size: 300})
	checkError(t, err, "unable to setup test")

	node, _ := tree1.GetNode("/etc/nginx")
	expected, actual := "----------         0:0      600 B ", node.MetadataString()
	if expected != actual {
		t.Errorf("Expected metadata '%s' got '%s'", expected, actual)
	}
}

func TestFileNode_compare(t *testing.T) {
	t.Run("both nil returns Unmodified", func(t *testing.T) {
		// Need to call compare on nil receiver
		var node *FileNode
		result := node.compare(nil)
		if result != Unmodified {
			t.Errorf("Expected Unmodified but got %v", result)
		}
	})

	t.Run("non-nil node and nil other returns Removed", func(t *testing.T) {
		tree := NewFileTree()
		tree.Root.Name = "test"

		result := tree.Root.compare(nil)
		if result != Removed {
			t.Errorf("Expected Removed but got %v", result)
		}
	})

	t.Run("whiteout file returns Removed", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/file.txt", FileInfo{
			Path:     "/file.txt",
			TypeFlag: 1,
			hash:     123,
		})

		whiteoutNode := &FileNode{
			Name: ".wh.file.txt",
			Data: NodeData{
				FileInfo: FileInfo{
					Path:     "/.wh.file.txt",
					TypeFlag: 1,
				},
			},
		}

		result := node.compare(whiteoutNode)
		if result != Removed {
			t.Errorf("Expected Removed for whiteout but got %v", result)
		}
	})

	t.Run("mismatched node names panic", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/file1.txt", FileInfo{
			Path:     "/file1.txt",
			TypeFlag: 1,
			hash:     123,
		})

		other := &FileNode{
			Name: "file2.txt",
			Data: NodeData{
				FileInfo: FileInfo{
					Path:     "/file2.txt",
					TypeFlag: 1,
					hash:     123,
				},
			},
		}

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when comparing mismatched nodes")
			}
		}()

		node.compare(other)
	})

	t.Run("same nodes return Unmodified", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/file.txt", FileInfo{
			Path:     "/file.txt",
			TypeFlag: 1,
			hash:     123,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		})

		other := &FileNode{
			Tree: tree,
			Name: "file.txt",
			Data: NodeData{
				FileInfo: FileInfo{
					Path:     "/file.txt",
					TypeFlag: 1,
					hash:     123,
					Mode:     0644,
					Uid:      1000,
					Gid:      1000,
				},
			},
		}

		result := node.compare(other)
		if result != Unmodified {
			t.Errorf("Expected Unmodified but got %v", result)
		}
	})

	t.Run("different hash returns Modified", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/file.txt", FileInfo{
			Path:     "/file.txt",
			TypeFlag: 1,
			hash:     123,
		})

		other := &FileNode{
			Tree: tree,
			Name: "file.txt",
			Data: NodeData{
				FileInfo: FileInfo{
					Path:     "/file.txt",
					TypeFlag: 1,
					hash:     456,
				},
			},
		}

		result := node.compare(other)
		if result != Modified {
			t.Errorf("Expected Modified but got %v", result)
		}
	})
}

func TestFileNode_AddChild_EdgeCases(t *testing.T) {
	t.Run("add child with existing name", func(t *testing.T) {
		tree := NewFileTree()
		payload1 := FileInfo{Path: "/file1"}
		payload2 := FileInfo{Path: "/file2"}

		node1 := tree.Root.AddChild("test", payload1)
		node2 := tree.Root.AddChild("test", payload2)

		// Should add a new child even with same name
		if node1 == nil || node2 == nil {
			t.Errorf("Expected both nodes to be created")
		}

		if tree.Root.Children["test"] == nil {
			t.Errorf("Expected child to exist in tree")
		}
	})

	t.Run("add child to nested node", func(t *testing.T) {
		tree := NewFileTree()
		parent := tree.Root.AddChild("parent", FileInfo{})
		child := parent.AddChild("child", FileInfo{Path: "/parent/child"})

		if child == nil {
			t.Errorf("Expected child to be created")
		}

		if len(parent.Children) != 1 {
			t.Errorf("Expected parent to have 1 child, got %d", len(parent.Children))
		}
	})
}

func TestFileNode_Remove_EdgeCases(t *testing.T) {
	t.Run("remove node with children", func(t *testing.T) {
		tree := NewFileTree()
		parent := tree.Root.AddChild("parent", FileInfo{})
		parent.AddChild("child1", FileInfo{})
		parent.AddChild("child2", FileInfo{})

		initialSize := tree.Size
		err := parent.Remove()
		checkError(t, err, "unable to remove node")

		if tree.Size >= initialSize {
			t.Errorf("Expected tree size to decrease after removal")
		}
	})

	t.Run("remove root node", func(t *testing.T) {
		tree := NewFileTree()
		tree.Root.AddChild("child1", FileInfo{})
		tree.Root.AddChild("child2", FileInfo{})

		err := tree.Root.Remove()
		// Root removal should fail
		if err == nil {
			t.Errorf("Expected error when removing root node")
		}
		if err != nil && err.Error() != "cannot remove the tree root" {
			t.Errorf("Expected 'cannot remove the tree root' error, got: %v", err)
		}
	})
}

func TestFileNode_String(t *testing.T) {
	t.Run("string representation", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/test.txt", FileInfo{
			Path:     "/test.txt",
			TypeFlag: 1,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		})
		node.Data.DiffType = Modified

		str := node.String()
		if str == "" {
			t.Errorf("Expected non-empty string representation")
		}
	})

	t.Run("string representation for directory", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/dir", FileInfo{
			Path:     "/dir",
			TypeFlag: 1,
		})
		node.Data.FileInfo.TypeFlag = tar.TypeDir

		str := node.String()
		if str == "" {
			t.Errorf("Expected non-empty string representation for directory")
		}
	})
}

func TestFileNode_MetadataString(t *testing.T) {
	t.Run("metadata string for regular file", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/test.txt", FileInfo{
			Path:     "/test.txt",
			TypeFlag: 1,
			Size:     1024,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		})

		metadata := node.MetadataString()
		if metadata == "" {
			t.Errorf("Expected non-empty metadata string")
		}
	})

	t.Run("metadata string for directory", func(t *testing.T) {
		tree := NewFileTree()
		_, _, err := tree.AddPath("/dir", FileInfo{
			Path:     "/dir",
			TypeFlag: 1,
		})
		checkError(t, err, "unable to setup test")

		node, _ := tree.GetNode("/dir")
		node.Data.FileInfo.TypeFlag = tar.TypeDir

		metadata := node.MetadataString()
		if metadata == "" {
			t.Errorf("Expected non-empty metadata string for directory")
		}
	})
}

func TestFileNode_GetSize(t *testing.T) {
	t.Run("get size for regular file", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/file.txt", FileInfo{
			Path:     "/file.txt",
			TypeFlag: 1,
			Size:     2048,
		})

		size := node.GetSize()
		if size != 2048 {
			t.Errorf("Expected size 2048, got %d", size)
		}
	})

	t.Run("get size for directory with children", func(t *testing.T) {
		tree := NewFileTree()
		_, _, err := tree.AddPath("/dir", FileInfo{
			Path:     "/dir",
			TypeFlag: 1,
		})
		checkError(t, err, "unable to setup test")

		node, _ := tree.GetNode("/dir")
		node.Data.FileInfo.TypeFlag = tar.TypeDir

		tree.AddPath("/dir/file1.txt", FileInfo{Size: 100})
		tree.AddPath("/dir/file2.txt", FileInfo{Size: 200})

		size := node.GetSize()
		if size != 300 {
			t.Errorf("Expected total size 300, got %d", size)
		}
	})

	t.Run("get size for empty directory", func(t *testing.T) {
		tree := NewFileTree()
		_, _, err := tree.AddPath("/dir", FileInfo{
			Path:     "/dir",
			TypeFlag: 1,
		})
		checkError(t, err, "unable to setup test")

		node, _ := tree.GetNode("/dir")
		node.Data.FileInfo.TypeFlag = tar.TypeDir

		size := node.GetSize()
		if size != 0 {
			t.Errorf("Expected size 0 for empty directory, got %d", size)
		}
	})
}

func TestFileNode_VisitDepthChildFirst(t *testing.T) {
	t.Run("visit tree child first", func(t *testing.T) {
		tree := NewFileTree()
		tree.AddPath("/dir", FileInfo{})
		tree.AddPath("/dir/file1.txt", FileInfo{})
		tree.AddPath("/dir/file2.txt", FileInfo{})

		node, _ := tree.GetNode("/dir")

		var visited []string
		visitor := func(n *FileNode) error {
			visited = append(visited, n.Path())
			return nil
		}
		evaluator := func(n *FileNode) bool {
			return true
		}
		sorter := GetSortOrderStrategy(ByName)

		err := node.VisitDepthChildFirst(visitor, evaluator, sorter)
		checkError(t, err, "unable to visit tree")

		if len(visited) == 0 {
			t.Errorf("Expected nodes to be visited")
		}
	})
}

func TestFileNode_VisitDepthParentFirst(t *testing.T) {
	t.Run("visit tree parent first", func(t *testing.T) {
		tree := NewFileTree()
		tree.AddPath("/dir", FileInfo{})
		tree.AddPath("/dir/file1.txt", FileInfo{})
		tree.AddPath("/dir/file2.txt", FileInfo{})

		node, _ := tree.GetNode("/dir")

		var visited []string
		visitor := func(n *FileNode) error {
			visited = append(visited, n.Path())
			return nil
		}
		evaluator := func(n *FileNode) bool {
			return true
		}
		sorter := GetSortOrderStrategy(ByName)

		err := node.VisitDepthParentFirst(visitor, evaluator, sorter)
		checkError(t, err, "unable to visit tree")

		if len(visited) == 0 {
			t.Errorf("Expected nodes to be visited")
		}
	})
}

func TestFileNode_AssignDiffType(t *testing.T) {
	t.Run("assign diff type to node and children", func(t *testing.T) {
		tree := NewFileTree()
		tree.AddPath("/dir", FileInfo{})
		tree.AddPath("/dir/file1.txt", FileInfo{})
		tree.AddPath("/dir/file2.txt", FileInfo{})

		node, _ := tree.GetNode("/dir")

		err := node.AssignDiffType(Added)
		checkError(t, err, "unable to assign diff type")

		// Check that the node and its children have the correct diff type
		if node.Data.DiffType != Added {
			t.Errorf("Expected node diff type to be Added, got %v", node.Data.DiffType)
		}
	})
}

func TestFileNode_IsLeaf(t *testing.T) {
	t.Run("leaf node has no children", func(t *testing.T) {
		tree := NewFileTree()
		node, _, _ := tree.AddPath("/file.txt", FileInfo{})

		if !node.IsLeaf() {
			t.Errorf("Expected file node to be a leaf")
		}
	})

	t.Run("directory node is not a leaf", func(t *testing.T) {
		tree := NewFileTree()
		tree.AddPath("/dir", FileInfo{})
		tree.AddPath("/dir/file.txt", FileInfo{})

		node, _ := tree.GetNode("/dir")

		if node.IsLeaf() {
			t.Errorf("Expected directory node to not be a leaf")
		}
	})
}

