package embedstore

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestBuildFileSplitsOnTokenLimit(t *testing.T) {
	dataset := Dataset{DatasetName: "d", DatasetVersion: "v", Items: []Item{{ID: "a", Content: "12345", Data: json.RawMessage(`{}`)}, {ID: "b", Content: "12345", Data: json.RawMessage(`{}`)}}}
	fake := &buildFake{vectors: [][]float32{{1, 0}, {0, 1}}}
	err := BuildFile(context.Background(), dataset, filepath.Join(t.TempDir(), "out.embed"), BuildOptions{Embedder: fake, BatchSize: 100, MaxBatchTokens: 3, Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.calls) != 2 || len(fake.calls[0]) != 1 || len(fake.calls[1]) != 1 {
		t.Fatalf("token batches %#v", fake.calls)
	}
}
