package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
)

func TestCalculateImageStats(t *testing.T) {
	tests := []struct {
		name     string
		analysis *image.Analysis
		want     ImageStats
	}{
		{
			name:     "nil analysis",
			analysis: nil,
			want:     ImageStats{},
		},
		{
			name: "empty analysis",
			analysis: &image.Analysis{
				Image:          "test-image",
				SizeBytes:      1000,
				WastedBytes:    100,
				Efficiency:     0.9,
				Inefficiencies: filetree.EfficiencySlice{},
			},
			want: ImageStats{
				ImageName:         "test-image",
				TotalSizeBytes:    1000,
				WastedBytes:       100,
				EfficiencyScore:   90.0,
				FilesAboveZeroKB:  0,
				InefficiencyCount: 0,
			},
		},
		{
			name: "analysis with inefficiencies",
			analysis: &image.Analysis{
				Image:       "test-image",
				SizeBytes:   5000,
				WastedBytes: 2000,
				Efficiency:  0.6,
				Inefficiencies: filetree.EfficiencySlice{
					{
						Path:           "/path/to/file1",
						CumulativeSize: 100,
						Nodes: []*filetree.FileNode{
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 50}}},
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 0}}}, // Should not be counted
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 100}}},
						},
					},
					{
						Path:           "/path/to/file2",
						CumulativeSize: 200,
						Nodes: []*filetree.FileNode{
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 200}}},
						},
					},
					{
						Path:           "/path/to/empty",
						CumulativeSize: 0, // Should not be counted as inefficiency
						Nodes: []*filetree.FileNode{
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 0}}},
						},
					},
				},
			},
			want: ImageStats{
				ImageName:         "test-image",
				TotalSizeBytes:    5000,
				WastedBytes:       2000,
				EfficiencyScore:   60.0,
				FilesAboveZeroKB:  3, // 50, 100, 200 are > 0
				InefficiencyCount: 2, // Only 2 inefficiencies have CumulativeSize > 0
			},
		},
		{
			name: "efficiency calculation",
			analysis: &image.Analysis{
				Image:       "efficiency-test",
				SizeBytes:   10000,
				WastedBytes: 1000,
				Efficiency:  0.9,
				Inefficiencies: filetree.EfficiencySlice{
					{
						Path:           "/file",
						CumulativeSize: 500,
						Nodes: []*filetree.FileNode{
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 500}}},
						},
					},
				},
			},
			want: ImageStats{
				ImageName:         "efficiency-test",
				TotalSizeBytes:    10000,
				WastedBytes:       1000,
				EfficiencyScore:   90.0, // 0.9 * 100
				FilesAboveZeroKB:  1,
				InefficiencyCount: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateImageStats(tt.analysis)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountFilesAboveZeroBytes(t *testing.T) {
	tests := []struct {
		name     string
		analysis *image.Analysis
		want     int
	}{
		{
			name:     "nil analysis",
			analysis: nil,
			want:     0,
		},
		{
			name: "no inefficiencies",
			analysis: &image.Analysis{
				Inefficiencies: filetree.EfficiencySlice{},
			},
			want: 0,
		},
		{
			name: "mixed file sizes",
			analysis: &image.Analysis{
				Inefficiencies: filetree.EfficiencySlice{
					{
						Nodes: []*filetree.FileNode{
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 100}}},
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 0}}},
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 50}}},
						},
					},
				},
			},
			want: 2, // Only 100 and 50 are > 0
		},
		{
			name: "all zero size files",
			analysis: &image.Analysis{
				Inefficiencies: filetree.EfficiencySlice{
					{
						Nodes: []*filetree.FileNode{
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 0}}},
							{Data: filetree.NodeData{FileInfo: filetree.FileInfo{Size: 0}}},
						},
					},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countFilesAboveZeroBytes(tt.analysis)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountInefficiencies(t *testing.T) {
	tests := []struct {
		name     string
		analysis *image.Analysis
		want     int
	}{
		{
			name:     "nil analysis",
			analysis: nil,
			want:     0,
		},
		{
			name: "no inefficiencies",
			analysis: &image.Analysis{
				Inefficiencies: filetree.EfficiencySlice{},
			},
			want: 0,
		},
		{
			name: "mixed cumulative sizes",
			analysis: &image.Analysis{
				Inefficiencies: filetree.EfficiencySlice{
					{CumulativeSize: 100},
					{CumulativeSize: 0},
					{CumulativeSize: 50},
				},
			},
			want: 2, // Only 2 have CumulativeSize > 0
		},
		{
			name: "all zero cumulative size",
			analysis: &image.Analysis{
				Inefficiencies: filetree.EfficiencySlice{
					{CumulativeSize: 0},
					{CumulativeSize: 0},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countInefficiencies(tt.analysis)
			assert.Equal(t, tt.want, got)
		})
	}
}
