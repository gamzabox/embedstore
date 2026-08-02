package embedstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BuildOptions configures a sequential embedding build.
type BuildOptions struct {
	Embedder       Embedder
	BatchSize      int
	MaxBatchTokens int
	Overwrite      bool
	IncludeContent bool
}

// BuildFile embeds a dataset in sequential batches, verifies the temporary file,
// then atomically publishes it. Existing output is never modified on failure.
func BuildFile(ctx context.Context, dataset Dataset, output string, options BuildOptions) error {
	if options.Embedder == nil {
		return fmt.Errorf("embedder is required")
	}
	if len(dataset.Items) == 0 {
		return fmt.Errorf("dataset has no items")
	}
	if _, err := os.Stat(output); err == nil && !options.Overwrite {
		return fmt.Errorf("output %q exists; enable overwrite", output)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	batch := options.BatchSize
	if batch == 0 {
		batch = 100
	}
	if batch < 1 {
		return fmt.Errorf("batch size must be positive")
	}
	include := options.IncludeContent
	items := append([]Item(nil), dataset.Items...)
	flat := []float32{}
	dimension := 0
	for start := 0; start < len(items); {
		end := start
		tokens := 0
		for end < len(items) && end-start < batch {
			estimated := estimateTokens(items[end].Content)
			if options.MaxBatchTokens > 0 && end > start && tokens+estimated > options.MaxBatchTokens {
				break
			}
			tokens += estimated
			end++
		}
		contents := make([]string, end-start)
		for i := start; i < end; i++ {
			contents[i-start] = items[i].Content
		}
		vectors, err := options.Embedder.Embed(ctx, contents)
		if err != nil {
			return err
		}
		if len(vectors) != len(contents) {
			return fmt.Errorf("embedder returned %d vectors for %d inputs", len(vectors), len(contents))
		}
		for _, v := range vectors {
			if dimension == 0 {
				dimension = len(v)
				if dimension == 0 {
					return fmt.Errorf("embedder returned zero-dimensional vector")
				}
			}
			if len(v) != dimension {
				return fmt.Errorf("%w: inconsistent embedding response", ErrDimensionMismatch)
			}
			flat = append(flat, v...)
		}
		start = end
	}
	if options.Embedder.Dimensions() > 0 && options.Embedder.Dimensions() != dimension {
		return fmt.Errorf("%w: embedder declares %d, returned %d", ErrDimensionMismatch, options.Embedder.Dimensions(), dimension)
	}
	if !include {
		for i := range items {
			items[i].Content = ""
		}
	}
	source, _ := json.Marshal(dataset)
	sum := sha256.Sum256(source)
	m := Manifest{FormatVersion: 1, DatasetName: dataset.DatasetName, DatasetVersion: dataset.DatasetVersion, Embedding: options.Embedder.Embedding(), Dimensions: dimension, VectorType: "float32", Normalized: true, ItemCount: len(items), CreatedAt: time.Now().UTC(), SourceChecksum: "sha256:" + hex.EncodeToString(sum[:]), ContentIncluded: include}
	dir := filepath.Dir(output)
	tmp, err := os.CreateTemp(dir, ".embedstore-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)
	if err = WriteFile(tmpPath, m, items, flat); err != nil {
		return err
	}
	if _, err = VerifyFile(tmpPath); err != nil {
		return err
	}
	return os.Rename(tmpPath, output)
}

func estimateTokens(content string) int { return len([]rune(content))/4 + 1 }
