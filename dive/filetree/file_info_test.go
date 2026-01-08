package filetree

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileInfoFromTarHeader(t *testing.T) {
	t.Run("regular file", func(t *testing.T) {
		// Create a tar header for a regular file
		header := &tar.Header{
			Name:     "test.txt",
			Typeflag: tar.TypeReg,
			Size:     1024,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
			Linkname: "",
		}

		// Create a reader with some content
		content := []byte("hello world")
		reader := tar.NewReader(bytes.NewReader(content))

		// Advance reader to setup (normally done by tar.Next())
		// We need to manually set up the reader state
		result := NewFileInfoFromTarHeader(reader, header, "test.txt")

		assert.Equal(t, "test.txt", result.Path)
		assert.Equal(t, byte(tar.TypeReg), result.TypeFlag)
		assert.Equal(t, int64(1024), result.Size)
		assert.Equal(t, 1000, result.Uid)
		assert.Equal(t, 1000, result.Gid)
		assert.False(t, result.IsDir)
		assert.NotEqual(t, uint64(0), result.hash) // hash should be computed
		// Don't check Mode as it can be platform-dependent
	})

	t.Run("directory", func(t *testing.T) {
		header := &tar.Header{
			Name:     "testdir",
			Typeflag: tar.TypeDir,
			Size:     0,
			Mode:     0755,
			Uid:      1000,
			Gid:      1000,
		}

		reader := tar.NewReader(bytes.NewReader([]byte{}))
		result := NewFileInfoFromTarHeader(reader, header, "testdir")

		assert.Equal(t, "testdir", result.Path)
		assert.Equal(t, byte(tar.TypeDir), result.TypeFlag)
		assert.Equal(t, int64(0), result.Size)
		assert.True(t, result.IsDir)
		assert.Equal(t, uint64(0), result.hash) // directories have no hash
	})

	t.Run("symlink", func(t *testing.T) {
		header := &tar.Header{
			Name:     "link.txt",
			Typeflag: tar.TypeSymlink,
			Size:     0,
			Mode:     0777,
			Linkname: "target.txt",
		}

		reader := tar.NewReader(bytes.NewReader([]byte{}))
		result := NewFileInfoFromTarHeader(reader, header, "link.txt")

		assert.Equal(t, "link.txt", result.Path)
		assert.Equal(t, byte(tar.TypeSymlink), result.TypeFlag)
		assert.Equal(t, "target.txt", result.Linkname)
		assert.False(t, result.IsDir)
		// Note: current implementation computes hash for symlinks (it should only skip dirs)
		// The hash will be the xxhash of empty content since reader is empty
		assert.NotEqual(t, uint64(0), result.hash)
	})

	t.Run("custom path", func(t *testing.T) {
		header := &tar.Header{
			Name:     "original/name.txt",
			Typeflag: tar.TypeReg,
			Size:     512,
			Mode:     0600,
		}

		content := []byte("test content")
		reader := tar.NewReader(bytes.NewReader(content))

		result := NewFileInfoFromTarHeader(reader, header, "custom/path.txt")

		assert.Equal(t, "custom/path.txt", result.Path)
		assert.Equal(t, int64(512), result.Size)
	})
}

func TestFileInfo_Copy(t *testing.T) {
	t.Run("copy file info", func(t *testing.T) {
		original := FileInfo{
			Path:     "/test/file.txt",
			TypeFlag: byte(tar.TypeReg),
			Linkname: "",
			hash:     12345,
			Size:     1024,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
			IsDir:    false,
		}

		copied := original.Copy()

		assert.NotNil(t, copied)
		assert.Equal(t, original.Path, copied.Path)
		assert.Equal(t, original.TypeFlag, copied.TypeFlag)
		assert.Equal(t, original.Linkname, copied.Linkname)
		assert.Equal(t, original.hash, copied.hash)
		assert.Equal(t, original.Size, copied.Size)
		assert.Equal(t, original.Mode, copied.Mode)
		assert.Equal(t, original.Uid, copied.Uid)
		assert.Equal(t, original.Gid, copied.Gid)
		assert.Equal(t, original.IsDir, copied.IsDir)

		// Verify it's a different instance
		assert.NotSame(t, &original, copied)
	})

	t.Run("copy nil file info", func(t *testing.T) {
		var original *FileInfo
		copied := original.Copy()

		assert.Nil(t, copied)
	})

	t.Run("copy directory info", func(t *testing.T) {
		original := FileInfo{
			Path:     "/test/dir",
			TypeFlag: byte(tar.TypeDir),
			IsDir:    true,
			Size:     0,
		}

		copied := original.Copy()

		assert.NotNil(t, copied)
		assert.Equal(t, original.Path, copied.Path)
		assert.True(t, copied.IsDir)
	})

	t.Run("modifying copy doesn't affect original", func(t *testing.T) {
		original := FileInfo{
			Path:  "/test/file.txt",
			Size:  1024,
			hash:  12345,
		}

		copied := original.Copy()
		copied.Size = 2048
		copied.hash = 54321

		assert.Equal(t, int64(1024), original.Size)
		assert.Equal(t, uint64(12345), original.hash)
		assert.Equal(t, int64(2048), copied.Size)
		assert.Equal(t, uint64(54321), copied.hash)
	})
}

func TestFileInfo_Compare(t *testing.T) {
	t.Run("identical files", func(t *testing.T) {
		info1 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		info2 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		result := info1.Compare(info2)
		assert.Equal(t, Unmodified, result)
	})

	t.Run("different type flag", func(t *testing.T) {
		info1 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		info2 := FileInfo{
			TypeFlag: byte(tar.TypeDir),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		result := info1.Compare(info2)
		assert.Equal(t, Modified, result)
	})

	t.Run("different hash", func(t *testing.T) {
		info1 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		info2 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     54321,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		result := info1.Compare(info2)
		assert.Equal(t, Modified, result)
	})

	t.Run("different mode", func(t *testing.T) {
		info1 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		info2 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0755,
			Uid:      1000,
			Gid:      1000,
		}

		result := info1.Compare(info2)
		assert.Equal(t, Modified, result)
	})

	t.Run("different uid", func(t *testing.T) {
		info1 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      1000,
			Gid:      1000,
		}

		info2 := FileInfo{
			TypeFlag: byte(tar.TypeReg),
			hash:     12345,
			Mode:     0644,
			Uid:      2000,
			Gid:      1000,
		}

		result := info1.Compare(info2)
		assert.Equal(t, Modified, result)
	})
}

func TestGetHashFromReader(t *testing.T) {
	t.Run("hash of empty reader", func(t *testing.T) {
		reader := bytes.NewReader([]byte{})
		hash := getHashFromReader(reader)

		assert.Equal(t, uint64(17241709254077376921), hash) // xxhash of empty string
	})

	t.Run("hash of simple string", func(t *testing.T) {
		reader := bytes.NewReader([]byte("hello world"))
		hash := getHashFromReader(reader)

		assert.NotEqual(t, uint64(0), hash)
		// Verify consistency
		hash2 := getHashFromReader(bytes.NewReader([]byte("hello world")))
		assert.Equal(t, hash, hash2)
	})

	t.Run("hash of binary data", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
		reader := bytes.NewReader(data)
		hash := getHashFromReader(reader)

		assert.NotEqual(t, uint64(0), hash)
	})

	t.Run("hash of large data", func(t *testing.T) {
		// Create data larger than buffer size (1024 bytes)
		largeData := make([]byte, 2048)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		reader := bytes.NewReader(largeData)
		hash := getHashFromReader(reader)

		assert.NotEqual(t, uint64(0), hash)
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		hash1 := getHashFromReader(bytes.NewReader([]byte("content1")))
		hash2 := getHashFromReader(bytes.NewReader([]byte("content2")))

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("same content produces same hash", func(t *testing.T) {
		content := []byte("same content")
		hash1 := getHashFromReader(bytes.NewReader(content))
		hash2 := getHashFromReader(bytes.NewReader(content))

		assert.Equal(t, hash1, hash2)
	})
}

func TestNewFileInfo(t *testing.T) {
	t.Run("create regular file info", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)

		info, err := os.Stat(filePath)
		require.NoError(t, err)

		fileInfo := NewFileInfo(filePath, "test.txt", info)

		assert.Equal(t, "test.txt", fileInfo.Path)
		assert.Equal(t, byte(tar.TypeReg), fileInfo.TypeFlag)
		assert.Equal(t, int64(12), fileInfo.Size) // "test content" is 12 bytes
		assert.False(t, fileInfo.IsDir)
		assert.Equal(t, -1, fileInfo.Uid) // UID/GID not supported, set to -1
		assert.Equal(t, -1, fileInfo.Gid)
		assert.NotEqual(t, uint64(0), fileInfo.hash) // hash should be computed
		// Mode may have additional bits set on different systems, just check it's not zero
		assert.NotEqual(t, os.FileMode(0), fileInfo.Mode)
	})

	t.Run("create directory info", func(t *testing.T) {
		tmpDir := t.TempDir()
		dirPath := filepath.Join(tmpDir, "testdir")
		err := os.Mkdir(dirPath, 0755)
		require.NoError(t, err)

		info, err := os.Stat(dirPath)
		require.NoError(t, err)

		fileInfo := NewFileInfo(dirPath, "testdir", info)

		assert.Equal(t, "testdir", fileInfo.Path)
		assert.Equal(t, byte(tar.TypeDir), fileInfo.TypeFlag)
		assert.True(t, fileInfo.IsDir)
		assert.Equal(t, uint64(0), fileInfo.hash) // directories have no hash
		// Check that directory mode has dir bit set
		assert.True(t, fileInfo.Mode&os.ModeDir != 0)
	})

	t.Run("create symlink info", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "target.txt")
		err := os.WriteFile(targetPath, []byte("target"), 0644)
		require.NoError(t, err)

		linkPath := filepath.Join(tmpDir, "link.txt")
		err = os.Symlink("target.txt", linkPath)
		require.NoError(t, err)

		info, err := os.Lstat(linkPath)
		require.NoError(t, err)

		fileInfo := NewFileInfo(linkPath, "link.txt", info)

		assert.Equal(t, "link.txt", fileInfo.Path)
		assert.Equal(t, byte(tar.TypeSymlink), fileInfo.TypeFlag)
		assert.Equal(t, "target.txt", fileInfo.Linkname)
		assert.False(t, fileInfo.IsDir)
		// Note: current implementation computes hash for symlinks (from the target file content)
		assert.NotEqual(t, uint64(0), fileInfo.hash)
	})
}
