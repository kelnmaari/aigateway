package inference

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestTRTConverter_MetadataSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	conv, err := NewTRTConverter(TRTConverterConfig{
		EnginesDir: tmpDir,
		HFCacheDir: tmpDir,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("NewTRTConverter failed: %v", err)
	}

	// Create engine dir
	engineDir := filepath.Join(tmpDir, "test-engine")
	if err := os.MkdirAll(engineDir, 0755); err != nil {
		t.Fatalf("create engine dir: %v", err)
	}

	meta := &TRTEngineMetadata{
		ModelID:       "meta-llama/Llama-3.1-8B",
		CUDAVersion:   "12.4",
		TRTVersion:    "10.0.1",
		DriverVersion: "550.54.14",
		GPUSMVersion:  "89",
		Dtype:         "float16",
		MaxBatchSize:  8,
		MaxSeqLen:     4096,
		CreatedAt:     time.Now(),
	}

	metaPath := conv.MetadataPath("test-engine")
	if err := conv.saveMetadata(metaPath, meta); err != nil {
		t.Fatalf("saveMetadata failed: %v", err)
	}

	// Load and verify
	loaded, err := conv.loadMetadata(metaPath)
	if err != nil {
		t.Fatalf("loadMetadata failed: %v", err)
	}

	if loaded.ModelID != meta.ModelID {
		t.Errorf("ModelID = %q, want %q", loaded.ModelID, meta.ModelID)
	}
	if loaded.CUDAVersion != meta.CUDAVersion {
		t.Errorf("CUDAVersion = %q, want %q", loaded.CUDAVersion, meta.CUDAVersion)
	}
	if loaded.GPUSMVersion != meta.GPUSMVersion {
		t.Errorf("GPUSMVersion = %q, want %q", loaded.GPUSMVersion, meta.GPUSMVersion)
	}
}

func TestTRTConverter_Compatibility(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	conv, err := NewTRTConverter(TRTConverterConfig{
		EnginesDir: tmpDir,
		HFCacheDir: tmpDir,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("NewTRTConverter failed: %v", err)
	}

	tests := []struct {
		name       string
		cached     *TRTEngineMetadata
		current    *TRTEngineMetadata
		compatible bool
	}{
		{
			name: "same_versions",
			cached: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "89",
			},
			current: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "89",
			},
			compatible: true,
		},
		{
			name: "same_major_versions",
			cached: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "89",
			},
			current: &TRTEngineMetadata{
				CUDAVersion:  "12.6",
				TRTVersion:   "10.2.0",
				GPUSMVersion: "89",
			},
			compatible: true,
		},
		{
			name: "different_sm_version",
			cached: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "89", // Ada Lovelace (RTX 40xx)
			},
			current: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "86", // Ampere (RTX 30xx)
			},
			compatible: false,
		},
		{
			name: "different_cuda_major",
			cached: &TRTEngineMetadata{
				CUDAVersion:  "11.8",
				TRTVersion:   "8.6.1",
				GPUSMVersion: "89",
			},
			current: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "89",
			},
			compatible: false,
		},
		{
			name: "different_trt_major",
			cached: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "9.2.0",
				GPUSMVersion: "89",
			},
			current: &TRTEngineMetadata{
				CUDAVersion:  "12.4",
				TRTVersion:   "10.0.1",
				GPUSMVersion: "89",
			},
			compatible: false,
		},
		{
			name:       "nil_cached",
			cached:     nil,
			current:    &TRTEngineMetadata{},
			compatible: false,
		},
		{
			name:       "nil_current",
			cached:     &TRTEngineMetadata{},
			current:    nil,
			compatible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := conv.isCompatible(tt.cached, tt.current)
			if result != tt.compatible {
				t.Errorf("isCompatible() = %v, want %v", result, tt.compatible)
			}
		})
	}
}

func TestTRTConverter_ListEngines(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	conv, err := NewTRTConverter(TRTConverterConfig{
		EnginesDir: tmpDir,
		HFCacheDir: tmpDir,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("NewTRTConverter failed: %v", err)
	}

	// Create two engine dirs with metadata
	engines := []string{"engine-1", "engine-2"}
	for i, name := range engines {
		engineDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(engineDir, 0755); err != nil {
			t.Fatalf("create engine dir: %v", err)
		}
		meta := &TRTEngineMetadata{
			ModelID:      "model-" + name,
			CUDAVersion:  "12.4",
			MaxBatchSize: i + 1,
		}
		data, _ := json.MarshalIndent(meta, "", "  ")
		if err := os.WriteFile(filepath.Join(engineDir, "metadata.json"), data, 0644); err != nil {
			t.Fatalf("write metadata: %v", err)
		}
	}

	// List engines
	list, err := conv.ListEngines()
	if err != nil {
		t.Fatalf("ListEngines failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("ListEngines returned %d engines, want 2", len(list))
	}
}

func TestTRTConverter_DeleteEngine(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	conv, err := NewTRTConverter(TRTConverterConfig{
		EnginesDir: tmpDir,
		HFCacheDir: tmpDir,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("NewTRTConverter failed: %v", err)
	}

	// Create engine dir
	engineDir := filepath.Join(tmpDir, "delete-me")
	if err := os.MkdirAll(engineDir, 0755); err != nil {
		t.Fatalf("create engine dir: %v", err)
	}

	// Verify exists
	if _, err := os.Stat(engineDir); os.IsNotExist(err) {
		t.Fatal("engine dir should exist before delete")
	}

	// Delete
	if err := conv.DeleteEngine("delete-me"); err != nil {
		t.Fatalf("DeleteEngine failed: %v", err)
	}

	// Verify deleted
	if _, err := os.Stat(engineDir); !os.IsNotExist(err) {
		t.Error("engine dir should not exist after delete")
	}
}

func TestSanitizeAlias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"meta-llama/Llama-3.1-8B", "meta-llama-Llama-3.1-8B"},
		{"model:latest", "model-latest"},
		{"simple", "simple"},
		{"a/b:c/d", "a-b-c-d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeAlias(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeAlias(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

