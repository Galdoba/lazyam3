package ingest

import (
	"os"
	"path/filepath"
	"testing"

	ingest "github.com/Galdoba/lazyam3/internal/domain/ingest"
)

func TestIngest_RealFiles(t *testing.T) {
	testDataDir := filepath.Join("..", "..", "..", "testdata", "examples", "AS_A_REINCARNATED_ARISTOCRAT_ILL_USE_MY_APPRAISA_S02E001_3S_AA_EPS_XX_0000130_20261015_224646_anylhbab")

	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skip("testdata not available")
	}

	ing := New("/usr/bin/ffprobe", nil)
	mu, err := ing.Ingest(testDataDir)
	if err != nil {
		t.Fatalf("Ingest() error: %v", err)
	}

	if mu.ID != 63506 {
		t.Errorf("ID = %d, want 63506", mu.ID)
	}

	if len(mu.Filenames) == 0 {
		t.Error("Expected at least one filename")
	}

	if len(mu.Streams) == 0 {
		t.Error("Expected at least one stream")
	}

	hasVideo := false
	for _, s := range mu.Streams {
		if s.Kind == ingest.StreamKindVideo {
			hasVideo = true
			break
		}
	}
	if !hasVideo {
		t.Error("Expected at least one video stream")
	}
}

func TestIngest_Synthetic(t *testing.T) {
	syntheticDir := filepath.Join("..", "..", "..", "testdata", "synthetic")
	_, err := os.Stat(syntheticDir)
	if os.IsNotExist(err) {
		t.Skip("synthetic testdata not available")
	}

	entries, err := os.ReadDir(syntheticDir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}

	ing := New("/usr/bin/ffprobe", nil)

	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		t.Run(dir.Name(), func(t *testing.T) {
			mu, err := ing.Ingest(filepath.Join(syntheticDir, dir.Name()))
			if err != nil {
				t.Fatalf("Ingest() error: %v", err)
			}

			hasVideo := false
			for _, s := range mu.Streams {
				if s.Kind == ingest.StreamKindVideo {
					hasVideo = true
					break
				}
			}
			if !hasVideo {
				t.Error("Expected at least one video stream")
			}

			if mu.ID == 0 {
				t.Error("Expected non-zero ID")
			}

			if len(mu.Filenames) == 0 {
				t.Error("Expected at least one filename")
			}
		})
	}
}
