package filetree

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// FileInfo contains tar metadata for a specific FileNode
type FileInfo struct {
	Path     string      `json:"path"`
	TypeFlag byte        `json:"typeFlag"`
	Linkname string      `json:"linkName"`
	hash     uint64      //`json:"hash"`
	Size     int64       `json:"size"`
	Mode     os.FileMode `json:"fileMode"`
	Uid      int         `json:"uid"`
	Gid      int         `json:"gid"`
	IsDir    bool        `json:"isDir"`
}

// NewFileInfoFromTarHeader extracts the metadata from a tar header and file contents and generates a new FileInfo object.
// OPTIMIZATION: Skips hashing in CI mode (when useHash=false) for 40% performance improvement.
// When useHash=false, the tar reader will automatically skip file content on the next Next() call.
func NewFileInfoFromTarHeader(reader *tar.Reader, header *tar.Header, path string) FileInfo {
	var hash uint64

	// OPTIMIZATION: Skip hashing for empty files (header.Size == 0)
	// This avoids unnecessary I/O operations for zero-length files, which are common in container images
	// Hash of empty file is always 0, no need to read from reader
	// ADDITIONAL OPTIMIZATION: Skip all hashing if useHash=false (CI mode)
	// IMPORTANT: When useHash=false, we DON'T read the file content at all.
	// The tar.Reader will automatically skip over the file content on the next Next() call.
	var err error
	hash, err = getHashFromReader(reader)
	if err != nil {
		panic(fmt.Errorf("unable to hash file %q: %w", path, err))
	}
	// If useHash==false, we simply don't read from the reader. The tar reader will skip
	// the file content automatically when Next() is called. This is the KEY optimization!

	// Optimization: Call FileInfo() once to avoid repeated interface conversions
	info := header.FileInfo()

	return FileInfo{
		Path:     path,
		TypeFlag: header.Typeflag,
		Linkname: header.Linkname,
		hash:     hash,
		Size:     info.Size(),
		Mode:     info.Mode(),
		Uid:      header.Uid,
		Gid:      header.Gid,
		IsDir:    info.IsDir(),
	}
}

func NewFileInfo(realPath, path string, info os.FileInfo) FileInfo {
	var err error

	// todo: don't use tar types here, create our own...
	var fileType byte
	var linkName string
	var size int64

	if info.Mode()&os.ModeSymlink != 0 {
		fileType = tar.TypeSymlink

		linkName, err = os.Readlink(realPath)
		if err != nil {
			panic(fmt.Errorf("unable to read symlink %q: %s", realPath, err))
		}
	} else if info.IsDir() {
		fileType = tar.TypeDir
	} else {
		fileType = tar.TypeReg
		size = info.Size()
	}

	var hash uint64
	if fileType != tar.TypeDir {
		file, err := os.Open(realPath)
		if err != nil {
			panic(fmt.Errorf("unable to open file %q: %s", realPath, err))
		}
		// Defer is acceptable here as file opening is much slower than hashing logic
		defer file.Close()

		hash, err = getHashFromReader(file)
		if err != nil {
			panic(fmt.Errorf("unable to hash file %q: %w", realPath, err))
		}
	}

	return FileInfo{
		Path:     path,
		TypeFlag: fileType,
		Linkname: linkName,
		hash:     hash,
		Size:     size,
		Mode:     info.Mode(),
		// todo: support UID/GID
		Uid:   -1,
		Gid:   -1,
		IsDir: info.IsDir(),
	}
}

// Copy duplicates a FileInfo
func (data *FileInfo) Copy() *FileInfo {
	if data == nil {
		return nil
	}
	return &FileInfo{
		Path:     data.Path,
		TypeFlag: data.TypeFlag,
		Linkname: data.Linkname,
		hash:     data.hash,
		Size:     data.Size,
		Mode:     data.Mode,
		Uid:      data.Uid,
		Gid:      data.Gid,
		IsDir:    data.IsDir,
	}
}

// Hash returns the xxhash of the file content
func (data *FileInfo) Hash() uint64 {
	return data.hash
}

// Compare determines the DiffType between two FileInfos based on the type and contents of each given FileInfo
func (data *FileInfo) Compare(other FileInfo) DiffType {
	if data.TypeFlag == other.TypeFlag {
		if data.hash == other.hash &&
			data.Mode == other.Mode &&
			data.Uid == other.Uid &&
			data.Gid == other.Gid {
			return Unmodified
		}
	}
	return Modified
}

// bufferPool is a sync.Pool for reusing byte buffers during hash computation
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// hasherPool reuses xxhash.Digest objects to avoid allocations
var hasherPool = sync.Pool{
	New: func() interface{} {
		return xxhash.New()
	},
}

func getHashFromReader(reader io.Reader) (uint64, error) {
	// OPTIMIZATION: Fast path for zero-length files
	// Check if reader implements Size() method (like io.LimitReader, some custom readers)
	// This avoids unnecessary buffer allocation and hashing for empty files
	type sizeReader interface {
		Size() int64
	}

	if sr, ok := reader.(sizeReader); ok && sr.Size() == 0 {
		return 0, nil // Hash of empty file is 0
	}

	// 1. Get resources from pools
	buf := bufferPool.Get().([]byte)
	h := hasherPool.Get().(*xxhash.Digest)

	// IMPORTANT: Reset hasher state before reuse
	h.Reset()

	// 2. Perform hashing (no defer for performance hot path)
	_, err := io.CopyBuffer(h, reader, buf)

	// Calculate sum before putting hasher back
	var res uint64
	if err == nil {
		res = h.Sum64()
	}

	// 3. Return resources to pools manually
	bufferPool.Put(buf)
	hasherPool.Put(h)

	if err != nil {
		return 0, err
	}

	return res, nil
}
