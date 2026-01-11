# Performance Improvements - Dive

This document tracks performance bottlenecks and optimization opportunities identified through profiling with GitLab CE image (109,802 files).

## Profiling Results Summary

**Image:** gitlab/gitlab-ce:latest (109,802 files, 3.8 GB)
**Profiling Date:** 2026-01-08

### Before vs After Comparison

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| CPU Total | 3.79s (9.50%) | 3.26s (7.63%) | ✅ 14% faster |
| getHashFromReader | 1.91s (50.40%) | 1.54s (47.24%) | ✅ 19% faster |
| Memory Total | 62.34 MB | 53.14 MB | ✅ 15% less |
| NewNode | 26.50 MB (42.52%) | 21 MB (39.52%) | ✅ 21% less |

---

## 🎯 Current Performance Issues (by Priority)

### 🔴 CRITICAL - Memory Hogs

#### 1. NewNode - 21 MB (39.52% of total memory)

**File:** [`dive/filetree/file_node.go:37-51`](dive/filetree/file_node.go#L37-L51)

```go
func NewNode(parent *FileNode, name string, data FileInfo) (node *FileNode) {
    node = new(FileNode)
    node.Name = name
    node.Data = *NewNodeData()
    node.Data.FileInfo = *data.Copy()
    node.Size = -1 // signal lazy load later

    node.Children = make(map[string]*FileNode)  // ❌ PROBLEM: Creates map for ALL nodes
    node.Parent = parent
    if parent != nil {
        node.Tree = parent.Tree
    }

    return node
}
```

**Problem:** Creates a `map[string]*FileNode` for EVERY node, including leaf files that never have children.

**Impact:** For 109,802 files, this creates ~109K empty maps, wasting ~21 MB of memory.

**Solution:** Use lazy initialization - only create the map when actually adding a child.

```go
// Proposed fix:
// In NewNode(): Remove node.Children = make(...)
// In AddChild(): Add lazy initialization
func (node *FileNode) AddChild(name string, data FileInfo) {
    if node.Children == nil {
        node.Children = make(map[string]*FileNode)
    }
    // ... rest of code
}
```

**Expected Improvement:** Save ~15-20 MB (30-40% reduction)

---

#### 2. AddChild - 8.55 MB (16.09% of total memory)

**File:** [`dive/filetree/file_node.go:90-106`](dive/filetree/file_node.go#L90-L106)

```go
func (node *FileNode) AddChild(name string, data FileInfo) (child *FileNode) {
    // never allow processing of purely whiteout flag files (for now)
    if strings.HasPrefix(name, doubleWhiteoutPrefix) {
        return nil
    }

    child = NewNode(node, name, data)  // ❌ Calls NewNode which creates empty map
    if node.Children[name] != nil {
        // tree node already exists, replace the payload, keep the children
        node.Children[name].Data.FileInfo = *data.Copy()  // ❌ Unnecessary copy
    } else {
        node.Children[name] = child
        node.Tree.Size++
    }

    return child
}
```

**Problems:**
1. Calls `NewNode()` which creates empty maps for all children
2. Does unnecessary `data.Copy()` when replacing existing node

**Solution:** Combine with NewNode fix above.

---

### 🟡 MEDIUM - Performance Bottlenecks

#### 3. Path() - 5 MB (9.41% of total memory)

**File:** [`dive/filetree/file_node.go:275-296`](dive/filetree/file_node.go#L275-L296)

```go
func (node *FileNode) Path() string {
    if node.path == "" {
        var path []string
        curNode := node
        for {
            if curNode.Parent == nil {
                break
            }

            name := curNode.Name
            if curNode == node {
                // white out prefixes are fictitious on leaf nodes
                name = strings.TrimPrefix(name, whiteoutPrefix)
            }

            path = append([]string{name}, path...)  // ❌ PROBLEM: O(n²) prepend
            curNode = curNode.Parent
        }
        node.path = "/" + strings.Join(path, "/")
    }
    return strings.Replace(node.path, "//", "/", -1)
}
```

**Problem:** Using `append([]string{name}, path...)` creates a new slice on every iteration, resulting in O(n²) time complexity.

**Solution 1:** Build in reverse, then reverse at the end.

```go
func (node *FileNode) Path() string {
    if node.path == "" {
        var path []string
        curNode := node
        for curNode.Parent != nil {
            name := curNode.Name
            if curNode == node {
                name = strings.TrimPrefix(name, whiteoutPrefix)
            }
            path = append(path, name)  // Append in reverse order
            curNode = curNode.Parent
        }

        // Reverse the slice
        for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
            path[i], path[j] = path[j], path[i]
        }

        node.path = "/" + strings.Join(path, "/")
    }
    return strings.Replace(node.path, "//", "/", -1)
}
```

**Solution 2:** Use strings.Builder with better cache locality.

**Expected Improvement:** Save ~3-4 MB, improve Path() performance significantly

---

#### 4. Efficiency.func1 - 5.51 MB (10.37% of total memory)

**File:** [`dive/filetree/efficiency.go`](dive/filetree/efficiency.go) (needs investigation)

**Status:** Need to review code to identify the specific issue

---

### 🟢 LOW - Minor Optimizations

#### 5. NewNodeData Allocation

**File:** [`dive/filetree/node_data.go:13-19`](dive/filetree/node_data.go#L13-L19)

```go
func NewNodeData() *NodeData {
    return &NodeData{
        ViewInfo: *NewViewInfo(),  // Allocates ViewInfo for all nodes
        FileInfo: FileInfo{},
        DiffType: Unmodified,
    }
}
```

**Problem:** Allocates ViewInfo for every node, even if not used.

---

## ✅ Completed Optimizations

### getHashFromReader - 19% CPU improvement

**File:** [`dive/filetree/file_info.go:163-200`](dive/filetree/file_info.go#L163-L200)

**Changes Made:**
1. ✅ Increased buffer from 1KB to 32KB
2. ✅ Added sync.Pool for buffer reuse
3. ✅ Added sync.Pool for hasher reuse
4. ✅ Replaced manual loop with `io.CopyBuffer`
5. ✅ Added fast path for empty files
6. ✅ Removed defer from hot path

**Before:**
```go
func getHashFromReader(reader io.Reader) uint64 {
    h := xxhash.New()
    buf := make([]byte, 1024)  // ❌ Small buffer, new allocation every time
    for {
        n, err := reader.Read(buf)
        if err != nil && err != io.EOF {
            panic(fmt.Errorf("unable to read file: %w", err))
        }
        if n == 0 {
            break
        }
        _, err = h.Write(buf[:n])
        if err != nil {
            panic(fmt.Errorf("unable to write to hash: %w", err))
        }
    }
    return h.Sum64()
}
```

**After:**
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 32*1024)  // ✅ 32x larger buffer
    },
}

var hasherPool = sync.Pool{
    New: func() interface{} {
        return xxhash.New()  // ✅ Reuse hashers
    },
}

func getHashFromReader(reader io.Reader) (uint64, error) {
    // Fast path for zero-length files
    if sr, ok := reader.(interface{ Size() int64 }); ok && sr.Size() == 0 {
        return 0, nil
    }

    buf := bufferPool.Get().([]byte)
    h := hasherPool.Get().(*xxhash.Digest)
    h.Reset()

    _, err := io.CopyBuffer(h, reader, buf)  // ✅ More efficient

    var res uint64
    if err == nil {
        res = h.Sum64()
    }

    bufferPool.Put(buf)
    hasherPool.Put(h)

    if err != nil {
        return 0, err
    }
    return res, nil
}
```

**Results:**
- CPU: 19% faster on large images
- Memory: 15% reduction
- Allocations: Significantly reduced due to sync.Pool

---

## 📊 Profiling Commands

```bash
# Run profiling
./profile.sh gitlab/gitlab-ce:latest

# View CPU profile
go tool pprof -top profiling/cpu.pprof

# View memory profile
go tool pprof -top profiling/mem.pprof

# Interactive CPU profiling
go tool pprof profiling/cpu.pprof
# (pprof) top
# (pprof) list getHashFromReader
# (pprof) web

# Generate flame graph
go tool pprof -http=:8080 profiling/cpu.pprof
```

---

## 🎯 Next Steps Priority Order

1. **[HIGH]** Implement lazy initialization for `Children` map in `NewNode` - Expected: ~15-20 MB savings
2. **[MEDIUM]** Optimize `Path()` function - Expected: ~3-4 MB savings
3. **[LOW]** Investigate `Efficiency.func1` - Needs code review
4. **[LOW]** Optimize `NewNodeData` ViewInfo allocation

---

## 📝 Notes

- All profiling done with `-gcflags="all=-l"` to disable inlining for accurate profiling
- Large images (100K+ files) show realistic bottlenecks
- Small images (Alpine) may not reveal all performance issues
- Memory profiling shows in-use space at peak allocation

---

## 🔗 Related Files

- Main filetree implementation: [`dive/filetree/file_tree.go`](dive/filetree/file_tree.go)
- Node structure: [`dive/filetree/file_node.go`](dive/filetree/file_node.go)
- File info and hashing: [`dive/filetree/file_info.go`](dive/filetree/file_info.go)
- Efficiency calculation: [`dive/filetree/efficiency.go`](dive/filetree/efficiency.go)
- Profiling script: [`profile.sh`](profile.sh)
