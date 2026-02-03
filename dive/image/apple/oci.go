//go:build darwin

package apple

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zstd"

	"github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
	"github.com/wagoodman/dive/internal/log"
)

// OCI image index structure
type ociIndex struct {
	SchemaVersion int           `json:"schemaVersion"`
	Manifests     []ociManifest `json:"manifests"`
}

type ociManifest struct {
	MediaType   string            `json:"mediaType"`
	Size        int64             `json:"size"`
	Digest      string            `json:"digest"`
	Platform    *ociPlatform      `json:"platform,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ociPlatform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

// OCI image manifest (single platform)
type ociImageManifest struct {
	SchemaVersion int              `json:"schemaVersion"`
	MediaType     string           `json:"mediaType"`
	Config        ociDescriptor    `json:"config"`
	Layers        []ociDescriptor  `json:"layers"`
}

type ociDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// OCI config structure
type ociConfig struct {
	Architecture string            `json:"architecture"`
	OS           string            `json:"os"`
	History      []ociHistoryEntry `json:"history,omitempty"`
	RootFS       ociRootFS         `json:"rootfs"`
}

type ociHistoryEntry struct {
	Created    string `json:"created,omitempty"`
	CreatedBy  string `json:"created_by,omitempty"`
	EmptyLayer bool   `json:"empty_layer,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

type ociRootFS struct {
	Type    string   `json:"type"`
	DiffIDs []string `json:"diff_ids"`
}

// parseOCIArchiveForPlatform parses an OCI archive and returns only layers for the current platform
func parseOCIArchiveForPlatform(tarFile io.ReadCloser) (*image.Image, string, error) {
	// Get current architecture
	currentArch := runtime.GOARCH
	// Map Go arch names to OCI arch names
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"arm":   "arm",
	}
	ociArch, ok := archMap[currentArch]
	if !ok {
		ociArch = currentArch
	}

	log.WithFields("arch", ociArch).Debug("parsing OCI archive for platform")

	bufferedReader := bufio.NewReaderSize(tarFile, 32*1024)
	tarReader := tar.NewReader(bufferedReader)

	// First pass: collect all JSON files and blobs info
	jsonFiles := make(map[string][]byte)
	blobData := make(map[string][]byte) // Store small blobs in memory for re-reading

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("error reading tar: %w", err)
		}

		name := header.Name

		if header.Typeflag == tar.TypeReg {
			if name == "index.json" || name == "oci-layout" {
				data, err := io.ReadAll(tarReader)
				if err != nil {
					return nil, "", err
				}
				jsonFiles[name] = data
			} else if strings.HasPrefix(name, "blobs/") {
				// Read blob data
				data, err := io.ReadAll(tarReader)
				if err != nil {
					return nil, "", err
				}
				// Extract digest from path (blobs/sha256/abc123 -> sha256:abc123)
				parts := strings.Split(name, "/")
				if len(parts) >= 3 {
					digest := parts[1] + ":" + parts[2]
					blobData[digest] = data
				}
			}
		}
	}

	// Parse index.json
	indexData, ok := jsonFiles["index.json"]
	if !ok {
		return nil, "", fmt.Errorf("index.json not found in OCI archive")
	}

	var index ociIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, "", fmt.Errorf("failed to parse index.json: %w", err)
	}

	// Find the image index manifest (contains platform-specific manifests)
	var imageIndexDigest string
	for _, m := range index.Manifests {
		if m.MediaType == "application/vnd.oci.image.index.v1+json" {
			imageIndexDigest = m.Digest
			break
		}
	}

	if imageIndexDigest == "" {
		return nil, "", fmt.Errorf("no image index found in OCI archive")
	}

	// Parse image index to find platform-specific manifest
	imageIndexData, ok := blobData[imageIndexDigest]
	if !ok {
		return nil, "", fmt.Errorf("image index blob not found: %s", imageIndexDigest)
	}

	var imageIndex ociIndex
	if err := json.Unmarshal(imageIndexData, &imageIndex); err != nil {
		return nil, "", fmt.Errorf("failed to parse image index: %w", err)
	}

	// Find manifest for current architecture
	var platformManifestDigest string
	for _, m := range imageIndex.Manifests {
		if m.Platform != nil && m.Platform.Architecture == ociArch && m.Platform.OS == "linux" {
			// Skip attestation manifests
			if m.Annotations != nil {
				if _, isAttestation := m.Annotations["vnd.docker.reference.type"]; isAttestation {
					continue
				}
			}
			platformManifestDigest = m.Digest
			log.WithFields("arch", ociArch, "digest", platformManifestDigest).Debug("found platform manifest")
			break
		}
	}

	if platformManifestDigest == "" {
		return nil, "", fmt.Errorf("no manifest found for architecture %s", ociArch)
	}

	// Parse platform manifest
	platformManifestData, ok := blobData[platformManifestDigest]
	if !ok {
		return nil, "", fmt.Errorf("platform manifest blob not found: %s", platformManifestDigest)
	}

	var platformManifest ociImageManifest
	if err := json.Unmarshal(platformManifestData, &platformManifest); err != nil {
		return nil, "", fmt.Errorf("failed to parse platform manifest: %w", err)
	}

	// Parse config
	configData, ok := blobData[platformManifest.Config.Digest]
	if !ok {
		return nil, "", fmt.Errorf("config blob not found: %s", platformManifest.Config.Digest)
	}

	var config ociConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, "", fmt.Errorf("failed to parse config: %w", err)
	}

	// Process only the layers referenced by this platform's manifest
	trees := make([]*filetree.FileTree, 0, len(platformManifest.Layers))
	layers := make([]*image.Layer, 0, len(platformManifest.Layers))

	histIdx := 0
	for idx, layerDesc := range platformManifest.Layers {
		layerData, ok := blobData[layerDesc.Digest]
		if !ok {
			return nil, "", fmt.Errorf("layer blob not found: %s", layerDesc.Digest)
		}

		tree, err := processLayerBlob(layerDesc.Digest, layerData)
		if err != nil {
			return nil, "", fmt.Errorf("failed to process layer %s: %w", layerDesc.Digest, err)
		}

		trees = append(trees, tree)

		// Find matching history entry (skip empty layers)
		command := "(missing)"
		for ; histIdx < len(config.History); histIdx++ {
			if !config.History[histIdx].EmptyLayer {
				command = config.History[histIdx].CreatedBy
				histIdx++
				break
			}
		}

		layers = append(layers, &image.Layer{
			Id:      layerDesc.Digest,
			Index:   idx,
			Command: command,
			Size:    tree.FileSize,
			Tree:    tree,
		})
	}

	return &image.Image{
		Trees:  trees,
		Layers: layers,
	}, ociArch, nil
}

func processLayerBlob(name string, data []byte) (*filetree.FileTree, error) {
	reader := bytes.NewReader(data)

	// Try gzip
	if gzReader, err := gzip.NewReader(reader); err == nil {
		tree, err := processLayerTar(name, tar.NewReader(gzReader))
		if err == nil {
			return tree, nil
		}
	}

	// Try zstd
	reader.Reset(data)
	if zstdReader, err := zstd.NewReader(reader); err == nil {
		tree, err := processLayerTar(name, tar.NewReader(zstdReader))
		if err == nil {
			return tree, nil
		}
	}

	// Try plain tar
	reader.Reset(data)
	return processLayerTar(name, tar.NewReader(reader))
}

func processLayerTar(name string, reader *tar.Reader) (*filetree.FileTree, error) {
	tree := filetree.NewFileTree()
	tree.Name = name

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		headerName := header.Name
		if headerName == "." || headerName == "" {
			continue
		}

		// Clean path if needed
		if strings.Contains(headerName, "..") || strings.Contains(headerName, "//") {
			headerName = path.Clean(headerName)
			if headerName == "." {
				continue
			}
		}

		switch header.Typeflag {
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		default:
			info := filetree.NewFileInfoFromTarHeader(reader, header, headerName)
			tree.FileSize += uint64(info.Size)
			if _, _, err := tree.AddPath(info.Path, info); err != nil {
				return nil, err
			}
		}
	}

	return tree, nil
}
